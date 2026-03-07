package controllers

import "context"

// ArrayController reconciles Array resources and generates snapraid intent.
type ArrayController struct{}

func (c *ArrayController) Name() string { return "array-controller" }

func (c *ArrayController) Reconcile(_ context.Context, _ ReconcileRequest) (ReconcileResult, error) {
	// TODO: validate data/parity disk refs and push snapraid config to node-agent.
	return ReconcileResult{}, nil
}
