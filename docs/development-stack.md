# Development Stack

## Core Languages and Frameworks

- **Operator**: Go + Operator SDK + controller-runtime
- **Node Agent**: Go (recommended) or Rust (alternative for performance-critical host ops)
- **CLI**: Go (cobra + client-go)
- **Future Web UI**: React + TypeScript

## Kubernetes Integration

- Kubernetes API (`client-go`, controller-runtime)
- CRDs with OpenAPI schemas
- RBAC, leader election, health/readiness probes

## Build & Delivery Tooling

- Go modules + `make`
- `golangci-lint`
- `ginkgo` / `envtest` for controller tests
- GitHub Actions for CI
- `kustomize` and Helm chart packaging

## GitOps & Operations

- Argo CD or FluxCD for deployment management
- Prometheus + Alertmanager + Grafana for observability
- Loki for log aggregation (optional)
