package controllers

import "context"

// ShareController reconciles SMB/NFS shares and applies RBAC mappings from K8s users/groups.
type ShareController struct{}

func (c *ShareController) Name() string { return "share-controller" }

func (c *ShareController) Reconcile(_ context.Context, _ ReconcileRequest) (ReconcileResult, error) {
	// TODO: translate Share.spec.authz subjects (User/Group) to effective SMB/NFS ACLs.
	// TODO: integrate SubjectAccessReview for policy evaluation before publishing writes.
	return ReconcileResult{}, nil
}
