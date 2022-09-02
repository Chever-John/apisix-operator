package controllers

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apisixoperatorv1alpha1 "github.com/chever-john/apisix-operator/apis/v1alpha1"
	"github.com/chever-john/apisix-operator/internal/consts"
	dataplaneutils "github.com/chever-john/apisix-operator/internal/utils/dataplane"
)

// -----------------------------------------------------------------------------
// DataPlane - Private Functions - Generators
// -----------------------------------------------------------------------------

func generateNewDeploymentForDataPlane(dataplane *apisixoperatorv1alpha1.DataPlane, certSecretName string) *appsv1.Deployment {
	var dataplaneImage string
	if dataplane.Spec.ContainerImage != nil {
		dataplaneImage = *dataplane.Spec.ContainerImage
		if dataplane.Spec.Version != nil {
			dataplaneImage = fmt.Sprintf("%s:%s", dataplaneImage, *dataplane.Spec.Version)
		}
	} else {
		dataplaneImage = consts.DefaultDataPlaneImage
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    dataplane.Namespace,
			GenerateName: fmt.Sprintf("%s-%s-", consts.DataPlanePrefix, dataplane.Name),
			Labels: map[string]string{
				"app": dataplane.Name,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": dataplane.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": dataplane.Name,
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "cluster-certificate",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: certSecretName,
									Items: []corev1.KeyToPath{
										{
											Key:  "tls.crt",
											Path: "tls.crt",
										},
										{
											Key:  "tls.key",
											Path: "tls.key",
										},
										{
											Key:  "ca.crt",
											Path: "ca.crt",
										},
									},
								},
							},
						},
					},
					Containers: []corev1.Container{{
						Name: consts.DataPlaneProxyContainerName,
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "cluster-certificate",
								ReadOnly:  true,
								MountPath: "/var/cluster-certificate",
							},
						},
						Env:             dataplane.Spec.Env,
						EnvFrom:         dataplane.Spec.EnvFrom,
						Image:           dataplaneImage,
						ImagePullPolicy: corev1.PullIfNotPresent,
						Lifecycle: &corev1.Lifecycle{
							PreStop: &corev1.LifecycleHandler{
								Exec: &corev1.ExecAction{
									Command: []string{
										"/bin/sh",
										"-c",
										" quit",
									},
								},
							},
						},
						Ports: []corev1.ContainerPort{
							{
								Name:          "proxy",
								ContainerPort: consts.DataPlaneProxyPort,
								Protocol:      corev1.ProtocolTCP,
							},
							{
								Name:          "proxy-ssl",
								ContainerPort: consts.DataPlaneProxySSLPort,
								Protocol:      corev1.ProtocolTCP,
							},
							{
								Name:          "metrics",
								ContainerPort: consts.DataPlaneMetricsPort,
								Protocol:      corev1.ProtocolTCP,
							},
							{
								Name:          "admin-ssl",
								ContainerPort: consts.DataPlaneAdminAPIPort,
								Protocol:      corev1.ProtocolTCP,
							},
						},
						ReadinessProbe: &corev1.Probe{
							FailureThreshold:    3,
							InitialDelaySeconds: 5,
							PeriodSeconds:       10,
							SuccessThreshold:    1,
							TimeoutSeconds:      1,
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path:   "/status",
									Port:   intstr.FromInt(8100),
									Scheme: corev1.URISchemeHTTP,
								},
							},
						},
					}},
				},
			},
		},
	}
	return deployment
}

func generateNewServiceForDataplane(dataplane *apisixoperatorv1alpha1.DataPlane) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    dataplane.Namespace,
			GenerateName: fmt.Sprintf("%s-%s-", consts.DataPlanePrefix, dataplane.Name),
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeLoadBalancer,
			Selector: map[string]string{"app": dataplane.Name},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       dataplaneutils.DefaultHTTPPort,
					TargetPort: intstr.FromInt(dataplaneutils.DefaultAPISIXHTTPPort),
				},
				{
					Name:       "https",
					Protocol:   corev1.ProtocolTCP,
					Port:       dataplaneutils.DefaultHTTPSPort,
					TargetPort: intstr.FromInt(dataplaneutils.DefaultAPISIXHTTPSPort),
				},
				{
					Name:     "admin",
					Protocol: corev1.ProtocolTCP,
					Port:     dataplaneutils.DefaultAPISIXAdminPort,
				},
			},
		},
	}
}

// -----------------------------------------------------------------------------
// DataPlane - Private Functions - Kubernetes Object Labels
// -----------------------------------------------------------------------------

func addLabelForDataplane(obj client.Object) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[consts.GatewayOperatorControlledLabel] = consts.DataPlaneManagedLabelValue
	obj.SetLabels(labels)
}

// -----------------------------------------------------------------------------
// DataPlane - Private Functions - Equality Checks
// -----------------------------------------------------------------------------

func dataplaneSpecDeepEqual(spec1, spec2 *apisixoperatorv1alpha1.DataPlaneDeploymentOptions) bool {
	return deploymentOptionsDeepEqual(&spec1.DeploymentOptions, &spec2.DeploymentOptions)
}
