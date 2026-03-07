# Repository Dependency Graph

This graph captures cross-component dependencies observed during the audit pass.

## High-Level Graph

```text
examples/*.yaml
   └─> CRDs in deploy/crds/*.yaml
         └─> operator/api/v1alpha1/*_types.go
               └─> operator/controllers/*.go
                     └─> operator/main.go

operator/controllers/kubernetes_agent_client.go
   └─> ConfigMaps (operator↔agent contract)
         └─> node-agent/cmd/kubenas-node-agent/main.go
               ├─> node-agent/pkg/disk
               ├─> node-agent/pkg/smart
               └─> node-agent/pkg/parity

storage/scheduler
storage/mergerfs
storage/snapraid
   └─> pure-Go utility modules (independent, unit-tested)

scripts/install.sh
   └─> deploy/scc/scc.yaml (OpenShift)
   └─> deploy/rbac/rbac.yaml
   └─> deploy/crds/*.yaml
   └─> deploy/operator.yaml
   └─> deploy/node-agent.yaml
```

## Go Module Graph

- `operator/go.mod`
  - depends on Kubernetes API machinery (`k8s.io/*`) and `controller-runtime`.
- `node-agent/go.mod`
  - depends on Kubernetes client-go + logging.
- `storage/go.mod`
  - local storage utility module.
- `storage/scheduler/go.mod`
  - isolated scheduling module for placement logic.

## Manifest Dependency Flow

1. Apply namespace/RBAC (`deploy/rbac/rbac.yaml`).
2. Apply SCCs on OpenShift (`deploy/scc/scc.yaml`).
3. Apply CRDs (`deploy/crds/*.yaml` or `deploy/crds.yaml`).
4. Deploy operator (`deploy/operator.yaml`).
5. Deploy node agent (`deploy/node-agent.yaml`).
6. Apply example custom resources from `examples/`.
