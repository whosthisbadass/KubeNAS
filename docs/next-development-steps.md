# KubeNAS Next Development Steps (Operator MVP on OKD SNO)

This document defines the **next concrete engineering steps** to move KubeNAS from prototype scaffolding to a working Kubernetes Operator for NAS workflows on **Single Node OpenShift (OKD)**.

## 1. Repository Assessment

### What already exists

- **Operator runtime scaffolding** using controller-runtime manager with health/readiness endpoints and registration for Disk, Array, Pool, Share, Parity, and Failure reconcilers.  
- **CRD API models (`v1alpha1`)** for the core resources (`Disk`, `Array`, `Pool`, `Share`) and extended resources (`ParitySchedule`, `PlacementPolicy`, `RebalanceJob`, `DiskFailure`, `CachePool`).  
- **Controller implementations** with initial reconciliation behavior:
  - Disk reconciliation through a `NodeAgentClient` abstraction (status, mount/unmount).  
  - Array reconciliation with SnapRAID config rendering and degraded-disk tracking.  
  - Pool reconciliation generating mergerfs mount intent.  
  - Share reconciliation generating SMB/NFS config ConfigMaps.  
  - Parity reconciliation creating SnapRAID CronJobs.  
  - Placement/failure/rebalance scaffolding and status transitions.  
- **Node agent prototype** with:
  - periodic discovery loop,
  - SMART loop,
  - ConfigMap-based operator↔agent communication.
- **Deployment artifacts** for CRDs, RBAC, operator deployment, node-agent DaemonSet example, SCCs, and OLM catalog/CSV stubs.
- **Architecture and roadmap docs** already in place.

### What is incomplete / high-risk gaps

- **No robust operation acknowledgement protocol** between operator and node-agent (currently async ConfigMap posting with optimistic success paths).
- **No controller wiring for all declared controller types** (Placement/Rebalance/Cache are not all registered in manager startup).
- **Share controller does not yet reconcile serving Pods/Services end-to-end** (currently mostly config generation and status updates).
- **Parity execution path split** (CronJob-based path exists; direct node-agent parity operations are not consistently integrated).
- **No persistent agent state CRD** (disk status is currently stored in ConfigMaps; no versioned intent/status schema for operations).
- **No validating/mutating admission layer** for strong safety checks (e.g., parity disk >= largest data disk, mountpoint path policy, immutable fields).
- **Insufficient e2e/integration testing** for OKD SNO + privileged node operations.
- **Node agent parity package has a stubbed file write path**, indicating non-production parity config application code.

### Architectural decisions that appear deliberate

- **ConfigMap-based agent communication channel** was chosen over direct RPC for better OpenShift network-policy fit and simplified bootstrap.
- **Independent per-disk filesystem model** (Unraid-style) with mergerfs union mount and SnapRAID parity intent.
- **Controller-per-domain model** with CRD-first API and status/conditions driving automation state.
- **OpenShift-first operational model** including SCC manifests and SNO scheduling tolerations.

### Missing critical components

- End-to-end **operation state machine** for agent operations (requested → running → succeeded/failed).
- **Finalized node-agent API contract** (request schema, idempotency keys, retries, timeout policy, error taxonomy).
- **Production share data-plane** manifests and lifecycle handling (Pod/Service/ConfigMap/Secret ownership + rollout/reload).
- **Recovery/rebuild orchestration** from `DiskFailure` into guided replacement + parity rebuild flows.
- **Observability stack** (Prometheus metrics with stable labels, alerting rules, structured events).

## 2. Immediate Next Development Milestones (MVP-first)

1. **Introduce AgentOperation CRD + controller (replace ad-hoc operation ConfigMaps)**  
   Why: gives durable operation state, retries, auditability, and clean reconciliation semantics.

2. **Harden DiskController with explicit mount/format workflow and safety gates**  
   Why: disk operations are destructive; must enforce role-aware and idempotent behavior with clear preconditions.

