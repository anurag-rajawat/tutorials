use log::error;
use proxy_wasm::traits::{Context, HttpContext, RootContext};
use proxy_wasm::types::{Action, ContextType, LogLevel};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::time::{Duration, UNIX_EPOCH};

/// Metadata about an API event, including context ID, timestamp, and information about the traffic source and node.
#[derive(Serialize, Default)]
struct Metadata {
    context_id: u32,
    timestamp: u64,

    /// Name of the traffic source being used (e.g., Istio).
    traffic_source_name: String,

    /// Version of the traffic source being used (e.g., 1.24).
    traffic_source_version: String,

    /// The name of the Kubernetes node where the workload is running. If the workload
    /// is not running in a Kubernetes environment, this field is empty.
    node_name: String,
}

/// Represents an incoming HTTP request, including headers and body.
#[derive(Serialize, Default)]
struct Reqquest {
    headers: HashMap<String, String>,
    body: String,
}

/// Represents an outgoing HTTP response, including headers and body.
#[derive(Serialize, Default)]
struct Ressponse {
    headers: HashMap<String, String>,
    body: String,
}

/// Represents a generic workload, which can be a Kubernetes or non-Kubernetes resource.
/// It serves as a source or destination for access within a system.
#[derive(Serialize, Default)]
struct Workload {
    /// Name of the workload.
    name: String,

    /// The namespace in which the workload is deployed. This field is only applicable
    /// for Kubernetes workloads.
    namespace: String,

    /// IP address of the workload.
    ip: String,

    /// Port number used by the workload.
    port: u16,
}

/// Represents an API event, encapsulating metadata, request, response, source, and destination workloads.
#[derive(Serialize, Default)]
struct APIEvent {
    /// Metadata about the API event.
    metadata: Metadata,

    /// Incoming HTTP request.
    request: Reqquest,

    /// Outgoing HTTP response.
    response: Ressponse,

    /// Source workload of the API call.
    source: Workload,

    /// Destination workload of the API call.
    destination: Workload,

    protocol: String,
}

/// Configuration for the plugin.
#[derive(Deserialize, Clone, Default)]
struct PluginConfig {
    /// Name of the upstream service.
    upstream_name: String,

    /// Path to the upstream service.
    path: String,

    /// Authority of the upstream service.
    authority: String,
}

/// Represents a plugin instance, holding configuration and API event information.
#[derive(Default)]
struct Plugin {
    /// Context ID for the plugin instance.
    _context_id: u32,

    /// Configuration for the plugin.
    config: PluginConfig,

    /// API event being processed by the plugin.
    api_event: APIEvent,
}

/// Maximum allowed size for request and response bodies.
const MAX_BODY_SIZE: usize = 1_000_000; // 1MB

impl Context for Plugin {
    fn on_done(&mut self) -> bool {
        dispatch_http_call_to_upstream(self);
        true
    }
}

fn dispatch_http_call_to_upstream(obj: &mut Plugin) {
    update_metadata(obj);
    let telemetry_json = serde_json::to_string(&obj.api_event).unwrap_or_default();

    let headers = vec![
        (":method", "POST"),
        (":authority", &obj.config.authority),
        (":path", &obj.config.path),
        ("accept", "*/*"),
        ("Content-Type", "application/json"),
    ];

    let http_call_res = obj.dispatch_http_call(
        &obj.config.upstream_name,
        headers,
        Some(telemetry_json.as_bytes()),
        vec![],
        Duration::from_secs(1),
    );

    if http_call_res.is_err() {
        error!(
            "failed to dispatch HTTP call, to '{}' status: {http_call_res:#?}",
            &obj.config.upstream_name,
        );
    }
}

fn update_metadata(obj: &mut Plugin) {
    obj.api_event.metadata.node_name = String::from_utf8(
        obj.get_property(vec!["node", "metadata", "NODE_NAME"])
            .unwrap_or_default(),
    )
    .unwrap_or_default();
    obj.api_event.metadata.traffic_source_name = "Envoy".to_string();
}

impl RootContext for Plugin {
    fn on_configure(&mut self, _plugin_configuration_size: usize) -> bool {
        if let Some(config_bytes) = self.get_plugin_configuration() {
            if let Ok(config) = serde_json::from_slice::<PluginConfig>(&config_bytes) {
                self.config = config;
            } else {
                error!(
                    "failed to parse plugin config: {}",
                    String::from_utf8_lossy(&config_bytes)
                );
            }
        } else {
            error!("no plugin config found");
        }
        true
    }

