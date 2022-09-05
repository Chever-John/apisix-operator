package controllers

// -----------------------------------------------------------------------------
// ControlPlane - Finalizers
// -----------------------------------------------------------------------------

// ControlPlaneFinalizer defines finalizers added by controlplane controller.
type ControlPlaneFinalizer string

const (
	// ControlPlaneFinalizerCleanupClusterRole is the finalizer to cleanup clusterroles owned by controlplane on deleting.
	ControlPlaneFinalizerCleanupClusterRole ControlPlaneFinalizer = "apisix-operator.apisix.apache.org/cleanup-clusterrole"
	// ControlPlaneFinalizerCleanupClusterRoleBinding is the finalizer to cleanup clusterrolebindings owned by controlplane on deleting.
	ControlPlaneFinalizerCleanupClusterRoleBinding ControlPlaneFinalizer = "apisix-operator.apisix.apache.org/cleanup-clusterrolebinding"
)