3. **Complete node-agent operation executor with ack/status updates**  
   Why: operator needs deterministic completion/error reporting for mount, mergerfs, snapraid, and share-related actions.

4. **Implement ArrayController parity validation + snapraid config ownership model**  
   Why: parity correctness is core NAS value; require strict checks (largest-disk constraints, content file placement, array state conditions).

5. **Implement PoolController mounted-state verification from agent feedback**  
   Why: mergerfs mount must be continuously verified; status cannot rely on optimistic assumptions.

6. **Implement ShareController data-plane resources (SMB first, NFS second)**  
   Why: MVP requires usable network share export; start with SMB on SNO due common homelab priority.

7. **ParityController integration test path (scheduled sync/check/scrub)**  
   Why: parity job reliability is core to resilience claims; needs tested CronJob behavior and status reflection.

8. **Add admission webhooks for invariant validation**  
   Why: prevent invalid specs from ever reaching reconciler (disk path policy, parity counts, share path constraints).

9. **Add conformance/e2e suite for OKD SNO**  
   Why: privileged host-path and SCC behavior differ from vanilla Kubernetes; test where it runs.

10. **OLM packaging hardening for installability**  
    Why: on OpenShift, operator adoption is OLM-driven; packaging should become a first-class release artifact.

## 3. Minimum Viable Prototype Definition

### MVP scope (first real working release)

- Disk discovery reflected in `Disk.status` with health + capacity.
- Array creation from selected data/parity disks.
- mergerfs pool creation and mount verification.
- SnapRAID parity config + scheduled sync/check/scrub execution.
- SMB share publication backed by pool path.

### MVP architecture

- **Operator** (single deployment) manages CRDs and desired state.
- **Node Agent DaemonSet** (privileged) performs host operations.
- **AgentOperation CRD** carries host-action intents and status.
- **ConfigMaps/Secrets** used only for generated runtime configs (e.g., snapraid.conf, smb.conf), not as primary operation transport.
- **Status-driven UX** through conditions on `Disk`, `Array`, `Pool`, `Share`, `ParitySchedule`.

### MVP success criteria

- New node disk appears and can become `Disk Ready`.
- Creating `Array` drives valid snapraid config generation.
- Creating `Pool` results in active mergerfs mount on host.
- Scheduled parity sync job executes and records status.
- SMB share is reachable and serves files from pool path.

## 4. Operator Development Plan (Controller Responsibilities)

### DiskController
- **Responsibility**: lifecycle of physical disks as CR-backed resources.
- **Reconciliation**: resolve live disk status, mount state, health score, phase transitions.
- **Dependencies**: NodeAgent operation/status channel, admission validation, disk health metrics.

### ArrayController
- **Responsibility**: map data/parity disks into SnapRAID-ready array config.
- **Reconciliation**: validate disk refs and roles, generate snapraid.conf, set degraded/rebuilding phases.
- **Dependencies**: Disk readiness, parity rules, node-agent config apply path.

### PoolController
- **Responsibility**: mergerfs union pool desired state.
- **Reconciliation**: compute branches from array data disks, apply/verify mount options, update utilization.
- **Dependencies**: Array readiness, agent mount execution, disk availability.

### ShareController
- **Responsibility**: SMB/NFS export lifecycle and policy mapping.
- **Reconciliation**: reconcile protocol config, serving pods/services, authz-derived access controls.
- **Dependencies**: Pool mounted state, secrets/config templates, network policy.

### ParityController
- **Responsibility**: recurring and on-demand parity jobs.
- **Reconciliation**: create/manage CronJobs or AgentOperation schedules, update parity run status.
- **Dependencies**: Array config readiness, node placement, SnapRAID runtime image/agent capability.

### PlacementController
- **Responsibility**: placement policy evaluation for new writes and rebalance planning.
- **Reconciliation**: compute target disk recommendations from free space, health, and strategy weights.
- **Dependencies**: up-to-date disk metrics and policy CRDs.

