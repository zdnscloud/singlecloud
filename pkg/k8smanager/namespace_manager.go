package k8smanager

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/gok8s/client"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/logger"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type NamespaceManager struct {
	cluster *types.Cluster
}

func newNamespaceManager(cluster *types.Cluster) NamespaceManager {
	return NamespaceManager{cluster: cluster}
}

func (m NamespaceManager) Create(namespace *types.Namespace, yamlConf []byte) (interface{}, *resttypes.APIError) {
	err := createNamespace(m.cluster.KubeClient, namespace.Name)
	if err == nil {
		namespace.SetID(namespace.Name)
		return namespace, nil
	}

	if apierrors.IsAlreadyExists(err) {
		return nil, resttypes.NewAPIError(resttypes.DuplicateResource, fmt.Sprintf("duplicate namespace name %s", namespace.Name))
	} else {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create namespace failed %s", err.Error()))
	}
}

func (m NamespaceManager) List() interface{} {
	k8sNamespaces, err := getNamespaces(m.cluster.KubeClient)
	if err != nil {
		logger.Warn("get node info failed:%s", err.Error())
		return nil
	}

	var namespaces []*types.Namespace
	for _, ns := range k8sNamespaces.Items {
		namespaces = append(namespaces, k8sNamespaceToSCNamespace(&ns))
	}
	return namespaces
}

func getNamespaces(cli client.Client) (*corev1.NamespaceList, error) {
	namespaces := corev1.NamespaceList{}
	err := cli.List(context.TODO(), nil, &namespaces)
	return &namespaces, err
}

func createNamespace(cli client.Client, name string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	return cli.Create(context.TODO(), ns)
}

func hasNamespace(cli client.Client, name string) (bool, error) {
	ns := corev1.Namespace{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{"", name}, &ns)
	if err == nil {
		return true, nil
	} else if apierrors.IsNotFound(err) {
		return false, nil
	} else {
		return false, err
	}
}

func k8sNamespaceToSCNamespace(k8sNamespace *corev1.Namespace) *types.Namespace {
	ns := &types.Namespace{
		Name: k8sNamespace.Name,
	}
	ns.SetID(k8sNamespace.Name)
	ns.SetType(types.NamespaceType)
	ns.SetCreationTimestamp(k8sNamespace.CreationTimestamp.Time)
	return ns
}
