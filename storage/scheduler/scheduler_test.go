package scheduler

import "testing"

func TestSelectBalanced(t *testing.T) {
	e := NewEngine()
	disks := []Disk{
		{Name: "a", CapacityBytes: 100, AvailableBytes: 20, HealthScore: 1.0, IOLoadScore: 0.2, Eligible: true},
		{Name: "b", CapacityBytes: 100, AvailableBytes: 50, HealthScore: 0.7, IOLoadScore: 0.7, Eligible: true},
	}
	selected, err := e.Select(StrategyBalanced, disks)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if selected.Name != "b" {
		t.Fatalf("expected b, got %s", selected.Name)
	}
}

func TestSelectTieredPrefersSSD(t *testing.T) {
	e := NewEngine()
	disks := []Disk{
		{Name: "hdd", CapacityBytes: 100, AvailableBytes: 80, HealthScore: 1.0, IOLoadScore: 0.5, Rotational: true, Eligible: true},
		{Name: "ssd", CapacityBytes: 100, AvailableBytes: 30, HealthScore: 1.0, IOLoadScore: 0.9, Rotational: false, Eligible: true},
	}
	selected, err := e.Select(StrategyTiered, disks)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if selected.Name != "ssd" {
		t.Fatalf("expected ssd, got %s", selected.Name)
	}
}
