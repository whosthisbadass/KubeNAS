# CRD Reference

This document describes KubeNAS custom resources and provides example manifests.

> API group used below: `storage.kubenas.io/v1alpha1`

---

## 1) Disk

Represents a host block device and mount intent.

### Spec highlights

- `nodeName`: node containing the disk
- `devicePath`: e.g. `/dev/sdb`
- `filesystem`: `xfs` recommended
- `mountPoint`: e.g. `/mnt/disks/disk1`
- `role`: `data`, `parity`, or `cache`

```yaml
apiVersion: storage.kubenas.io/v1alpha1
kind: Disk
metadata:
  name: disk1
spec:
  nodeName: snode-1
  devicePath: /dev/sdb
  filesystem: xfs
  mountPoint: /mnt/disks/disk1
  role: data
```

---

## 2) Array

Defines a NAS array composed of data and parity disks.

### Spec highlights

- `dataDisks`: list of `Disk` references
- `parityDisks`: list of parity disk references
- `snapraidConfig`: tuning for SnapRAID behavior

```yaml
apiVersion: storage.kubenas.io/v1alpha1
kind: Array
metadata:
  name: media-array
spec:
  dataDisks:
    - disk1
    - disk2
    - disk3
  parityDisks:
    - parity1
  snapraidConfig:
    contentFiles:
      - /mnt/disks/disk1/.snapraid.content
      - /mnt/disks/disk2/.snapraid.content
    excludePatterns:
      - '*.tmp'
```

---

## 3) Pool

Creates mergerfs pool from an `Array`.

### Spec highlights

- `arrayRef`: backing array
- `mountPoint`: pooled path
- `mergerfs.categoryCreate`: file create policy

```yaml
apiVersion: storage.kubenas.io/v1alpha1
kind: Pool
metadata:
  name: main-pool
spec:
  arrayRef: media-array
  mountPoint: /mnt/pool
  mergerfs:
    categoryCreate: epmfs
    minFreeSpace: 50Gi
```

---

## 4) Share

Declares SMB/NFS export for pooled directories.

### Spec highlights

- `poolRef`: pool reference
- `path`: subpath under pool
- `protocol`: `SMB` or `NFS`
- protocol-specific config sections

```yaml
apiVersion: storage.kubenas.io/v1alpha1
kind: Share
metadata:
  name: media-smb
spec:
  poolRef: main-pool
  path: /media
  protocol: SMB
  smb:
    browseable: true
    readOnly: false
    validUsers:
      - mediauser
```

---

## 5) CachePool

Defines SSD/NVMe cache tier for selected pools.

### Spec highlights

- `poolRef`: target pool
- `cacheDisks`: list of cache disk refs
- `mode`: `write-back` or `write-through`
- `flushPolicy`: age/size thresholds

```yaml
apiVersion: storage.kubenas.io/v1alpha1
kind: CachePool
metadata:
  name: fast-cache
spec:
  poolRef: main-pool
  cacheDisks:
    - nvme1
  mode: write-back
  flushPolicy:
    maxDirtyAge: 30m
    highWatermarkPercent: 80
```

---

## 6) PlacementPolicy

Controls how files are placed across disks.

### Spec highlights

- `strategy`: `balanced`, `least-used`, `fill-first`, `tiered`
- `weights`: free space/load/health weighting
- `minFreeSpace`: reserve per disk

```yaml
apiVersion: storage.kubenas.io/v1alpha1
kind: PlacementPolicy
metadata:
  name: default-placement
spec:
  strategy: balanced
  weights:
    freeSpace: 0.7
    load: 0.2
    health: 0.1
  minFreeSpace: 20Gi
```

---

## 7) ParitySchedule

Schedules parity maintenance actions.

### Spec highlights

- `arrayRef`: target array
- `syncCron`, `checkCron`, `scrubCron`
- optional retention/history settings

```yaml
apiVersion: storage.kubenas.io/v1alpha1
kind: ParitySchedule
metadata:
  name: weekly-parity
spec:
  arrayRef: media-array
  syncCron: "0 2 * * *"
  checkCron: "0 3 * * 0"
  scrubCron: "0 4 1 * *"
```

---

## 8) RebalanceJob

Executes rebalancing of data to meet policy goals.

### Spec highlights

- `poolRef`: pool to rebalance
- `placementPolicyRef`: policy to enforce
- `maxConcurrentMoves`: throttling
- `dryRun`: planning mode

```yaml
apiVersion: storage.kubenas.io/v1alpha1
kind: RebalanceJob
metadata:
  name: rebalance-2026-01
spec:
  poolRef: main-pool
  placementPolicyRef: default-placement
  maxConcurrentMoves: 2
  dryRun: false
```

---

## 9) DiskFailure

Tracks detection and remediation of failed disks.

### Spec highlights

- `diskRef`: failed disk
- `severity`: `warning`, `critical`
- `recommendedAction`: generated guidance
- `replacementDiskRef`: optional after replacement

```yaml
apiVersion: storage.kubenas.io/v1alpha1
kind: DiskFailure
metadata:
  name: disk2-failure-001
spec:
  diskRef: disk2
  severity: critical
  reason: SMART_PREFAIL
  recommendedAction: Replace disk and start rebuild
```
