# Install Nimbus

Install Nimbus operator using the official helm chart.

```shell
helm repo add nimbus https://anurag-rajawat.github.io/charts
helm repo update nimbus
helm upgrade --install nimbus-operator anurag-rajawat/nimbus -n nimbus --create-namespace
```

Install Nimbus using Helm charts locally (for testing)

```shell
cd deployments/nimbus/
helm upgrade --install nimbus-operator . -n nimbus --create-namespace
```

## Values

| Key              | Type   | Default               | Description                                            |
|------------------|--------|-----------------------|--------------------------------------------------------|
| image.repository | string | anurag-rajawat/nimbus | Image repository from which to pull the operator image |
| image.pullPolicy | string | IfNotPresent          | Operator image pull policy                             |
| image.tag        | string | v0.1                  | Operator image tag                                     |

## Uninstall the Operator

To uninstall, just run:

```bash
helm uninstall nimbus-operator -n nimbus
```
