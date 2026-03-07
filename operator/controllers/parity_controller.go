package controllers

import "context"

// ParityController reconciles parity schedules and sync/check/scrub jobs.
type ParityController struct{}

func (c *ParityController) Name() string { return "parity-controller" }

func (c *ParityController) Reconcile(_ context.Context, _ ReconcileRequest) (ReconcileResult, error) {
	// TODO: create CronJobs and call node-agent snapraid workflows.
	return ReconcileResult{}, nil
}
