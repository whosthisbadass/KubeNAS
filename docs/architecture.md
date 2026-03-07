# KubeNAS Architecture

## 1) System Architecture

```text
+-------------------------------------------------------------+
|                    Kubernetes / OpenShift                   |
|                                                             |
|  +--------------------+      watches/reconciles             |
|  | KubeNAS Operator   | <--------------------------------+  |
|  |--------------------|                                  |  |
|  | DiskController     |                                  |  |
|  | ArrayController    |                                  |  |
|  | PoolController     |                                  |  |
|  | ShareController    |                                  |  |
|  | ParityController   |                                  |  |
|  | PlacementController|                                  |  |
|  | FailureController  |                                  |  |
|  | RebalanceController|                                  |  |
|  | CacheController    |                                  |  |
|  +---------+----------+                                  |  |
|            |                                             |  |
|            | gRPC/HTTP + K8s API                         |  |
|            v                                             |  |
|  +--------------------------+                            |  |
|  | Node Agent (DaemonSet)   | ---- SMART/disk ops ----+ |  |
|  +--------------------------+                         | |  |
+------------------------------------------------------+ |  |
                                                        |  |
Host filesystem:                                        |  |
  /mnt/disks/disk1 (xfs)                                |  |
  /mnt/disks/disk2 (xfs)                                |  |
  /mnt/disks/disk3 (xfs)                                |  |
                                                        |  |
mergerfs pool: /mnt/pool <------------------------------+  |
snapraid parity + content files                             |
SMB/NFS exports from pool paths                              |
+-------------------------------------------------------------+
```

## 2) Operator Architecture

The operator is built with Operator SDK (Go) and reconciles CRDs into host-level operations via the Node Agent.

### Controllers

- **DiskController**: validates disk metadata, mount intent, health conditions.
- **ArrayController**: groups data/parity disks; enforces array status transitions.
- **PoolController**: renders mergerfs policy and mount options.
- **ShareController**: materializes SMB/NFS exports from `Share` resources.
- **ParityController**: schedules and executes SnapRAID sync/scrub/check workflows.
- **PlacementController**: computes target disk selection and placement hints.
- **FailureController**: opens/remediates `DiskFailure` lifecycle.
- **RebalanceController**: orchestrates background file movement.
- **CacheController**: configures SSD/NVMe write/read cache behavior.

## 3) Node Agent Design

Node Agent runs privileged on the host and abstracts hardware operations:

- disk discovery and metadata collection (`lsblk`, `blkid`, udev)
- filesystem mount/unmount
- SMART checks and health scoring
- parity command execution (`snapraid sync/scrub/check`)
- spin-down and energy state controls
- metrics endpoint for Prometheus

### Agent API Responsibilities

- receive desired operation requests from operator
- execute idempotently with retries
- emit operation logs/events/status

## 4) Storage Layer Design

KubeNAS keeps each disk independently mountable and recoverable.

- Data disk layout:
  - `/mnt/disks/disk1`
  - `/mnt/disks/disk2`
  - `/mnt/disks/disk3`
- Pool mount:
  - mergerfs mounted at `/mnt/pool`
- Benefits:
  - mixed disk sizes
  - simpler single-disk recovery
  - no full-array stripe dependence

## 5) Disk Scheduling Engine

Placement strategies:

- `balanced`: weighted score for free space/load/health
- `least-used`: picks lowest utilized disk
- `fill-first`: fills one disk before rotating
- `tiered`: SSD/NVMe first, then HDD

Scoring function:

```text
score = (free_space_ratio * 0.7)
      + (inverse_disk_load * 0.2)
      + (health_score * 0.1)
```

Highest score receives new writes (subject to policy constraints and minimum free space).

## 6) Parity System

Parity is implemented with SnapRAID:

- parity disks can be dedicated and larger-or-equal to largest data disk
- scheduled sync/check/scrub via `ParitySchedule`
- parity drift metrics tracked by controller
- manual or automated parity refresh after rebalance and recovery

## 7) Failure Recovery Workflow

1. Node Agent detects SMART failure or disk read/write instability.
2. `Disk` status transitions to `Degraded`/`Failed`.
3. `FailureController` creates/updates `DiskFailure` CR.
4. Operator fences writes to impacted disk (policy-based).
5. Replacement disk onboarding via new `Disk` CR.
6. Rebuild/reconstruct + parity sync.
7. `DiskFailure` marked `Resolved` after validation.

## 8) Disk Rebalance Process

1. User or automation creates `RebalanceJob`.
2. `RebalanceController` plans file move batches.
3. Node Agent executes controlled copy + checksum verify.
4. Placement metadata updates and parity refresh triggered.
5. Job reports progress and completion metrics.

## 9) Cache Pool Architecture

`CachePool` introduces SSD/NVMe acceleration:

- write-back or write-through mode
- flush policies by age/size/pressure
- promotion/demotion hooks for tiered policy
- dirty data watermark protections

## 10) Network Share Architecture

`Share` resources describe protocol and path intents:

- SMB: ACL templates derived from Kubernetes users/groups, browse flags
- NFS: export CIDRs, squash policy, RO/RW controls, mapped group permissions
- path-level isolation under `/mnt/pool/<share-path>`
- controller validates target path, mount health, and subject permissions before publish
