# Repository Layout

```text
kubenas/
 ├── README.md
 ├── LICENSE
 ├── CONTRIBUTING.md
 ├── CODE_OF_CONDUCT.md
 ├── SECURITY.md
 ├── docs/
 │    ├── architecture.md
 │    ├── storage.md
 │    ├── crds.md
 │    ├── scheduling.md
 │    ├── roadmap.md
 │    ├── mvp-plan.md
 │    ├── future-vision.md
 │    └── repository-layout.md
 ├── operator/
 │    ├── api/
 │    │    └── v1alpha1/
 │    ├── controllers/
 │    ├── config/
 │    │    ├── crd/
 │    │    ├── manager/
 │    │    └── rbac/
 │    └── main.go
 ├── node-agent/
 │    ├── cmd/
 │    │    └── kubenas-node-agent/
 │    ├── disk/
 │    ├── smart/
 │    └── internal/
 ├── deploy/
 │    ├── crds/
 │    ├── operator.yaml
 │    └── examples/
 ├── charts/
 │    └── kubenas/
 ├── scripts/
 │    ├── install.sh
 │    ├── dev-up.sh
 │    └── lint.sh
 ├── examples/
 │    ├── disks.yaml
 │    ├── array.yaml
 │    ├── pool.yaml
 │    ├── parity-schedule.yaml
 │    ├── shares.yaml
 │    └── rebalance-job.yaml
 └── .github/
      ├── workflows/
      │    ├── ci.yml
      │    └── release.yml
      └── ISSUE_TEMPLATE/
           ├── bug_report.md
           └── feature_request.md
```

## Directory Responsibilities

- `docs/`: architecture, operations, CRD reference, roadmap.
- `operator/`: Kubernetes operator APIs/controllers and manager runtime.
- `node-agent/`: host-level disk and health operations.
- `deploy/`: deploy-time manifests and generated bundles.
- `charts/`: Helm packaging for easy install.
- `examples/`: sample CRs for common NAS scenarios.
- `scripts/`: local development and validation scripts.
