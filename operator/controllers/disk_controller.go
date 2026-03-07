package controllers

import "context"

// DiskController reconciles Disk resources with node-agent disk discovery state.
type DiskController struct{}

func (c *DiskController) Name() string { return "disk-controller" }

func (c *DiskController) Reconcile(_ context.Context, _ ReconcileRequest) (ReconcileResult, error) {
	// TODO: call node-agent to discover/mount disks and update Disk.status.
	return ReconcileResult{}, nil
}
