# MVP Implementation Plan

This plan targets a first working KubeNAS prototype on Single Node OpenShift.

## 1) Scaffold Operator

```bash
mkdir -p operator && cd operator
operator-sdk init --domain kubenas.io --repo github.com/kubenas/kubenas/operator
operator-sdk create api --group storage --version v1alpha1 --kind Disk --resource --controller
operator-sdk create api --group storage --version v1alpha1 --kind Array --resource --controller
operator-sdk create api --group storage --version v1alpha1 --kind Pool --resource --controller
operator-sdk create api --group storage --version v1alpha1 --kind Share --resource --controller
operator-sdk create api --group storage --version v1alpha1 --kind ParitySchedule --resource --controller
```

## 2) Define Initial CRDs

MVP CRDs to implement first:

- `Disk`
- `Array`
- `Pool`
- `Share`
- `ParitySchedule`

Add OpenAPI schema validation for required fields and enums.

## 3) Implement Minimal Controllers

### DiskController (MVP)

- reconcile desired mount metadata
- surface status (`Ready`, `Degraded`, `Failed`)
- annotate with capacity and SMART summary from Node Agent

### ArrayController (MVP)

- validate data/parity disk references
- generate snapraid.conf fragments
- ensure array status reflects parity readiness

### PoolController (MVP)

- build mergerfs branch list from array disks
- ensure `/mnt/pool` mount is active through Node Agent

### ShareController (MVP)

- generate SMB/NFS share definitions
- reload service after config changes

### ParityScheduleController (MVP)

- map cron spec to Kubernetes `CronJob`
- trigger Node Agent parity operations

## 4) Node Agent Implementation

Recommended: **Go** for parity with operator codebase.

### Core packages

- `disk`: discovery (`lsblk`, `blkid`), mount ops
- `smart`: SMART query + health score normalization
- `parity`: SnapRAID command execution wrapper
- `share`: Samba/NFS templating and reload

### DaemonSet permissions

- privileged container
- host PID/IPC optional
- hostPath mounts: `/dev`, `/proc`, `/sys`, `/etc`, `/mnt`

## 5) Example Deployment Flow

```bash
kubectl apply -f deploy/crds/
kubectl apply -f deploy/operator.yaml
kubectl apply -f deploy/examples/node-agent-daemonset.yaml
kubectl apply -f examples/disks.yaml
kubectl apply -f examples/array.yaml
kubectl apply -f examples/pool.yaml
kubectl apply -f examples/shares.yaml
kubectl apply -f examples/parity-schedule.yaml
```

## 6) MVP Exit Criteria

- disks discovered and reflected in `Disk.status`
- array reconciles without errors
- pool mounted and writable at `/mnt/pool`
- SMB share reachable on LAN
- daily parity sync cron executes successfully
