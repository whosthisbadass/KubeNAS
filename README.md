# KubeNAS

[![CI](https://github.com/whosthisbadass/KubeNAS/actions/workflows/ci.yml/badge.svg)](https://github.com/whosthisbadass/KubeNAS/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/whosthisbadass/KubeNAS)](https://goreportcard.com/report/github.com/whosthisbadass/KubeNAS)
[![License](https://img.shields.io/github/license/whosthisbadass/KubeNAS)](./LICENSE)
[![Release](https://img.shields.io/github/v/release/whosthisbadass/KubeNAS)](https://github.com/whosthisbadass/KubeNAS/releases)

KubeNAS is a **Kubernetes-native NAS platform** for homelab and edge environments, built for **Single Node OpenShift (OKD)** and Kubernetes clusters.

It brings Unraid-style storage concepts to a declarative, operator-driven platform:

- mixed disk-size arrays
- parity protection with SnapRAID
- union filesystem pooling with mergerfs
- SMB/NFS network shares with Kubernetes user/group RBAC mapping
- SSD/NVMe cache tiers
- GitOps-friendly, CRD-driven infrastructure

---

## Why KubeNAS?

Traditional NAS systems are often appliance-centric and manually administered. KubeNAS applies cloud-native patterns:

- **Declarative state via CRDs** instead of UI-only configuration
- **Controller-based reconciliation** for storage automation
- **GitOps workflows** for versioned storage changes
- **Kubernetes observability** (events, conditions, metrics)
- **Automated failure handling** in edge and homelab clusters

---

## Features

### Core NAS Capabilities

- Unraid-style disk arrays with independent per-disk filesystems (typically XFS)
- Mixed-capacity disk support
- SnapRAID parity validation and sync scheduling
- mergerfs storage pooling
- SMB/NFS share publishing with native OKD/Kubernetes user/group authorization

### Kubernetes-Native Automation

- Operator-managed lifecycle for disks, pools, shares, parity, and cache
- CRDs for full declarative configuration
- Automatic disk discovery and health reporting
- Placement policies (`balanced`, `least-used`, `fill-first`, `tiered`)
- Rebalance jobs for optimized data placement
- Disk failure detection and recovery workflows

### Edge-Friendly Operations

- Single Node OpenShift/OKD optimized
- Node Agent DaemonSet for hardware integration and SMART telemetry
- Gradual future path to multi-node support

---

## Architecture Summary

KubeNAS consists of:

1. **KubeNAS Operator**
   - Reconciles CRDs: `Disk`, `Array`, `Pool`, `Share`, `CachePool`, `PlacementPolicy`, `ParitySchedule`, `RebalanceJob`, `DiskFailure`
   - Drives SnapRAID/mergerfs/share lifecycle
2. **Node Agent (DaemonSet)**
   - Discovers disks, mounts filesystems, reads SMART data, reports health
   - Performs host-level actions for parity/check/scrub/spin-down
3. **Storage Data Plane**
   - Individual mounted disks under `/mnt/disks/*`
   - mergerfs pooled mount at `/mnt/pool`
   - SnapRAID parity disks + content files

See [Architecture Documentation](./docs/architecture.md) for full workflows.

---

## Comparison: KubeNAS vs Traditional NAS

| Capability | Traditional NAS | KubeNAS |
|---|---|---|
| Configuration model | UI/manual | GitOps + CRDs |
| Automation | Limited/imperative | Controller reconciliation |
| Storage pooling | Vendor specific | mergerfs + Kubernetes resources |
| Parity | Vendor-specific engines | SnapRAID-based parity scheduling |
| Failure handling | Appliance logic | Kubernetes events + DiskFailure CRD |
| Platform integration | Standalone | Native on OKD/Kubernetes |

---

## Quick Start (Prototype)

### Prerequisites

- Single Node OpenShift / OKD cluster (or Kubernetes for local development)
- `oc` or `kubectl`
- `helm` (optional)
- Host packages on node: `snapraid`, `mergerfs`, `smartmontools`, `samba`/`nfs-utils`

### 1) Install CRDs and Operator

```bash
kubectl apply -f deploy/crds.yaml
kubectl apply -f deploy/rbac/rbac.yaml
kubectl apply -f deploy/scc/scc.yaml
kubectl apply -f deploy/operator.yaml
kubectl apply -f deploy/node-agent.yaml
```

### 2) Register Disks and Array

```bash
kubectl apply -f examples/disks.yaml
kubectl apply -f examples/array.yaml
```

### 3) Configure Pool, Parity, and Shares

```bash
kubectl apply -f examples/pool.yaml
kubectl apply -f examples/parity-schedule.yaml
kubectl apply -f examples/shares.yaml
```

### 4) Validate

```bash
kubectl get disks,arrays,pools,shares,parityschedules
kubectl describe array media-array
```

---

## Screenshots

> Placeholders for future UI and operational screenshots.

- `docs/images/dashboard-overview.png`
- `docs/images/pool-health.png`
- `docs/images/rebalance-job.png`

---

## Documentation

- [Architecture](./docs/architecture.md)
- [Storage Model](./docs/storage.md)
- [CRD Reference + Examples](./docs/crds.md)
- [Scheduling Engine](./docs/scheduling.md)
- [Repository Layout](./docs/repository-layout.md)
- [Dependency Graph](./docs/dependency-graph.md)
- [Development Roadmap](./docs/roadmap.md)
- [MVP Implementation Plan](./docs/mvp-plan.md)
- [Next Development Steps](./docs/next-development-steps.md)
- [Development Stack](./docs/development-stack.md)
- [Future Vision](./docs/future-vision.md)

---

## Community

- Issues: <https://github.com/whosthisbadass/KubeNAS/issues>
- Discussions: <https://github.com/whosthisbadass/KubeNAS/discussions>
- Security reports: see [SECURITY.md](./SECURITY.md)
- Contributing guide: [CONTRIBUTING.md](./CONTRIBUTING.md)

---

## Project Status

KubeNAS currently includes a **working MVP prototype** with CRDs, core reconcilers (Disk/Array/Pool/Share/Parity/Failure), a ConfigMap-driven node-agent integration path, OpenShift deployment manifests, and OLM bundle assets.

Current priorities are controller coverage completion (including APIs already defined in CRDs), operation-lifecycle durability, stronger admission validation, and OKD SNO end-to-end hardening before production-readiness claims.
