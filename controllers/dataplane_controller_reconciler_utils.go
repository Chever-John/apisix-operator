package controllers

import (
	"context"
	"fmt"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apisixoperatorv1alpha1 "github.com/chever-john/apisix-operator/apis/v1alpha1"
	"github.com/chever-john/apisix-operator/internal/consts"
	k8sutils "github.com/chever-john/apisix-operator/internal/utils/kubernetes"
	k8sresources "github.com/chever-john/apisix-operator/internal/utils/kubernetes/resources"
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

func (r *DataPlaneReconciler) ensureIsMarkedProvisioned(
	dataplane *apisixoperatorv1alpha1.DataPlane,
) {
	condition := k8sutils.NewCondition(
		DataPlaneConditionTypeProvisioned,
		metav1.ConditionTrue,
		DataPlaneConditionReasonPodsReady,
		"pods for all Deployments are ready",
	)
	k8sutils.SetCondition(condition, dataplane)
	k8sutils.SetReady(dataplane)
}

func (r *DataPlaneReconciler) ensureDataPlaneServiceStatus(
	ctx context.Context,
	dataplane *apisixoperatorv1alpha1.DataPlane,
	serviceName string,
) error {
	dataplane.Status.Service = serviceName
	return r.Status().Update(ctx, dataplane)
}

// isSameDataPlaneCondition returns true if two `metav1.Condition`s
// indicates the same condition of a `DataPlane` resource.
func isSameDataPlaneCondition(condition1, condition2 metav1.Condition) bool {
	return condition1.Type == condition2.Type &&
		condition1.Status == condition2.Status &&
		condition1.Reason == condition2.Reason &&
		condition1.Message == condition2.Message
}

func (r *DataPlaneReconciler) ensureDataPlaneIsMarkedNotProvisioned(
	ctx context.Context,
	dataplane *apisixoperatorv1alpha1.DataPlane,
	reason k8sutils.ConditionReason, message string,
) error {
	notProvisionedCondition := metav1.Condition{
		Type:               string(DataPlaneConditionTypeProvisioned),
		Status:             metav1.ConditionFalse,
		Reason:             string(reason),
		Message:            message,
		ObservedGeneration: dataplane.Generation,
		LastTransitionTime: metav1.Now(),
	}

	conditionFound := false
	shouldUpdate := false
	for i, condition := range dataplane.Status.Conditions {
		// update the condition if condition has type `provisioned`, and the condition is not the same.
		if condition.Type == string(DataPlaneConditionTypeProvisioned) {
			conditionFound = true
			// update the slice if the condition is not the same as we expected.
			if !isSameDataPlaneCondition(notProvisionedCondition, condition) {
				dataplane.Status.Conditions[i] = notProvisionedCondition
				shouldUpdate = true
			}
		}
	}

	if !conditionFound {
		// append a new condition if provisioned condition is not found.
		dataplane.Status.Conditions = append(dataplane.Status.Conditions, notProvisionedCondition)
		shouldUpdate = true
	}

	if shouldUpdate {
		return r.Status().Update(ctx, dataplane)
	}
	return nil
}

// -----------------------------------------------------------------------------
// DataPlaneReconciler - Owned Resource Management
// -----------------------------------------------------------------------------

func (r *DataPlaneReconciler) ensureCertificate(
	ctx context.Context,
	dataplane *apisixoperatorv1alpha1.DataPlane,
	serviceName string,
) (bool, *corev1.Secret, error) {
	usages := []certificatesv1.KeyUsage{
		certificatesv1.UsageKeyEncipherment,
		certificatesv1.UsageDigitalSignature, certificatesv1.UsageServerAuth,
	}
	return maybeCreateCertificateSecret(ctx,
		dataplane,
		fmt.Sprintf("%s.%s.svc", serviceName, dataplane.Namespace),
		r.ClusterCASecretName,
		r.ClusterCASecretNamespace,
		usages,
		r.Client)
}

func (r *DataPlaneReconciler) ensureDeploymentForDataPlane(
	ctx context.Context,
	dataplane *apisixoperatorv1alpha1.DataPlane,
	certSecretName string,
) (createdOrUpdate bool, deploy *appsv1.Deployment, err error) {
	deployments, err := k8sutils.ListDeploymentsForOwner(
		ctx,
		r.Client,
		consts.GatewayOperatorControlledLabel,
		consts.DataPlaneManagedLabelValue,
		dataplane.Namespace,
		dataplane.UID,
	)
	if err != nil {
		return false, nil, err
	}

	count := len(deployments)
	if count > 1 {
		return false, nil, fmt.Errorf("found %d deployments for DataPlane currently unsupported: expected 1 or less", count)
	}

	generatedDeployment := generateNewDeploymentForDataPlane(dataplane, certSecretName)
	k8sutils.SetOwnerForObject(generatedDeployment, dataplane)
	addLabelForDataplane(generatedDeployment)

	if count == 1 {
		var updated bool
		existingDeployment := &deployments[0]
		updated, existingDeployment.ObjectMeta = k8sutils.EnsureObjectMetaIsUpdated(existingDeployment.ObjectMeta, generatedDeployment.ObjectMeta)

		// We do not want to permit direct edits of the Deployment environment. Any user-supplied values should be set
		// in the DataPlane. If the actual Deployment environment does not match the generated environment, either
		// something requires an update or there was a manual edit we want to purge.
		container := k8sresources.GetPodContainerByName(&existingDeployment.Spec.Template.Spec, consts.DataPlaneProxyContainerName)
		if container == nil {
			// someone has deleted the main container from the Deployment for ??? reasons. we can't fathom why they
			// would do this, but don't allow it and replace the container set entirely
			existingDeployment.Spec.Template.Spec.Containers = generatedDeployment.Spec.Template.Spec.Containers
			updated = true
			container = k8sresources.GetPodContainerByName(&existingDeployment.Spec.Template.Spec, consts.DataPlaneProxyContainerName)
		}
		if !reflect.DeepEqual(container.Env, dataplane.Spec.Env) {
			container.Env = dataplane.Spec.Env
			updated = true
		}

		if !reflect.DeepEqual(container.EnvFrom, dataplane.Spec.EnvFrom) {
			container.EnvFrom = dataplane.Spec.EnvFrom
			updated = true
		}

		if updated {
			return true, existingDeployment, r.Client.Update(ctx, existingDeployment)
		}
		return false, existingDeployment, nil
	}

	return true, generatedDeployment, r.Client.Create(ctx, generatedDeployment)
}

func (r *DataPlaneReconciler) ensureServiceForDataPlane(
	ctx context.Context,
	dataplane *apisixoperatorv1alpha1.DataPlane,
) (createdOrUpdated bool, svc *corev1.Service, err error) {
	services, err := k8sutils.ListServicesForOwner(
		ctx,
		r.Client,
		consts.GatewayOperatorControlledLabel,
		consts.DataPlaneManagedLabelValue,
		dataplane.Namespace,
		dataplane.UID,
	)
	if err != nil {
		return false, nil, err
	}

	count := len(services)
	if count > 1 {
		return false, nil, fmt.Errorf("found %d services for DataPlane currently unsupported: expected 1 or less", count)
	}

	generatedService := generateNewServiceForDataplane(dataplane)
	addLabelForDataplane(generatedService)
	k8sutils.SetOwnerForObject(generatedService, dataplane)

	if count == 1 {
		var updated bool
		existingService := &services[0]
		updated, existingService.ObjectMeta = k8sutils.EnsureObjectMetaIsUpdated(existingService.ObjectMeta, generatedService.ObjectMeta)
		if updated {
			return true, existingService, r.Client.Update(ctx, existingService)
		}
		return false, existingService, nil
	}

	return true, generatedService, r.Client.Create(ctx, generatedService)
}
