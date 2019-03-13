package handler

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/zdnscloud/gok8s/client"
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
