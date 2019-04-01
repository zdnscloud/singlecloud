package handler

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
	DefaultHandler
	clusters *ClusterManager
}

func newNamespaceManager(clusters *ClusterManager) *NamespaceManager {
	return &NamespaceManager{clusters: clusters}
}

func (m *NamespaceManager) Create(obj resttypes.Object, yamlConf []byte) (interface{}, *resttypes.APIError) {
	cluster := m.clusters.GetClusterForSubResource(obj)
	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, "cluster s doesn't exist")
	}

	namespace := obj.(*types.Namespace)
	err := createNamespace(cluster.KubeClient, namespace.Name)
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

func (m *NamespaceManager) List(obj resttypes.Object) interface{} {
	cluster := m.clusters.GetClusterForSubResource(obj)
	if cluster == nil {
		return nil
	}

	k8sNamespaces, err := getNamespaces(cluster.KubeClient)
	if err != nil {
		logger.Warn("get namespace info failed:%s", err.Error())
		return nil
	}

	var namespaces []*types.Namespace
	for _, ns := range k8sNamespaces.Items {
		namespaces = append(namespaces, k8sNamespaceToSCNamespace(&ns))
	}
	return namespaces
}

func (m *NamespaceManager) Get(obj resttypes.Object) interface{} {
	cluster := m.clusters.GetClusterForSubResource(obj)
	if cluster == nil {
		return nil
	}

	namespace := obj.(*types.Namespace)
	k8sNamespace, err := getNamespace(cluster.KubeClient, namespace.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			logger.Warn("get namespace info failed:%s", err.Error())
		}
		return nil
	}

	return k8sNamespaceToSCNamespace(k8sNamespace)
}

func (m *NamespaceManager) Delete(obj resttypes.Object) *resttypes.APIError {
	cluster := m.clusters.GetClusterForSubResource(obj)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	namespace := obj.(*types.Namespace)
	err := deleteNamespace(cluster.KubeClient, namespace.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("namespace %s desn't exist", namespace.Name))
		} else {
			return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete namespace failed %s", err.Error()))
		}
	}
	return nil
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

func deleteNamespace(cli client.Client, name string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	return cli.Delete(context.TODO(), ns)
}

func getNamespace(cli client.Client, name string) (*corev1.Namespace, error) {
	ns := corev1.Namespace{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{"", name}, &ns)
	return &ns, err
}

func hasNamespace(cli client.Client, name string) (bool, error) {
	_, err := getNamespace(cli, name)
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
