package scheduler

import (
	"errors"
	"sort"
)

// Strategy defines disk placement approach.
type Strategy string

const (
	StrategyBalanced  Strategy = "balanced"
	StrategyLeastUsed Strategy = "least-used"
	StrategyFillFirst Strategy = "fill-first"
	StrategyTiered    Strategy = "tiered"
)

// Weights control balanced score behavior.
type Weights struct {
	FreeSpace float64
	Health    float64
	IOLoad    float64
}

// Disk represents a placement candidate.
type Disk struct {
	Name           string
	CapacityBytes  int64
	AvailableBytes int64
	HealthScore    float64
	IOLoadScore    float64
	Rotational     bool
	Eligible       bool
}

// Score couples a disk with a computed score.
type Score struct {
	Disk  Disk
	Value float64
}

// Engine performs placement selection.
type Engine struct {
	Weights Weights
}

// NewEngine returns engine with MVP defaults from KubeNAS design.
func NewEngine() Engine {
	return Engine{Weights: Weights{FreeSpace: 0.7, Health: 0.2, IOLoad: 0.1}}
}

// Select returns the best disk based on strategy.
func (e Engine) Select(strategy Strategy, disks []Disk) (Disk, error) {
	eligible := filterEligible(disks)
	if len(eligible) == 0 {
		return Disk{}, errors.New("no eligible disks")
	}

	switch strategy {
	case StrategyLeastUsed:
		return selectLeastUsed(eligible), nil
	case StrategyFillFirst:
		return selectFillFirst(eligible), nil
	case StrategyTiered:
		return selectTiered(eligible), nil
	default:
		return e.selectBalanced(eligible), nil
	}
}

func filterEligible(disks []Disk) []Disk {
	out := make([]Disk, 0, len(disks))
	for _, d := range disks {
		if !d.Eligible || d.CapacityBytes <= 0 || d.AvailableBytes <= 0 {
			continue
		}
		out = append(out, d)
	}
	return out
}

func (e Engine) selectBalanced(disks []Disk) Disk {
	scored := make([]Score, 0, len(disks))
	for _, d := range disks {
		freeRatio := float64(d.AvailableBytes) / float64(d.CapacityBytes)
		score := freeRatio*e.Weights.FreeSpace + d.HealthScore*e.Weights.Health + d.IOLoadScore*e.Weights.IOLoad
		scored = append(scored, Score{Disk: d, Value: score})
	}
	sort.Slice(scored, func(i, j int) bool { return scored[i].Value > scored[j].Value })
	return scored[0].Disk
}

func selectLeastUsed(disks []Disk) Disk {
	best := disks[0]
	for _, d := range disks[1:] {
		if d.AvailableBytes > best.AvailableBytes {
			best = d
		}
	}
	return best
}

func selectFillFirst(disks []Disk) Disk {
	best := disks[0]
	for _, d := range disks[1:] {
		if (d.CapacityBytes - d.AvailableBytes) > (best.CapacityBytes - best.AvailableBytes) {
			best = d
		}
	}
	return best
}

func selectTiered(disks []Disk) Disk {
	for _, d := range disks {
		if !d.Rotational {
			return d
		}
	}
	return selectLeastUsed(disks)
}
