package controllers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apisixoperatorv1alpha1 "github.com/chever-john/apisix-operator/apis/v1alpha1"
	k8sutils "github.com/chever-john/apisix-operator/internal/utils/kubernetes"
)
// -----------------------------------------------------------------------------
// DataPlaneReconciler - Status Management
// -----------------------------------------------------------------------------

func (r *DataPlaneReconciler) ensureIsMarkedScheduled(
	dataplane *apisixoperatorv1alpha1.DataPlane,
) bool {
	_, present := k8sutils.GetCondition(DataPlaneConditionTypeProvisioned, dataplane)
	if !present {
		condition := k8sutils.NewCondition(
			DataPlaneConditionTypeProvisioned,
			metav1.ConditionFalse,
			DataPlaneConditionReasonPodsNotReady,
			"DataPlane resource is scheduled for provisioning",
		)

		k8sutils.SetCondition(condition, dataplane)
		return true
	}
	return false
}
