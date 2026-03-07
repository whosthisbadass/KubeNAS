package controllers

import "context"

// ReconcileRequest carries the minimal identity for a reconciliation target.
type ReconcileRequest struct {
	Namespace string
	Name      string
}

// ReconcileResult is a simplified result contract for controller loops.
type ReconcileResult struct {
	Requeue      bool
	RequeueAfter string
}

// Reconciler is a minimal interface used by KubeNAS controller scaffolding.
type Reconciler interface {
	Name() string
	Reconcile(ctx context.Context, req ReconcileRequest) (ReconcileResult, error)
}
