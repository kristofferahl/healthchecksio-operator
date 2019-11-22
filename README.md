# healthchecksio-operator

A Kubernetes operator for [Healthchecks.io](https://healthchecks.io/), implemented in go using [kubebuilder](https://kubebuilder.io/).

## Status

![GitHub](https://img.shields.io/badge/status-alpha-blue?style=for-the-badge)
![GitHub](https://img.shields.io/github/license/kristofferahl/healthchecksio-operator?style=for-the-badge)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/kristofferahl/healthchecksio-operator?style=for-the-badge)

## Supported resources
- Check

## Example
```yaml
---
apiVersion: monitoring.healthchecks.io/v1alpha1
kind: Check
metadata:
  name: check-sample
spec:
  schedule: "*/10 * * * *"
  timezone: "Europe/Stockholm"
  gracePeriod: 120
  channels:
    - "email"
    - "webhook"
  tags:
    - healthchecksio-operator
    - prod
```

### Configuration

| Flag                   | Environment variable            | Type     | Required | Description                                                                                                           |
|------------------------|---------------------------------|----------|----------|-----------------------------------------------------------------------------------------------------------------------|
| -                      | HEALTHCHECKSIO_API_KEY          | string   | true     | The healthchecks.io API Key.                                                                                          |
| metrics-addr           | OPERATOR_METRICS_ADDR           | string   | false    | The address the metric endpoint binds to.                                                                             |
| enable-leader-election | OPERATOR_ENABLE_LEADER_ELECTION | bool     | false    | Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager. |
| development            | OPERATOR_DEVELOPMENT            | bool     | false    | Run the operator in development mode.                                                                                 |
| log-level              | OPERATOR_LOG_LEVEL              | string   | false    | The log level used by the operator.                                                                                   |
| name-prefix            | OPERATOR_NAME_PREFIX            | string   | false    | Prefix used to create unique resources across clusters.                                                               |
| reconcile-interval     | OPERATOR_RECONCILE_INTERVAL     | duration | false    | The interval for the reconcile loop.                                                                                  |


## Development

### Pre-requisites
- [Go](https://golang.org/) 1.13 or later
- [Kubebuilder](https://kubebuilder.io/) 2.1.0
- [Healthchecks.io](https://healthchecks.io/) account and an API key
- [Kubernetes](https://kubernetes.io/) cluster

### Getting started
```bash
export HEALTHCHECKSIO_API_KEY='<API_KEY>'
export OPERATOR_DEVELOPMENT='true'
make install
make run
```

### Running tests
```bash
make test
```
