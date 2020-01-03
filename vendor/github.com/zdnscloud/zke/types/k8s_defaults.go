package types

import (
	"fmt"
	"reflect"

	"gopkg.in/yaml.v2"
)

const (
	DefaultK8s = "v1.13.10"
)

var (
	imageConfig        string
	K8sVersionsCurrent = []string{
		"v1.13.10",
	}

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

	AllK8sVersions = map[string]ZKEConfigImages{}
)

func init() {
	if err := yaml.Unmarshal([]byte(imageConfig), &AllK8sVersions); err != nil {
		panic(err.Error())
	}
	if err := validateImageConfig(AllK8sVersions); err != nil {
		panic(err.Error())
	}
}

func validateImageConfig(in map[string]ZKEConfigImages) error {
	for version, images := range in {
		t := reflect.TypeOf(images)
		v := reflect.ValueOf(images)
		for i := 0; i < t.NumField(); i++ {
			if v.Field(i).String() == "" {
				return fmt.Errorf("validate image config failed: k8s version %s field %s nil", version, t.Field(i).Name)
			}
		}
	}
	return nil
}
