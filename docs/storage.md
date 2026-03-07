# Storage Model

KubeNAS uses **independent data disks + parity + union pool**.

## Data Disks

- each disk is mounted separately (`/mnt/disks/<name>`)
- filesystem recommendation: `xfs`
- mixed-size disk support by design

## Parity

- SnapRAID parity disk(s) managed through `Array` + `ParitySchedule`
- periodic `sync`, `check`, and `scrub`

## Pooling

- mergerfs combines data disks into one namespace (`/mnt/pool`)
- placement policy driven by `PlacementPolicy`

## Recovery Benefits

- disk-level recoverability
- no striping lock-in
- straightforward host-level inspection
