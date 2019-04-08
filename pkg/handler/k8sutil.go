package handler

import (
	"context"
	"fmt"
	"strings"

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

func scLimitRangeTypeToK8sLimitRangeType(typ string) (t corev1.LimitType, err error) {
	switch strings.ToLower(typ) {
	case "pod":
		t = corev1.LimitTypePod
	case "container":
		t = corev1.LimitTypeContainer
	case "persistentvolumeclaim":
		t = corev1.LimitTypePersistentVolumeClaim
	default:
		err = fmt.Errorf("limit range type %s isn`t supported", typ)
	}
	return
}

func scLimitsResourceNameToK8sResourceName(name string) (k8sname corev1.ResourceName, err error) {
	switch strings.ToLower(name) {
	case "cpu":
		k8sname = corev1.ResourceCPU
	case "memory":
		k8sname = corev1.ResourceMemory
	case "storage":
		k8sname = corev1.ResourceStorage
	case "ephemeral-storage":
		k8sname = corev1.ResourceEphemeralStorage
	default:
		err = fmt.Errorf("limitrange resoucename %s isn`t supported", name)
	}
	return
}

func scQuotaResourceNameToK8sResourceName(name string) (k8sname corev1.ResourceName, err error) {
	switch strings.ToLower(name) {
	case "cpu":
		k8sname = corev1.ResourceCPU
	case "memory":
		k8sname = corev1.ResourceMemory
	case "storage":
		k8sname = corev1.ResourceStorage
	case "ephemeral-storage":
		k8sname = corev1.ResourceEphemeralStorage
	case "pods":
		k8sname = corev1.ResourcePods
	case "services":
		k8sname = corev1.ResourceServices
	case "replicationcontrollers":
		k8sname = corev1.ResourceReplicationControllers
	case "resourcequotas":
		k8sname = corev1.ResourceQuotas
	case "secrets":
		k8sname = corev1.ResourceSecrets
	case "configmaps":
		k8sname = corev1.ResourceConfigMaps
	case "persistentvolumeclaims":
		k8sname = corev1.ResourcePersistentVolumeClaims
	case "services.nodeports":
		k8sname = corev1.ResourceServicesNodePorts
	case "services.loadbalancers":
		k8sname = corev1.ResourceServicesLoadBalancers
	case "requests.cpu":
		k8sname = corev1.ResourceRequestsCPU
	case "requests.memory":
		k8sname = corev1.ResourceRequestsMemory
	case "requests.storage":
		k8sname = corev1.ResourceRequestsStorage
	case "requests.ephemeral-storage":
		k8sname = corev1.ResourceRequestsEphemeralStorage
	case "limits.cpu":
		k8sname = corev1.ResourceLimitsCPU
	case "limits.memory":
		k8sname = corev1.ResourceLimitsMemory
	case "limits.ephemeral-storage":
		k8sname = corev1.ResourceLimitsEphemeralStorage
	default:
		err = fmt.Errorf("resoucequota resourcename %s isn`t supported", name)
	}
	return
}

func scQuotaScopeToK8sQuotaScope(scope string) (s corev1.ResourceQuotaScope, err error) {
	switch strings.ToLower(scope) {
	case "terminating":
		s = corev1.ResourceQuotaScopeTerminating
	case "notterminating":
		s = corev1.ResourceQuotaScopeNotTerminating
	case "besteffort":
		s = corev1.ResourceQuotaScopeBestEffort
	case "notbesteffort":
		s = corev1.ResourceQuotaScopeNotBestEffort
	case "priorityclass":
		s = corev1.ResourceQuotaScopePriorityClass
	default:
		err = fmt.Errorf("resoucequota scope %s isn`t supported", scope)
	}
	return
}

func scQuotaOperatorToK8sQuotaOperator(operator string) (op corev1.ScopeSelectorOperator, err error) {
	switch strings.ToLower(operator) {
	case "in":
		op = corev1.ScopeSelectorOpIn
	case "notin":
		op = corev1.ScopeSelectorOpNotIn
	case "exists":
		op = corev1.ScopeSelectorOpExists
	case "doesnotexist":
		op = corev1.ScopeSelectorOpDoesNotExist
	default:
		err = fmt.Errorf("resoucequota operator %s isn`t supported", operator)
	}
	return
}
