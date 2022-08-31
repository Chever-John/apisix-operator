package resources_test

import (
	"reflect"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/chever-john/apisix-operator/internal/utils/kubernetes/resources"
	"github.com/chever-john/apisix-operator/internal/utils/kubernetes/resources/clusterroles"
)

func TestClusterroleHelpers(t *testing.T) {
	var testCases = []struct {
		controlplane        string
		image               string
		expectedClusterRole *rbacv1.ClusterRole
	}{
		{
			controlplane:        "test_2.1",
			image:               "apisix/apisix-ingress-controller:2.1",
			expectedClusterRole: clusterroles.GenerateNewClusterRoleForControlPlane_ge2_1_lt2_2("test_2.1"),
		},
		{
			controlplane:        "test_2.2.1",
			image:               "apisix/apisix-ingress-controller:2.2",
			expectedClusterRole: clusterroles.GenerateNewClusterRoleForControlPlane_ge2_2_lt2_3("test_2.2.1"),
		},
		{
			controlplane:        "test_2.3",
			image:               "apisix/apisix-ingress-controller:2.3",
			expectedClusterRole: clusterroles.GenerateNewClusterRoleForControlPlane_ge2_3_lt2_4("test_2.3"),
		},
		{
			controlplane:        "test_2.4.2",
			image:               "apisix/apisix-ingress-controller:2.4.2",
			expectedClusterRole: clusterroles.GenerateNewClusterRoleForControlPlane_ge2_4("test_2.4.2"),
		},
		{
			controlplane:        "test_latest",
			image:               "apisix/apisix-ingress-controller:latest",
			expectedClusterRole: clusterroles.GenerateNewClusterRoleForControlPlane_ge2_4("test_latest"),
		},
		{
			controlplane:        "test_empty",
			image:               "apisix/apisix-ingress-controller",
			expectedClusterRole: clusterroles.GenerateNewClusterRoleForControlPlane_ge2_4("test_empty"),
		},
	}

	for _, tc := range testCases {
		clusterRole, err := resources.GenerateNewClusterRoleForControlPlane(tc.controlplane, &tc.image)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(clusterRole, tc.expectedClusterRole) {
			t.Fatalf("clusterRole %s different from the expected one", clusterRole.Name)
		}
	}
}
