# Future Vision

KubeNAS aims to become a production-grade, Kubernetes-native storage platform for homelab, edge, and SMB environments.

## Long-Term Goals

- **Distributed Arrays**
  - support cross-node data placement and parity awareness
  - tolerate single-node failures in multi-node edge clusters

- **Kubernetes CSI Integration**
  - expose pooled storage as CSI-backed volumes
  - dynamic provisioning for workloads and app data

- **Advanced NVMe Tiering**
  - hot/cold data classification
  - predictive promotion/demotion policies

- **Object Storage Gateway**
  - S3-compatible API in front of pooled storage
  - policy-driven bucket placement and lifecycle

- **Snapshot Support**
  - point-in-time snapshots with retention policies
  - restore workflows integrated with operator status

- **Multi-Node Cluster Support**
  - cluster-aware scheduling and failover
  - topology-aware redundancy models

## Design Principles

- keep host-level data inspectable and recoverable
- favor declarative APIs and automation over imperative procedures
- integrate with GitOps, observability, and security best practices
