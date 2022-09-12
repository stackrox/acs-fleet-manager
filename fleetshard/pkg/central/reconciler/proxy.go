package reconciler

import (
	"fmt"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

func getProxyEnvVars(namespace string) []corev1.EnvVar {
	var envVars []corev1.EnvVar
	proxyURL := fmt.Sprintf("http://egress-proxy.%s.svc:3128", namespace)
	// Use both upper- and lowercase env var names for maximum compatibility.
	for _, envVarName := range []string{"http_proxy", "HTTP_PROXY", "https_proxy", "HTTPS_PROXY", "all_proxy", "ALL_PROXY"} {
		envVars = append(envVars, corev1.EnvVar{
			Name:  envVarName,
			Value: proxyURL,
		})
	}

	directServicesAndPorts := map[string][]int{
		"central":    {443},
		"scanner":    {8080, 8443},
		"scanner-db": {5432},
	}
	var noProxyTargets []string
	for svcName, ports := range directServicesAndPorts {
		for _, port := range ports {
			noProxyTargets = append(noProxyTargets,
				fmt.Sprintf("%s:%d", svcName, port),
				fmt.Sprintf("%s.%s:%d", svcName, namespace, port),
				fmt.Sprintf("%s.%s.svc:%d", svcName, namespace, port),
			)
		}
	}
	sort.Strings(noProxyTargets) // ensure deterministic output
	noProxyStr := strings.Join(noProxyTargets, ",")
	for _, envVarName := range []string{"no_proxy", "NO_PROXY"} {
		envVars = append(envVars, corev1.EnvVar{
			Name:  envVarName,
			Value: noProxyStr,
		})
	}

	return envVars
}
