package handler

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/zdnscloud/cement/set"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

func createServiceAccount(cli client.Client, name, namespace string) error {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Create(context.TODO(), sa)
}

type ClusterRole string

const (
	ClusterAdmin    ClusterRole = "cluster-admin"
	ClusterReadOnly ClusterRole = "cluster-readonly"
)

func createClusterRole(cli client.Client, name string, role ClusterRole) error {
	r := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Rules:      policyRulesForRole(role),
	}
	return cli.Create(context.TODO(), r)
}

func policyRulesForRole(role ClusterRole) []rbacv1.PolicyRule {
	switch role {
	case ClusterAdmin:
		return []rbacv1.PolicyRule{
			rbacv1.PolicyRule{
				Verbs:     []string{rbacv1.VerbAll},
				APIGroups: []string{rbacv1.APIGroupAll},
				Resources: []string{rbacv1.ResourceAll},
			},
		}
	case ClusterReadOnly:
		return []rbacv1.PolicyRule{
			rbacv1.PolicyRule{
				Verbs:     []string{"get", "list", "watch"},
				APIGroups: []string{rbacv1.APIGroupAll},
				Resources: []string{rbacv1.ResourceAll},
			},
		}
	default:
		panic("unknown cluster role")
	}
}

func createRoleBinding(cli client.Client, clusterRoleName, serviceAccountName, serviceAccountNamespace string) error {
	binding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: clusterRoleName},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterRoleName,
		},
		Subjects: []rbacv1.Subject{
			rbacv1.Subject{
				Kind:      "ServiceAccount",
				Name:      serviceAccountName,
				Namespace: serviceAccountNamespace,
			},
		},
	}
	return cli.Create(context.TODO(), binding)
}

func scProtocolToK8SProtocol(protocol string) (p corev1.Protocol, err error) {
	switch strings.ToLower(protocol) {
	case "tcp":
		p = corev1.ProtocolTCP
	case "udp":
		p = corev1.ProtocolUDP
	default:
		err = fmt.Errorf("protocol %s isn't supported", protocol)
	}
	return
}

func scIngressProtocolToK8SProtocol(protocol types.IngressProtocol) corev1.Protocol {
	if protocol == types.IngressProtocolUDP {
		return corev1.ProtocolUDP
	} else {
		return corev1.ProtocolTCP
	}
}

func scServiceTypeToK8sServiceType(typ string) (p corev1.ServiceType, err error) {
	switch strings.ToLower(typ) {
	case "clusterip":
		p = corev1.ServiceTypeClusterIP
	case "nodeport":
		p = corev1.ServiceTypeNodePort
	default:
		err = fmt.Errorf("service type %s isn't supported", typ)
	}
	return
}

func scRestartPolicyToK8sRestartPolicy(policy string) (p corev1.RestartPolicy, err error) {
	switch strings.ToLower(policy) {
	case "onfailure":
		p = corev1.RestartPolicyOnFailure
	case "never":
		p = corev1.RestartPolicyNever
	default:
		err = fmt.Errorf("restart policy %s isn`t supported", policy)
	}
	return
}

func scLimitsResourceNameToK8sResourceName(name string) (k8sname corev1.ResourceName, err error) {
	switch strings.ToLower(name) {
	case "cpu":
		k8sname = corev1.ResourceCPU
	case "memory":
		k8sname = corev1.ResourceMemory
	default:
		err = fmt.Errorf("container limitrange resoucename %s isn`t supported", name)
	}
	return
}

func scQuotaResourceNameToK8sResourceName(name string) (k8sname corev1.ResourceName, err error) {
	switch strings.ToLower(name) {
	case "requests.cpu":
		k8sname = corev1.ResourceRequestsCPU
	case "requests.memory":
		k8sname = corev1.ResourceRequestsMemory
	case "limits.cpu":
		k8sname = corev1.ResourceLimitsCPU
	case "limits.memory":
		k8sname = corev1.ResourceLimitsMemory
	case "requests.storage":
		k8sname = corev1.ResourceRequestsStorage
	default:
		err = fmt.Errorf("resoucequota resourcename %s isn`t supported", name)
	}
	return
}

func createRole(cli client.Client, roleName string, role ClusterRole, namespace string) error {
	if err := createServiceAccount(cli, roleName, namespace); err != nil {
		return err
	}

	if err := createClusterRole(cli, roleName, role); err != nil {
		return err
	}

	if err := createRoleBinding(cli, roleName, roleName, namespace); err != nil {
		return err
	}

	return nil
}

func getConfigmapAndSecretContainersUse(namespace string, cli client.Client, containers []types.Container) ([]runtime.Object, error) {
	configmaps := set.NewStringSet()
	secrets := set.NewStringSet()
	var results []runtime.Object
	for _, container := range containers {
		for _, volume := range container.Volumes {
			if volume.Type == types.VolumeTypeConfigMap {
				if configmaps.Member(volume.Name) == false {
					configmaps.Add(volume.Name)
					cm, err := getConfigMap(cli, namespace, volume.Name)
					if err != nil {
						return nil, err
					}
					results = append(results, cm)
				}
			} else if volume.Type == types.VolumeTypeSecret {
				if secrets.Member(volume.Name) == false {
					secrets.Add(volume.Name)
					secret, err := getSecret(cli, namespace, volume.Name)
					if err != nil {
						return nil, err
					}
					results = append(results, secret)
				}
			}
		}
	}
	return results, nil
}

func calculateConfigHash(objects []runtime.Object) (string, error) {
	hashSource := struct {
		ConfigMaps map[string]map[string]string `json:"configMaps"`
		Secrets    map[string]map[string][]byte `json:"secrets"`
	}{
		ConfigMaps: make(map[string]map[string]string),
		Secrets:    make(map[string]map[string][]byte),
	}

	for _, obj := range objects {
		switch o := obj.(type) {
		case *corev1.ConfigMap:
			hashSource.ConfigMaps[o.Name] = o.Data
		case *corev1.Secret:
			hashSource.Secrets[o.Name] = o.Data
		default:
			return "", fmt.Errorf("unknown config type %v", obj)
		}
	}

	jsonData, err := json.Marshal(hashSource)
	if err != nil {
		return "", fmt.Errorf("unable to marshal JSON: %v", err)
	}

	hashBytes := sha256.Sum256(jsonData)
	return fmt.Sprintf("%x", hashBytes), nil
}
