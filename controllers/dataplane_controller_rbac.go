package controllers

// -----------------------------------------------------------------------------
// DataPlaneReconciler - RBAC
// -----------------------------------------------------------------------------

//+kubebuilder:rbac:groups=apisix-operator.apisix-operator.apisix.apache.org,resources=dataplanes,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=apisix-operator.apisix-operator.apisix.apache.org,resources=dataplanes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apisix-operator.apisix-operator.apisix.apache.org,resources=dataplanes/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=create;get;list;watch;update;patch
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get
//+kubebuilder:rbac:groups=core,resources=services,verbs=create;get;list;watch;update;patch
//+kubebuilder:rbac:groups=core,resources=services/status,verbs=get
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
