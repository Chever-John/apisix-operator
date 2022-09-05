package controllers

import (
	"context"
	"errors"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	apisixoperatorv1alpha1 "github.com/chever-john/apisix-operator/apis/v1alpha1"
	operatorerrors "github.com/chever-john/apisix-operator/internal/errors"
	gatewayutils "github.com/chever-john/apisix-operator/internal/utils/gateway"
	k8sutils "github.com/chever-john/apisix-operator/internal/utils/kubernetes"
)

// -----------------------------------------------------------------------------
// ControlPlaneReconciler
// -----------------------------------------------------------------------------

// ControlPlaneReconciler reconciles a ControlPlane object
type ControlPlaneReconciler struct {
	client.Client
	Scheme                   *runtime.Scheme
	ClusterCASecretName      string
	ClusterCASecretNamespace string
}

// SetupWithManager sets up the controller with the Manager.
func (r *ControlPlaneReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// for owned objects we need to check if updates to the objects resulted in the
	// removal of an OwnerReference to the parent object, and if so we need to
	// enqueue the parent object so that reconciliation can create a replacement.
	clusterRolePredicate := predicate.NewPredicateFuncs(r.clusterRoleHasControlplaneOwner)
	clusterRolePredicate.UpdateFunc = func(e event.UpdateEvent) bool {
		return r.clusterRoleHasControlplaneOwner(e.ObjectOld)
	}
	clusterRoleBindingPredicate := predicate.NewPredicateFuncs(r.clusterRoleBindingHasControlplaneOwner)
	clusterRoleBindingPredicate.UpdateFunc = func(e event.UpdateEvent) bool {
		return r.clusterRoleBindingHasControlplaneOwner(e.ObjectOld)
	}

	return ctrl.NewControllerManagedBy(mgr).
		// watch Controlplane objects
		For(&apisixoperatorv1alpha1.ControlPlane{}).
		// watch for changes in Secrets created by the controlplane controller
		Owns(&corev1.Secret{}).
		// watch for changes in ServiceAccounts created by the controlplane controller
		Owns(&corev1.ServiceAccount{}).
		// watch for changes in Deployments created by the controlplane controller
		Owns(&appsv1.Deployment{}).
		// watch for changes in ClusterRoles created by the controlplane controller.
		// Since the ClusterRoles are cluster-wide but controlplanes are namespaced,
		// we need to manually detect the owner by means of the UID
		// (Owns cannot be used in this case)
		Watches(&source.Kind{Type: &rbacv1.ClusterRole{}},
			handler.EnqueueRequestsFromMapFunc(r.getControlplaneForClusterRole),
			builder.WithPredicates(clusterRolePredicate)).
		// watch for changes in ClusterRoleBindings created by the controlplane controller.
		// Since the ClusterRoleBindings are cluster-wide but controlplanes are namespaced,
		// we need to manually detect the owner by means of the UID
		// (Owns cannot be used in this case)
		Watches(
			&source.Kind{Type: &rbacv1.ClusterRoleBinding{}},
			handler.EnqueueRequestsFromMapFunc(r.getControlplaneForClusterRoleBinding),
			builder.WithPredicates(clusterRoleBindingPredicate)).
		Watches(
			&source.Kind{Type: &apisixoperatorv1alpha1.DataPlane{}},
			&handler.EnqueueRequestForOwner{OwnerType: &apisixoperatorv1alpha1.ControlPlane{}, IsController: true}).
		Complete(r)
}

