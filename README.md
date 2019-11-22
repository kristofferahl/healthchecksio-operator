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

| Description                          | Environment variable                | Type   | Required |
|--------------------------------------|-------------------------------------|--------|----------|
| The healthchecks.io API Key          | HEALTHCHECKSIO_API_KEY              | string | true     |
| Run the operator in development mode | HEALTHCHECKSIO_OPERATOR_DEVELOPMENT | bool   | true     |

## Development

### Pre-requisites
- [Go](https://golang.org/) 1.13 or later
- [Kubebuilder](https://kubebuilder.io/) 2.1.0
- [Healthchecks.io](https://healthchecks.io/) account and an API key
- [Kubernetes](https://kubernetes.io/) cluster

### Getting started
```bash
export HEALTHCHECKSIO_API_KEY='<API_KEY>'
export HEALTHCHECKSIO_OPERATOR_DEVELOPMENT='true'
make install
make run
```

### Running tests
```bash
make test
```
