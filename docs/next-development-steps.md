# KubeNAS Next Development Steps (Current State + Priorities)

This document captures what is implemented in the repository today and what still needs to be completed to reach a production-ready Operator + Node Agent.

## Current implementation snapshot

### Implemented and wired in `operator/main.go`

The manager currently starts and registers these reconcilers:

- `DiskReconciler`
- `ArrayReconciler`
- `PoolReconciler`
- `ShareReconciler`
- `ParityReconciler`
- `FailureReconciler`

Health/readiness probes are also enabled.

### Implemented in codebase, but **not wired** in `operator/main.go`

- `RebalanceReconciler` exists but is not registered with the manager.
- Placement scheduler logic exists (`PlacementScheduler`) but there is no active PlacementPolicy reconciler registration.
- `CachePool` CRD exists, but no active cache controller is registered.

### What works in the current MVP prototype

- **Disk lifecycle loop**: Disk status is driven from node-agent-reported data (capacity, SMART summary, health score, mount state), with mount/unmount operations requested through the NodeAgent client abstraction.
- **Array loop**: Array reconciler resolves disk refs, tracks degraded disks, builds SnapRAID config, and publishes it through ConfigMap-mediated agent integration.
- **Pool loop**: Pool reconciler resolves mounted, ready data disks and asks the node agent to ensure a mergerfs mount.
- **Share loop (resource level)**: Share reconciler creates/updates protocol ConfigMaps and Service objects for SMB/NFS, and updates Share status.
- **Parity scheduling**: Parity reconciler creates/updates Kubernetes CronJobs for sync/check/scrub.
- **Failure detection baseline**: Failure reconciler opens/resolves `DiskFailure` resources based on `Disk.status.phase`.
- **Node agent baseline loops**: device discovery, SMART checks, and processing of operator operation requests via ConfigMaps.

## Known gaps and limitations

### Control plane / reconciliation gaps

1. **Operation channel is still ConfigMap-only and optimistic**
   - Operator posts operation request ConfigMaps and assumes success in several flows (for example mergerfs mount returns optimistic `true` after request post).
   - There is no durable request/acknowledgement lifecycle with retries and terminal states.

2. **Controller wiring mismatch against available APIs**
   - `RebalanceJob`, `PlacementPolicy`, and `CachePool` APIs exist but are not all active in manager startup.

3. **Parity safety validation is incomplete**
   - No strong invariant enforcement (for example parity disk size >= largest data disk) in admission or strict reconciliation checks.

4. **No admission webhook validation layer yet**
   - Invalid specs can still be submitted and only fail later in reconciliation.

5. **Share data plane is still lightweight**
   - Current implementation focuses on ConfigMaps/Service publication and status updates.
   - Full managed SMB/NFS serving workload lifecycle hardening (runtime pods, health checks, rolling reload behavior, secret handling) still needs production hardening.

### Node agent and execution-path gaps

1. **No durable operation object model**
   - No `AgentOperation`-style CRD exists for request IDs, phase transitions, and audit history.

2. **Limited execution guarantees**
   - Retries/backoff/idempotency behavior for operation processing are not yet formalized as a robust contract.

3. **Host-operation hardening remains pending**
   - Real-world error taxonomy, timeout handling, and stronger cleanup semantics need to be standardized.

### Quality and release-readiness gaps

1. **No comprehensive E2E suite for OKD SNO privileged flows**.
2. **Limited observability** (metrics/events/alerts are not yet comprehensive).
3. **OLM packaging exists, but install/upgrade test coverage is still light**.

## Priority implementation plan

## P0 — correctness and safety

1. **Introduce durable operation lifecycle (AgentOperation CRD + controller + agent integration)**
   - Replace ad-hoc operation ConfigMaps with request/status objects.
   - Define phases: `Pending`, `Running`, `Succeeded`, `Failed`, `TimedOut`.
   - Add idempotency key and retry policy.

2. **Register missing controllers in manager**
   - Wire `RebalanceReconciler` immediately.
   - Either implement/register Placement and Cache controllers or explicitly gate those APIs as experimental.

3. **Add admission webhooks for invariants**
   - Disk mount path policies.
   - Array parity/data validation (including size constraints).
   - Share path/protocol validation and immutable field rules.

## P1 — data plane hardening

4. **Harden share serving lifecycle**
   - Ensure controlled ownership/reconciliation for serving workloads and reload behavior.
   - Standardize SMB-first production path; keep NFS as secondary until similarly hardened.

5. **Parity execution hardening**
   - Ensure SnapRAID config ownership and propagation are deterministic.
   - Add integration checks for scheduled sync/check/scrub status reflection.

6. **Pool mount verification improvements**
   - Move from optimistic mount assumptions to verified mounted-state feedback loops.

## P2 — production readiness

7. **Observability baseline**
   - Export controller + node-agent metrics.
   - Emit consistent events/reasons and alertable conditions.

8. **End-to-end test matrix for SNO/OKD**
   - Disk discovery, array creation, pool mount, share publish, parity schedules, and failure resolution flows.

9. **OLM install/upgrade hardening**
   - Validate clean install, upgrade path, and CRD conversion/compatibility strategy.

## MVP completion criteria (updated)

The project should be considered a stable MVP when all of the following are true:

- Disk/Array/Pool/Share/Parity/Failure reconcilers are functioning on OKD SNO under repeated reconcile cycles.
- Missing declared APIs are either fully wired (`Rebalance`, `Placement`, `Cache`) or explicitly documented as unsupported.
- Operation execution path is durable and auditable (no optimistic-only success assumptions).
- Admission prevents known-invalid storage specs from entering reconciliation.
- An automated E2E suite validates privileged host workflows on every release candidate.
