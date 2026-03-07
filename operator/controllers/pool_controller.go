package controllers

import "context"

// PoolController reconciles mergerfs pool configuration.
type PoolController struct{}

func (c *PoolController) Name() string { return "pool-controller" }

func (c *PoolController) Reconcile(_ context.Context, _ ReconcileRequest) (ReconcileResult, error) {
	// TODO: compute mergerfs branches and ensure pool mount on host.
	return ReconcileResult{}, nil
}
