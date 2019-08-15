package types

import (
	"fmt"
	"strings"
)

const (
	DefaultK8s = "v1.13.9"
)

var (
	K8sBadVersions = map[string]bool{
		"v1.9.7":  true,
		"v1.10.1": true,
		"v1.8.11": true,
		"v1.8.10": true,
	}

	// K8sVersionsCurrent are the latest versions available for installation
	K8sVersionsCurrent = []string{
		"v1.13.1",
		"v1.13.9",
	}

	// K8sVersionToZKESystemImages is dynamically populated on init() with the latest versions
	K8sVersionToZKESystemImages map[string]ZKEConfigImages

	// K8sVersionServiceOptions - service options per k8s version
	K8sVersionServiceOptions = map[string]KubernetesServicesOptions{
		"v1.13": {
			KubeAPI: map[string]string{
				"tls-cipher-suites":        "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305",
				"enable-admission-plugins": "NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,DefaultTolerationSeconds,MutatingAdmissionWebhook,ValidatingAdmissionWebhook,ResourceQuota",
			},
			Kubelet: map[string]string{
				"tls-cipher-suites": "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305",
			},
		},
	}

	AllK8sVersions = map[string]ZKEConfigImages{
		"v1.13.1": {
			Etcd:                      "zdnscloud/coreos-etcd:v3.2.24",
			Kubernetes:                "zdnscloud/hyperkube:v1.13.1",
			Alpine:                    "zdnscloud/zke-tools:v0.1.40",
			NginxProxy:                "zdnscloud/zke-tools:v0.1.40",
			CertDownloader:            "zdnscloud/zke-tools:v0.1.40",
			KubernetesServicesSidecar: "zdnscloud/zke-tools:v0.1.40",
			Flannel:                   "zdnscloud/coreos-flannel:v0.10.0",
			FlannelCNI:                "zdnscloud/coreos-flannel-cni:v0.3.0",
			CalicoNode:                "zdnscloud/calico-node:v3.4.0",
			CalicoCNI:                 "zdnscloud/calico-cni:v3.4.0",
			CalicoCtl:                 "zdnscloud/calico-ctl:v2.0.0",
			PodInfraContainer:         "zdnscloud/pause-amd64:3.1",
			Ingress:                   "zdnscloud/nginx-ingress-controller:0.21.0",
			IngressBackend:            "zdnscloud/nginx-ingress-controller-defaultbackend:1.4",
			CoreDNS:                   "zdnscloud/coredns:1.2.6",
			CoreDNSAutoscaler:         "zdnscloud/cluster-proportional-autoscaler-amd64:1.0.0",
			ClusterAgent:              "zdnscloud/cluster-agent:v2.3",
			NodeAgent:                 "zdnscloud/node-agent:v1.2",
			MetricsServer:             "zdnscloud/metrics-server-amd64:v0.3.3",
			ZKERemover:                "zdnscloud/zke-remove:v0.5",
			StorageOperator:           "zdnscloud/storage-operator:v1.6",
		},
		"v1.13.9": {
			Etcd:                      "zdnscloud/coreos-etcd:v3.2.24",
			Kubernetes:                "zdnscloud/hyperkube:v1.13.9",
			Alpine:                    "zdnscloud/zke-tools:v0.1.40",
			NginxProxy:                "zdnscloud/zke-tools:v0.1.40",
			CertDownloader:            "zdnscloud/zke-tools:v0.1.40",
			KubernetesServicesSidecar: "zdnscloud/zke-tools:v0.1.40",
			Flannel:                   "zdnscloud/coreos-flannel:v0.10.0",
			FlannelCNI:                "zdnscloud/coreos-flannel-cni:v0.3.0",
			CalicoNode:                "zdnscloud/calico-node:v3.4.0",
			CalicoCNI:                 "zdnscloud/calico-cni:v3.4.0",
			CalicoCtl:                 "zdnscloud/calico-ctl:v2.0.0",
			PodInfraContainer:         "zdnscloud/pause-amd64:3.1",
			Ingress:                   "zdnscloud/nginx-ingress-controller:0.21.0",
			IngressBackend:            "zdnscloud/nginx-ingress-controller-defaultbackend:1.4",
			CoreDNS:                   "zdnscloud/coredns:1.2.6",
			CoreDNSAutoscaler:         "zdnscloud/cluster-proportional-autoscaler-amd64:1.0.0",
			ClusterAgent:              "zdnscloud/cluster-agent:v2.3",
			NodeAgent:                 "zdnscloud/node-agent:v1.2",
			MetricsServer:             "zdnscloud/metrics-server-amd64:v0.3.3",
			ZKERemover:                "zdnscloud/zke-remove:v0.5",
			StorageOperator:           "zdnscloud/storage-operator:v1.6",
		},
	}
)

func init() {
	if K8sVersionToZKESystemImages != nil {
		panic("Do not initialize or add values to K8sVersionToZKESystemImages")
	}

	K8sVersionToZKESystemImages = map[string]ZKEConfigImages{}

	for version, images := range AllK8sVersions {
		if K8sBadVersions[version] {
			continue
		}

		longName := "zdnscloud/hyperkube:" + version
		if !strings.HasPrefix(longName, images.Kubernetes) {
			panic(fmt.Sprintf("For K8s version %s, the Kubernetes image tag should be a substring of %s, currently it is %s", version, version, images.Kubernetes))
		}
	}

	for _, latest := range K8sVersionsCurrent {
		images, ok := AllK8sVersions[latest]
		if !ok {
			panic("K8s version " + " is not found in AllK8sVersions map")
		}
		K8sVersionToZKESystemImages[latest] = images
	}

	if _, ok := K8sVersionToZKESystemImages[DefaultK8s]; !ok {
		panic("Default K8s version " + DefaultK8s + " is not found in k8sVersionsCurrent list")
	}
}
