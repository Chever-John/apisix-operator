package resources

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/chever-john/apisix-operator/internal/consts"
)

// -----------------------------------------------------------------------------
// ServiceAccount generators
// -----------------------------------------------------------------------------

// GenerateNewServiceAccountForControlPlane is a helper to generate a ServiceAccount
// to be used by the controlplane deployment.
func GenerateNewServiceAccountForControlPlane(namespace, controlplaneName string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-%s-", consts.ControlPlanePrefix, controlplaneName),
			Namespace:    namespace,
			Labels: map[string]string{
				"app": controlplaneName,
			},
		},
	}
}
