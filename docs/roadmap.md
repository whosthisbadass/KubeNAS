# Implementation Roadmap

## Phase 1 — Prototype

- node disk discovery and inventory (`Disk` status only)
- manual array creation (`Array`)
- mergerfs pool mount (`Pool`)
- SnapRAID parity bootstrap + manual sync
- basic SMB share publishing (`Share`)

## Phase 2 — Operator Automation

- full CRD set in `v1alpha1`
- controller reconciliation loops per resource domain
- placement scheduler integration (`PlacementPolicy`)
- failure signal ingestion from SMART + `DiskFailure` lifecycle
- automated parity schedules (`ParitySchedule`)

## Phase 3 — Advanced Features

- rebalance planner/executor (`RebalanceJob`)
- cache pools (`CachePool`) with flush orchestration
- metrics, alerts, and health dashboards
- optional web UI (React)

## Phase 4 — Enterprise Features

- CSI driver for PVC workflow integration
- multi-node and distributed array modes
- snapshot orchestration and policy engine
- S3-compatible object gateway on pooled storage