### FailureController
- **Responsibility**: detect, track, and close disk failure incidents.
- **Reconciliation**: open `DiskFailure` on degraded/failed states, drive resolution on recovery.
- **Dependencies**: Disk health signals, eventing/alerting, rebuild workflows.

### RebalanceController
- **Responsibility**: controlled movement of data for placement correction.
- **Reconciliation**: plan batches, execute moves via agent, checkpoint progress and completion.
- **Dependencies**: placement policy, pool topology, parity refresh trigger.

### CacheController
- **Responsibility**: manage SSD/NVMe cache pools and flush policies.
- **Reconciliation**: attach cache devices, enforce watermarks/dirty age, publish cache health.
- **Dependencies**: Pool lifecycle, cache mode semantics, future write path hooks.

## 5. Node Disk Agent Design (DaemonSet)

### Core responsibilities

- discover disks and metadata
- collect SMART metrics and derive health score
- format + mount disks idempotently
- apply mergerfs mount intents
- execute SnapRAID operations
- publish operation results + telemetry

### Communication model with operator

- Watch `AgentOperation` objects filtered by node label.
- Transition operation status phases (`Pending`, `Running`, `Succeeded`, `Failed`) with timestamps and reason codes.
- Update per-disk live state through either:
  - `Disk.status` patch API (preferred long-term), or
  - dedicated `NodeDiskStatus` CRD (good decoupling option).

### Security considerations

- privileged only where host block/mount operations require it.
- read-only host mounts wherever possible (`/proc`, `/sys`), write only to required paths (`/mnt`, `/etc/snapraid`).
- no host network unless protocol-serving sidecar requires it.
- signed images + digest pinning for operator and agent.
- strict RBAC: node-agent should not mutate broad cluster resources.

### OpenShift SCC requirements

- dedicated SCC for node-agent with minimal required capabilities (`SYS_ADMIN`, `MKNOD`, optionally `DAC_READ_SEARCH`).
- operator should remain on restricted SCC/non-root profile.
- avoid blanket `privileged` for operator workloads.

## 6. OpenShift / OKD Compatibility Plan

- **SCC strategy**: keep split SCC model (restricted operator + privileged node-agent) and trim capabilities to least privilege.
- **RBAC**: separate service accounts and roles; no wildcard verbs/resources for node-agent.
- **Non-root**: operator and share-serving containers should default non-root; privileged only for host-level data-plane actions.
- **CRI-O compatibility**: verify mount propagation, hostPath semantics, and seccomp defaults with CRI-O.
- **OLM packaging**: maintain CSV with install modes, CRD descriptors, RBAC, SCC guidance, upgrade strategy.
- **Disconnected support**: provide `ImageContentSourcePolicy`, mirrored image references, and catalog docs for air-gapped installs.

## 7. Repository Refactor Suggestions

Recommended target layout:

```text
kubenas/
 ├── operator/
 │   ├── api/
 │   ├── controllers/
 │   ├── internal/
 │   └── cmd/
 ├── node-agent/
 │   ├── cmd/
 │   ├── pkg/
 │   └── internal/
 ├── deploy/
 │   ├── crds/
 │   ├── rbac/
 │   ├── scc/
 │   ├── olm/
 │   └── examples/
 ├── docs/
 ├── examples/
 ├── test/
 │   ├── e2e/
 │   └── integration/
 └── hack/
```

Additional refactor priorities:

- move generated-vs-handwritten manifests into clear folders.
- add `internal/agentops` package shared by operator + agent request/response schema.
- add Makefile targets for e2e-on-kind and e2e-on-okd-sno.

## 8. Development Roadmap

### Phase 1 — Basic Operator (now)
- stabilize Disk/Array/Pool/Share/Parity MVP path.
- implement AgentOperation contract and idempotent node-agent executor.
- publish first end-to-end demo on OKD SNO.

### Phase 2 — Disk Scheduling
- complete PlacementPolicy integration and planner outputs.
- wire placement signals into new-write and rebalance decisions.