// Reconcile moves the current state of an object to the intended state.
func (r *ControlPlaneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithName("ControlPlane")

	debug(log, "reconciling ControlPlane resource", req)
	controlplane := new(apisixoperatorv1alpha1.ControlPlane)
	if err := r.Client.Get(ctx, req.NamespacedName, controlplane); err != nil {
		if k8serrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// controlplane is deleted, just run garbage collection for cluster wide resources.
	if !controlplane.DeletionTimestamp.IsZero() {
		// wait for termination grace period before cleaning up roles and bindings
		if controlplane.DeletionTimestamp.After(metav1.Now().Time) {
			debug(log, "control plane deletion still under grace period", controlplane)
			return ctrl.Result{
				Requeue: true,
				// Requeue when grace period expires.
				// If deletion timestamp is changed,
				// the update will trigger another round of reconciliation.
				// so we do not consider updates of deletion timestamp here.
				RequeueAfter: time.Until(controlplane.DeletionTimestamp.Time),
			}, nil
		}

		debug(log, "removing owned cluster roles and cluster role bindings", controlplane)

		// ensure that the clusterrolebindings which were created for the ControlPlane are deleted
		if err := r.ensureOwnedClusterRoleBindingsDeleted(ctx, controlplane); err != nil {
			return ctrl.Result{}, err // ClusterRoleBinding deletion will requeue
		}

		// now that ClusterRoleBindings are cleaned up, remove the relevant finalizer
		if k8sutils.RemoveFinalizerInMetadata(&controlplane.ObjectMeta, string(ControlPlaneFinalizerCleanupClusterRoleBinding)) {
			return ctrl.Result{}, r.Client.Update(ctx, controlplane) // Controlplane update will requeue
		}

		// ensure that the clusterroles created for the controlplane are deleted
		if err := r.ensureOwnedClusterRolesDeleted(ctx, controlplane); err != nil {
			return ctrl.Result{}, err // ClusterRole deletion will requeue
		}

		// now that ClusterRoles are cleaned up, remove the relevant finalizer
		if k8sutils.RemoveFinalizerInMetadata(&controlplane.ObjectMeta, string(ControlPlaneFinalizerCleanupClusterRole)) {
			return ctrl.Result{}, r.Client.Update(ctx, controlplane) // Controlplane update will requeue
		}

		// cleanup completed
		return ctrl.Result{}, nil
	}

	// ensure the controlplane has a finalizer to delete owned cluster wide resources on delete.
	finalizersChanged := k8sutils.EnsureFinalizersInMetadata(&controlplane.ObjectMeta,
		string(ControlPlaneFinalizerCleanupClusterRole),
		string(ControlPlaneFinalizerCleanupClusterRoleBinding))
	if finalizersChanged {
		info(log, "update metadata of control plane to set finalizer", controlplane.ObjectMeta)
		return ctrl.Result{}, r.Client.Update(ctx, controlplane)
	}

	k8sutils.InitReady(controlplane)

	debug(log, "validating ControlPlane resource conditions", controlplane)
	if r.ensureIsMarkedScheduled(controlplane) {
		err := r.updateStatus(ctx, controlplane)
		if err != nil {
			debug(log, "unable to update ControlPlane resource", controlplane)
			return ctrl.Result{}, err
		}

		debug(log, "ControlPlane resource now marked as scheduled", controlplane)
		return ctrl.Result{}, nil // no need to requeue, status update will requeue
	}

	debug(log, "retrieving connected dataplane", controlplane)
	dataplane, err := gatewayutils.GetDataPlaneForControlPlane(ctx, r.Client, controlplane)
	var dataplaneServiceName string
	if err != nil {
		if !errors.Is(err, operatorerrors.ErrDataPlaneNotSet) {
			return ctrl.Result{}, err
		}
		debug(log, "no existing dataplane for controlplane", controlplane, "error", err)
	} else {
		if err := controllerutil.SetOwnerReference(controlplane, dataplane, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
		dataplaneServiceName, err = gatewayutils.GetDataplaneServiceName(ctx, r.Client, dataplane)
		if err != nil {
			debug(log, "no existing dataplane service for controlplane", controlplane, "error", err)
			return ctrl.Result{}, err
		}
	}

	debug(log, "validating ControlPlane configuration", controlplane)

	debug(log, "configuring ControlPlane resource", controlplane)
	changed := setControlPlaneDefaults(&controlplane.Spec.ControlPlaneDeploymentOptions, controlplane.Namespace, dataplaneServiceName, nil)
	if changed {
		debug(log, "updating ControlPlane resource after defaults are set since resource has changed", controlplane)
		err := r.Client.Update(ctx, controlplane)
		if err != nil {
			if k8serrors.IsConflict(err) {
				debug(log, "conflict found when updating ControlPlane resource, retrying", controlplane)
				return ctrl.Result{Requeue: true, RequeueAfter: requeueWithoutBackoff}, nil
			}
		}
		return ctrl.Result{}, err // no need to requeue, the update will trigger.
	}

	debug(log, "validating that the ControlPlane's DataPlane configuration is up to date", controlplane)
	if err = r.ensureDataPlaneConfiguration(ctx, controlplane, dataplaneServiceName); err != nil {
		if k8serrors.IsConflict(err) {
			debug(
				log,
				"conflict found when trying to ensure ControlPlane's DataPlane configuration was up to date, retrying",
				controlplane,
			)
			return ctrl.Result{Requeue: true, RequeueAfter: requeueWithoutBackoff}, nil
		}
		return ctrl.Result{}, err
	}

	debug(log, "validating ControlPlane's DataPlane status", controlplane)
	dataplaneIsSet := r.ensureDataPlaneStatus(controlplane)
	if dataplaneIsSet {
		debug(log, "DataPlane was set, deployment for ControlPlane will be provisioned", controlplane)
	} else {
		debug(log, "DataPlane not set, deployment for ControlPlane will remain dormant", controlplane)
	}

	debug(log, "ensuring ServiceAccount for ControlPlane deployment exists", controlplane)
	createdOrUpdated, controlplaneServiceAccount, err := r.ensureServiceAccountForControlPlane(ctx, controlplane)
	if err != nil {
		return ctrl.Result{}, err
	}
	if createdOrUpdated {
		return ctrl.Result{}, nil // requeue will be triggered by the creation or update of the owned object
	}

	debug(log, "ensuring ClusterRoles for ControlPlane deployment exist", controlplane)
	createdOrUpdated, controlplaneClusterRole, err := r.ensureClusterRoleForControlPlane(ctx, controlplane)
	if err != nil {
		return ctrl.Result{}, err
	}
	if createdOrUpdated {
		return ctrl.Result{}, nil // requeue will be triggered by the creation or update of the owned object
	}

	debug(log, "ensuring that ClusterRoleBindings for ControlPlane Deployment exist", controlplane)
	createdOrUpdated, _, err = r.ensureClusterRoleBindingForControlPlane(ctx, controlplane, controlplaneServiceAccount.Name, controlplaneClusterRole.Name)
	if err != nil {
		return ctrl.Result{}, err
	}
	if createdOrUpdated {
		return ctrl.Result{}, nil // requeue will be triggered by the creation or update of the owned object
	}

	debug(log, "creating mTLS certificate", controlplane)
	created, certSecret, err := r.ensureCertificate(ctx, controlplane)
	if err != nil {
		return ctrl.Result{}, err
	}
	if created {
		return ctrl.Result{}, nil // requeue will be triggered by the creation or update of the owned object
	}

	debug(log, "looking for existing Deployments for ControlPlane resource", controlplane)
	createdOrUpdated, controlplaneDeployment, err := r.ensureDeploymentForControlPlane(ctx, controlplane, controlplaneServiceAccount.Name, certSecret.Name)
	if err != nil {
		return ctrl.Result{}, err
	}
	if createdOrUpdated {
		if !dataplaneIsSet {
			debug(log, "DataPlane not set, deployment for ControlPlane has been scaled down to 0 replicas", controlplane)
			err := r.updateStatus(ctx, controlplane)
			if err != nil {
				debug(log, "unable to reconcile ControlPlane status", controlplane)
			}
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil // requeue will be triggered by the creation or update of the owned object
	}


	debug(log, "checking readiness of ControlPlane deployments", controlplane)

	if controlplaneDeployment.Status.Replicas == 0 || controlplaneDeployment.Status.AvailableReplicas < controlplaneDeployment.Status.Replicas {
		debug(log, "deployment for ControlPlane not yet ready, waiting", controlplane)
		return ctrl.Result{}, nil // requeue will be triggered by the status update
	}

	r.ensureIsMarkedProvisioned(controlplane)
	err = r.updateStatus(ctx, controlplane)

	if err != nil {
		if k8serrors.IsConflict(err) {
			// no need to throw an error for 409's, just requeue to get a fresh copy
			debug(log, "conflict during ControlPlane reconciliation", controlplane)
			return ctrl.Result{Requeue: true, RequeueAfter: requeueWithoutBackoff}, nil
		}
		debug(log, "unable to reconcile the ControlPlane resource", controlplane)
	} else {
		debug(log, "reconciliation complete for ControlPlane resource", controlplane)
	}
	return ctrl.Result{}, err
}

// updateStatus Updates the resource status only when there are changes in the Conditions
func (r *ControlPlaneReconciler) updateStatus(ctx context.Context, updated *apisixoperatorv1alpha1.ControlPlane) error {
	current := &apisixoperatorv1alpha1.ControlPlane{}
	err := r.Client.Get(ctx, client.ObjectKeyFromObject(updated), current)

	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	if k8sutils.NeedsUpdate(current, updated) {
		return r.Client.Status().Update(ctx, updated)
	}
	return nil
}
