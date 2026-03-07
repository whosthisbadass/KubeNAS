# Scheduling Engine

The placement engine determines which disk receives new files.

## Strategies

- `balanced`
- `least-used`
- `fill-first`
- `tiered`

## Weighted Scoring

```text
score = (free_space_ratio * 0.7)
      + (inverse_disk_load * 0.2)
      + (health_score * 0.1)
```

Additional constraints:

- minimum free space thresholds
- exclusion of unhealthy/degraded disks
- optional affinity to cache tiers

## Rebalance Interactions

- rebalancing can enforce target utilization bands
- scheduler aware of active rebalance jobs to avoid thrash
