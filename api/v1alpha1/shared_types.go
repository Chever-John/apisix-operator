package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
)

// DeploymentOptions 是一个 shared type，用来表示在一个 Deployment 中可能需要
// 注意的一些信息，比如 镜像、镜像版本、环境变量。
type DeploymentOptions struct {
	// 存储了部署的镜像
	//
	// +optional
	ContainerImage *string `json:"containerImage,omitempty"`

	// 镜像的版本
	//
	// +optional
	Version *string `json:"version,omitempty"`

	// Env 用来存储部署过程中可能需要存储的一些环境变量。
	//
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// EnvFrom 是为了存储一些参数，比如说 Secrets
	//
	// +optional
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`
}

type GatewayConfigureTargetKind string

const (
	GatewayConfigureTargetKindGateway GatewayConfigureTargetKind = "ApisixGateway"

	GatewayConfigureTargetKindGatewayClass GatewayConfigureTargetKind = "ApisixGatewayClass"
)
