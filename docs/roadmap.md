# Implementation Roadmap (Status-Aware)

This roadmap reflects the repository's current implementation state.

## Phase 1 — Prototype Foundation (**mostly complete**)

- ✅ Node disk discovery and inventory feeding `Disk.status`
- ✅ Core CRDs and controller implementations for Disk/Array/Pool/Share/Parity/Failure
- ✅ SnapRAID config rendering and parity CronJob scheduling
- ✅ mergerfs mount intent reconciliation
- ✅ OpenShift-oriented deployment assets (CRDs, RBAC, SCC, OLM manifests)

## Phase 2 — Controller Coverage + Safety (**in progress**)

- ⏳ Wire all implemented controllers in manager startup (notably `RebalanceReconciler`)
- ⏳ Decide and implement active reconciliation strategy for `PlacementPolicy` and `CachePool`
- ⏳ Add validating/mutating webhooks for critical storage invariants
- ⏳ Replace optimistic ConfigMap operation signaling with durable operation state tracking

## Phase 3 — Data Plane Hardening + Reliability (**not started**)

- ⏳ Harden SMB/NFS serving lifecycle and runtime ownership
- ⏳ Improve parity and pool mounted-state verification with deterministic feedback
- ⏳ Expand structured observability (metrics, events, alerting)
- ⏳ Build OKD SNO end-to-end conformance coverage

## Phase 4 — Advanced Features (**future**)

- `RebalanceJob` execution hardening and throughput controls
- `CachePool` orchestration and flush policies
- Optional operational UI

## Phase 5 — Platform Expansion (**future**)

- CSI integration for PVC-native workflows
- Multi-node and distributed modes
- Snapshot and policy engine
- Object gateway capabilities
