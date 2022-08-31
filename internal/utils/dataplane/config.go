package dataplane

import (
	"fmt"
	"sort"

	corev1 "k8s.io/api/core/v1"

	apisixoperatorv1alpha1 "github.com/chever-john/apisix-operator/apis/v1alpha1"
	k8sutils "github.com/chever-john/apisix-operator/internal/utils/kubernetes"
)

// -----------------------------------------------------------------------------
// Dataplane Utils - Config Vars & Consts
// -----------------------------------------------------------------------------

const (
	// DefaultHTTPPort is the default port used for HTTP ingress network traffic
	// from outside clusters.
	DefaultHTTPPort = 80

	// DefaultHTTPSPort is the default port used for HTTPS ingress network traffic
	// from outside clusters.
	DefaultHTTPSPort = 443

	// DefaultAPISIXHTTPPort is the APISIX proxy's default port used for HTTP traffic
	DefaultAPISIXHTTPPort = 9000

	// DefaultAPISIXHTTPSPort is the APISIX proxy's default port used for HTTPS traffic
	DefaultAPISIXHTTPSPort = 9443

	// DefaultAPISIXHTTPSPort is the default port used for APISIX Admin API traffic
	DefaultAPISIXAdminPort = 9444

	// DefaultAPISIXStatusPort is the default port used for APISIX proxy status
	DefaultAPISIXStatusPort = 9100
)

// APISIXDefaults are the baseline APISIX proxy configuration options needed for
// the proxy to function.
var APISIXDefaults = map[string]string{
	"APISIX_ADMIN_ACCESS_LOG":       "/dev/stdout",
	"APISIX_ADMIN_ERROR_LOG":        "/dev/stderr",
	"APISIX_ADMIN_GUI_ACCESS_LOG":   "/dev/stdout",
	"APISIX_ADMIN_GUI_ERROR_LOG":    "/dev/stderr",
	"APISIX_CLUSTER_LISTEN":         "off",
	"APISIX_DATABASE":               "off",
	"APISIX_NGINX_WORKER_PROCESSES": "2",
	"APISIX_PLUGINS":                "bundled",
	"APISIX_PORTAL_API_ACCESS_LOG":  "/dev/stdout",
	"APISIX_PORTAL_API_ERROR_LOG":   "/dev/stderr",
	"APISIX_PORT_MAPS":              "80:8000, 443:8443",
	"APISIX_PROXY_ACCESS_LOG":       "/dev/stdout",
	"APISIX_PROXY_ERROR_LOG":        "/dev/stderr",
	"APISIX_PROXY_LISTEN":           fmt.Sprintf("0.0.0.0:%d reuseport backlog=16384, 0.0.0.0:%d http2 ssl reuseport backlog=16384", DefaultAPISIXHTTPPort, DefaultAPISIXHTTPSPort),
	"APISIX_STATUS_LISTEN":          fmt.Sprintf("0.0.0.0:%d", DefaultAPISIXStatusPort),

	
	"APISIX_ADMIN_LISTEN": fmt.Sprintf("0.0.0.0:%d ssl reuseport backlog=16384", DefaultAPISIXAdminPort),

	// MTLS
	"APISIX_ADMIN_SSL_CERT":                     "/var/cluster-certificate/tls.crt",
	"APISIX_ADMIN_SSL_CERT_KEY":                 "/var/cluster-certificate/tls.key",
	"APISIX_NGINX_ADMIN_SSL_CLIENT_CERTIFICATE": "/var/cluster-certificate/ca.crt",
	"APISIX_NGINX_ADMIN_SSL_VERIFY_CLIENT":      "on",
	"APISIX_NGINX_ADMIN_SSL_VERIFY_DEPTH":       "3",
}

// -----------------------------------------------------------------------------
// DataPlane Utils - Config
// -----------------------------------------------------------------------------

// SetDataPlaneDefaults sets any unset default configuration options on the
// DataPlane. No configuration is overridden. EnvVars are sorted
// lexographically as a side effect.
func SetDataPlaneDefaults(spec *apisixoperatorv1alpha1.DataPlaneDeploymentOptions) {
	for k, v := range APISIXDefaults {
		envVar := corev1.EnvVar{Name: k, Value: v}
		if !k8sutils.IsEnvVarPresent(envVar, spec.Env) {
			spec.Env = append(spec.Env, envVar)
		}
	}
	sort.Sort(k8sutils.SortableEnvVars(spec.Env))
}