### Phase 3 — Failure Recovery
- automatic DiskFailure lifecycle plus replacement/rebuild workflow.
- parity-aware recovery runbooks + automation hooks.

### Phase 4 — Cache Pools
- implement CachePool controller and cache flush orchestration.
- integrate cache observability and policy controls.

### Phase 5 — CSI Driver
- expose pool-backed storage classes and PVC provisioning path.
- support RWX and topology semantics for NAS workloads.

### Phase 6 — Multi-Node Support
- node selection/affinity for arrays and pool branches across nodes.
- distributed coordination, fencing, and failure-domain aware policies.

## 9. Suggested Technologies

### Operator
- Go
- Operator SDK + controller-runtime
- Kubebuilder markers + webhook validation
- Envtest for controller unit/integration tests

### Node Agent
- Go (preferred for shared types/tooling)
- `udev` + `lsblk` integration
- smartmontools wrapper + normalized health scoring
- controlled command execution layer for mergerfs/snapraid

### Data plane / packaging
- UBI-based images for OpenShift friendliness
- OLM bundles (`bundle.Dockerfile`, CSV, channel metadata)
- Kustomize overlays for OKD SNO vs generic k8s

### Future UI
- React + PatternFly (OpenShift-native UX)
- backend via Kubernetes API (CRDs/status/events)

## 10. Recommended Next Pull Requests

### PR 1 — "Introduce AgentOperation API and reliable operator↔agent contract"
- **Description**: add `AgentOperation` CRD, status phases, and controller wiring; migrate existing ConfigMap op posting to CR-driven flow.
- **Files affected**:
  - `operator/api/v1alpha1/` (new types)
  - `operator/controllers/` (client + reconcilers)
  - `deploy/crds/`, `deploy/rbac/`, `deploy/olm/`
  - `node-agent/cmd/` (operation watcher)
- **Expected functionality**: deterministic ack/timeout/retry behavior for host actions.

### PR 2 — "MVP data-plane completion: mount + mergerfs + SMB publish"
- **Description**: complete Disk/Pool/Share reconciliation to produce actual mounted pool and reachable SMB service.
- **Files affected**:
  - `operator/controllers/disk_controller.go`
  - `operator/controllers/pool_controller.go`
  - `operator/controllers/share_controller.go`
  - `deploy/examples/` and `examples/`
- **Expected functionality**: reproducible disk->array->pool->SMB workflow on OKD SNO.

### PR 3 — "Parity reliability: SnapRAID scheduling and status reporting"
- **Description**: unify parity execution path (CronJob + agent execution feedback), add status and failure surfaces.
- **Files affected**:
  - `operator/controllers/parity_controller.go`
  - `node-agent/pkg/parity/`
  - `operator/api/v1alpha1/other_types.go`
  - `deploy/olm/clusterserviceversion.yaml`
- **Expected functionality**: observable, reliable parity sync/check/scrub jobs.

### PR 4 — "OpenShift hardening: SCC, RBAC, non-root, disconnected install docs"
- **Description**: tighten SCC capability set, split RBAC clearly, validate CRI-O assumptions, and document disconnected deployment steps.
- **Files affected**:
  - `deploy/scc/scc.yaml`
  - `deploy/rbac/rbac.yaml`
  - `deploy/olm/*`
  - `docs/`
- **Expected functionality**: secure-by-default install path for OKD SNO.

### PR 5 — "E2E test harness for OKD SNO MVP"
- **Description**: introduce automated e2e validating discovery, array, pool, parity schedule, and SMB share availability.
- **Files affected**:
  - `test/e2e/`
  - `Makefile`
  - `docs/`
- **Expected functionality**: repeatable validation gate for releases and PRs.

---

This plan prioritizes a **safe, deterministic MVP** while keeping the architecture aligned with long-term goals (placement, failure recovery, cache tiers, and eventual multi-node operation).
