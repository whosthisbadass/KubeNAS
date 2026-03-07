#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────
# KubeNAS SNO Install Script
# Deploys KubeNAS Operator + Node Agent on Single Node OpenShift/OKD.
#
# Prerequisites:
#   - oc / kubectl configured with cluster-admin
#   - Storage node labeled: kubectl label node <node> kubenas.io/storage-node=true
#   - Host packages installed on SNO node:
#       rpm-ostree install smartmontools mergerfs snapraid samba nfs-utils
#       systemctl reboot
# ─────────────────────────────────────────────────────────────────
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# ── Configuration ─────────────────────────────────────────────────
STORAGE_NODE="${STORAGE_NODE:-}"
NAMESPACE="kubenas-system"
OC="${OC:-oc}"   # Use 'kubectl' for plain Kubernetes.

# ── Helpers ───────────────────────────────────────────────────────
info()  { echo -e "\033[0;32m[INFO]\033[0m  $*"; }
warn()  { echo -e "\033[0;33m[WARN]\033[0m  $*"; }
error() { echo -e "\033[0;31m[ERROR]\033[0m $*" >&2; exit 1; }

check_deps() {
  for cmd in oc kubectl; do
    command -v "$OC" > /dev/null 2>&1 && return 0
  done
  error "Neither 'oc' nor 'kubectl' found. Install OpenShift CLI."
}

label_storage_node() {
  if [[ -z "$STORAGE_NODE" ]]; then
    info "Auto-detecting storage node..."
    STORAGE_NODE=$($OC get nodes -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
    if [[ -z "$STORAGE_NODE" ]]; then
      error "Could not detect storage node. Set STORAGE_NODE=<nodename>"
    fi
  fi
  info "Labeling storage node: $STORAGE_NODE"
  $OC label node "$STORAGE_NODE" kubenas.io/storage-node=true --overwrite
}

# ── Install Steps ─────────────────────────────────────────────────
main() {
  check_deps

  info "═══════════════════════════════════════════════"
  info "  KubeNAS Installation — Single Node OpenShift  "
  info "═══════════════════════════════════════════════"

  # Step 1: Label storage node.
  label_storage_node

  # Step 2: Apply SCCs (OpenShift only — skip on plain Kubernetes).
  if $OC api-resources --api-group=security.openshift.io 2>/dev/null | grep -q SecurityContextConstraints; then
    info "Applying OpenShift Security Context Constraints..."
    $OC apply -f "${REPO_ROOT}/deploy/scc/scc.yaml"
  else
    warn "OpenShift SCC API not found — skipping SCC creation (plain Kubernetes mode)."
  fi

  # Step 3: Apply RBAC (namespace, service accounts, roles, bindings).
  info "Applying RBAC manifests..."
  $OC apply -f "${REPO_ROOT}/deploy/rbac/rbac.yaml"

  # Step 4: Apply CRDs.
  info "Applying Custom Resource Definitions..."
  $OC apply -f "${REPO_ROOT}/deploy/crds/"

  # Wait for CRDs to be established.
  info "Waiting for CRDs to be established..."
  for crd in disks arrays pools shares parityschedules placementpolicies rebalancejobs diskfailures cachepools; do
    $OC wait crd/${crd}.storage.kubenas.io --for=condition=Established --timeout=60s 2>/dev/null || \
      warn "CRD ${crd} not yet established — continuing..."
  done

  # Step 5: Deploy operator.
  info "Deploying KubeNAS Operator..."
  $OC apply -f "${REPO_ROOT}/deploy/operator.yaml"

  # Step 6: Deploy node agent DaemonSet.
  info "Deploying KubeNAS Node Agent DaemonSet..."
  $OC apply -f "${REPO_ROOT}/deploy/examples/node-agent-daemonset.yaml"

  # Step 7: Wait for operator to be ready.
  info "Waiting for operator deployment to be ready (timeout: 120s)..."
  $OC rollout status deployment/kubenas-operator -n "$NAMESPACE" --timeout=120s || \
    warn "Operator deployment not yet ready — check: oc -n $NAMESPACE get pods"

  # Step 8: Wait for node agent to be ready.
  info "Waiting for node agent DaemonSet to be ready (timeout: 120s)..."
  $OC rollout status daemonset/kubenas-node-agent -n "$NAMESPACE" --timeout=120s || \
    warn "Node agent not yet ready — check: oc -n $NAMESPACE get pods"

  info "═══════════════════════════════════════════════"
  info "  KubeNAS installation complete!               "
  info "═══════════════════════════════════════════════"
  info ""
  info "Next steps:"
  info "  1. Apply example disk/array/pool resources:"
  info "     oc apply -f ${REPO_ROOT}/examples/disks.yaml"
  info "     oc apply -f ${REPO_ROOT}/examples/array.yaml"
  info "     oc apply -f ${REPO_ROOT}/examples/pool.yaml"
  info "     oc apply -f ${REPO_ROOT}/examples/shares.yaml"
  info "     oc apply -f ${REPO_ROOT}/examples/parity-schedule.yaml"
  info ""
  info "  2. Check status:"
  info "     oc -n $NAMESPACE get disks,arrays,pools,shares"
  info "     oc -n $NAMESPACE describe array media-array"
  info ""
  info "  3. View operator logs:"
  info "     oc -n $NAMESPACE logs -l app.kubernetes.io/name=kubenas-operator -f"
}

main "$@"
