/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	apisixoperatorv1alpha1 "github.com/chever-john/apisix-operator/apis/v1alpha1"
	k8sutils "github.com/chever-john/apisix-operator/internal/utils/kubernetes"
)

// DataPlaneReconciler reconciles a DataPlane object
type DataPlaneReconciler struct {
	client.Client
	Scheme                   *runtime.Scheme
	eventRecorder            record.EventRecorder
	ClusterCASecretName      string
	ClusterCASecretNamespace string
}

//+kubebuilder:rbac:groups=apisix-operator.apisix-operator.apisix.apache.org,resources=dataplanes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apisix-operator.apisix-operator.apisix.apache.org,resources=dataplanes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apisix-operator.apisix-operator.apisix.apache.org,resources=dataplanes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DataPlane object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// Reconcile 是 k8s 调和循环的一部分，其目的是使集群的当前状态更加接近于理想状态。
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *DataPlaneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithName("DataPlane")

	debug(log, "reconciling DataPlane resource", req)

	dataplane := new(apisixoperatorv1alpha1.DataPlane)
	if err := r.Client.Get(ctx, req.NamespacedName, dataplane); err != nil {
		if k8serrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	k8sutils.InitReady(dataplane)
	// TODO(user): your logic here
	debug(log, "validating DataPlane resource condition", dataplane)

	if r.ensureIsMarkedScheduled(dataplane) {
		err := r.updateStatus(ctx, dataplane)

		if err != nil {
			debug(log, "unable to update DataPlane resource", dataplane)
		}
		return ctrl.Result{}, err
	}

	debug(log, "exposing DataPlane deployment via service", dataplane)
	createdOrUpdated, dataplaneService, err := r.ensureServiceForDataPlane(ctx, dataplane)

	if err != nil {
		return ctrl.Result{}, err
	}

	if createdOrUpdated {
		return ctrl.Result{}, r.ensureDataPlaneServiceStatus(ctx, dataplane, dataplaneService.Name)
	}

	return ctrl.Result{}, err

}
func (r *DataPlaneReconciler) updateStatus(ctx context.Context, updated *apisixoperatorv1alpha1.DataPlane) error {
	current := &apisixoperatorv1alpha1.DataPlane{}

	err := r.Client.Get(ctx, client.ObjectKeyFromObject(updated), current)
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	if k8sutils.NeedsUpdate(current, updated) {
		return r.Client.Status().Update(ctx, updated)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DataPlaneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apisixoperatorv1alpha1.DataPlane{}).
		Complete(r)
}
