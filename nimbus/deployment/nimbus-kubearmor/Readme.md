# Install KubeArmor adapter

Install `nimbus-kubearmor` adapter using the official Helm chart.

```shell
helm repo add nimbus https://anurag-rajawat.github.io/charts
helm repo update nimbus
helm upgrade --dependency-update --install nimbus-kubearmor nimbus/nimbus-kubearmor -n nimbus
```

Install `nimbus-kubearmor` adapter using Helm charts locally (for testing)

```bash
cd deployments/nimbus-kubearmor/
helm upgrade --dependency-update --install nimbus-kubearmor . -n nimbus
```

## Values

| Key                           | Type   | Default                        | Description                                                                |
|-------------------------------|--------|--------------------------------|----------------------------------------------------------------------------|
| image.repository              | string | anuragrajawat/nimbus-kubearmor | Image repository from which to pull the `nimbus-kubearmor` adapter's image |
| image.pullPolicy              | string | IfNotPresent                   | `nimbus-kubearmor` adapter image pull policy                               |
| image.tag                     | string | v0.1                           | `nimbus-kubearmor` adapter image tag                                       |
| kubearmor-operator.autoDeploy | bool   | true                           | Auto deploy [KubeArmor](https://kubearmor.io/) with default configurations |

## Uninstall the KubeArmor adapter

To uninstall, just run:

```bash
helm uninstall nimbus-kubearmor -n nimbus
```
