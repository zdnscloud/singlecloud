package k8smanager

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/zdnscloud/gok8s/client"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

func getNodes(cli client.Client) (*corev1.NodeList, error) {
	nodes := corev1.NodeList{}
	err := cli.List(context.TODO(), nil, &nodes)
	return &nodes, err
}

func k8sNodeToSCNode(k8sNode *corev1.Node) *types.Node {
	status := &k8sNode.Status

	var address, host string
	for _, addr := range status.Addresses {
		if addr.Type == corev1.NodeHostName {
			host = addr.Address
		} else if addr.Type == corev1.NodeInternalIP || addr.Type == corev1.NodeExternalIP {
			if address == "" {
				address = addr.Address
			}
		}
	}

	var cpu, memory, storage string
	var podCount int
	for typ, c := range status.Capacity {
		if typ == corev1.ResourceCPU {
			cpu = c.String()
		} else if typ == corev1.ResourceMemory {
			memory = c.String()
		} else if typ == corev1.ResourceEphemeralStorage {
			storage = c.String()
		} else if typ == corev1.ResourcePods {
			v, _ := c.AsInt64()
			podCount = int(v)
		}
	}

	nodeInfo := &status.NodeInfo
	os := nodeInfo.OperatingSystem + " " + nodeInfo.KernelVersion
	osImage := nodeInfo.OSImage
	dockderVersion := nodeInfo.ContainerRuntimeVersion

	var roles []string
	v, ok := k8sNode.Labels["node-role.kubernetes.io/controlplane"]
	if ok && (v == "true") {
		roles = append(roles, "controlplane")
	}

	v, ok = k8sNode.Labels["node-role.kubernetes.io/etcd"]
	if ok && (v == "true") {
		roles = append(roles, "etcd")
	}

	v, ok = k8sNode.Labels["node-role.kubernetes.io/worker"]
	if ok && (v == "true") {
		roles = append(roles, "worker")
	}

	node := &types.Node{
		Name:                 host,
		Address:              address,
		Role:                 strings.Join(roles, ","),
		Labels:               k8sNode.Labels,
		Annotations:          k8sNode.Annotations,
		OperatingSystem:      os,
		OperatingSystemImage: osImage,
		DockerVersion:        dockderVersion,
		Cpu:                  cpu,
		Memory:               memory,
		Storage:              storage,
		PodCount:             podCount,
		CreationTimestamp:    k8sNode.CreationTimestamp.String(),
	}
	node.SetID(node.Name)
	node.SetType(resttypes.GetResourceType(node))
	return node
}

func createNamespace(cli client.Client, name string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	return cli.Create(context.TODO(), ns)
}

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