    fn create_http_context(&self, _context_id: u32) -> Option<Box<dyn HttpContext>> {
        Some(Box::new(Plugin {
            _context_id,
            config: self.config.clone(),
            api_event: Default::default(),
        }))
    }

    fn get_type(&self) -> Option<ContextType> {
        Some(ContextType::HttpContext)
    }
}

impl HttpContext for Plugin {
    fn on_http_request_headers(&mut self, _num_headers: usize, _end_of_stream: bool) -> Action {
        let (src_ip, src_port) = get_url_and_port(
            String::from_utf8(
                self.get_property(vec!["source", "address"])
                    .unwrap_or_default(),
            )
            .unwrap_or_default(),
        );

        let req_headers = self.get_http_request_headers();
        let mut headers: HashMap<String, String> = HashMap::with_capacity(req_headers.len());
        for header in req_headers {
            // Don't include Envoy's pseudo headers
            // https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_conn_man/headers#id13
            if !header.0.starts_with("x-envoy") {
                headers.insert(header.0, header.1);
            }
        }

        self.api_event.metadata.timestamp = self
            .get_current_time()
            .duration_since(UNIX_EPOCH)
            .unwrap_or_default()
            .as_secs();
        self.api_event.metadata.context_id = self._context_id;
        self.api_event.request.headers = headers;

        let protocol = String::from_utf8(
            self.get_property(vec!["request", "protocol"])
                .unwrap_or_default(),
        )
        .unwrap_or_default();
        self.api_event.protocol = protocol;

        self.api_event.source.ip = src_ip;
        self.api_event.source.port = src_port;
        self.api_event.source.name = String::from_utf8(
            self.get_property(vec!["node", "metadata", "NAME"])
                .unwrap_or_default(),
        )
        .unwrap_or_default();
        self.api_event.source.namespace = String::from_utf8(
            self.get_property(vec!["node", "metadata", "NAMESPACE"])
                .unwrap_or_default(),
        )
        .unwrap_or_default();

        Action::Continue
    }

    fn on_http_request_body(&mut self, _body_size: usize, _end_of_stream: bool) -> Action {
        let body = String::from_utf8(
            self.get_http_request_body(0, _body_size)
                .unwrap_or_default(),
        )
        .unwrap_or_default();

        if !body.is_empty() && body.len() <= MAX_BODY_SIZE {
            self.api_event.request.body = body;
        }
        Action::Continue
    }

    fn on_http_response_headers(&mut self, _num_headers: usize, _end_of_stream: bool) -> Action {
        let (dest_ip, dest_port) = get_url_and_port(
            String::from_utf8(
                self.get_property(vec!["destination", "address"])
                    .unwrap_or_default(),
            )
            .unwrap_or_default(),
        );

        let res_headers = self.get_http_response_headers();
        let mut headers: HashMap<String, String> = HashMap::with_capacity(res_headers.len());
        for res_header in res_headers {
            // Don't include Envoy's pseudo headers
            // https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_conn_man/headers#id13
            if !res_header.0.starts_with("x-envoy") {
                headers.insert(res_header.0, res_header.1);
            }
        }

        self.api_event.response.headers = headers;
        self.api_event.destination.ip = dest_ip;
        self.api_event.destination.port = dest_port;

        Action::Continue
    }

    fn on_http_response_body(&mut self, _body_size: usize, _end_of_stream: bool) -> Action {
        let body = String::from_utf8(
            self.get_http_response_body(0, _body_size)
                .unwrap_or_default(),
        )
        .unwrap_or_default();
        if !body.is_empty() && body.len() <= MAX_BODY_SIZE {
            self.api_event.response.body = body;
        }
        Action::Continue
    }
}

fn get_url_and_port(address: String) -> (String, u16) {
    let parts: Vec<&str> = address.split(':').collect();

    let mut url = "".to_string();
    let mut port = 0;

    if parts.len() == 2 {
        url = parts[0].parse().unwrap_or_default();
        port = parts[1].parse().unwrap_or_default();
    } else {
        error!("invalid address");
    }

    (url, port)
}

/// This is the entry point for the Wasm module and is part of ABI specification.
fn _start() {
    proxy_wasm::main! {{
        // Set the Log level to `warning`
        proxy_wasm::set_log_level(LogLevel::Warn);

        // Set the root context of the filter to a new instance of the Plugin type with its default value.
        proxy_wasm::set_root_context(|_| -> Box<dyn RootContext> {Box::new(Plugin::default())});
    }}
}
