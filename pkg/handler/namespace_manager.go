package handler

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	resterror "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/storage"
)

type NamespaceManager struct {
	clusters *ClusterManager
}

func newNamespaceManager(clusters *ClusterManager) *NamespaceManager {
	return &NamespaceManager{clusters: clusters}
}

func (m *NamespaceManager) Create(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resterror.NewAPIError(resterror.PermissionDenied, "only admin can create namespace")
	}

	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.(*types.Namespace)
	err := createNamespace(cluster.KubeClient, namespace.Name)
	if err == nil {
		namespace.SetID(namespace.Name)
		return namespace, nil
	}

	if apierrors.IsAlreadyExists(err) {
		return nil, resterror.NewAPIError(resterror.DuplicateResource, fmt.Sprintf("duplicate namespace name %s", namespace.Name))
	} else {
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create namespace failed %s", err.Error()))
	}
}

func (m *NamespaceManager) List(ctx *resource.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	k8sNamespaces, err := getNamespaces(cluster.KubeClient)
	if err != nil {
		log.Warnf("get namespace info failed:%s", err.Error())
		return nil
	}

	user := getCurrentUser(ctx)
	var namespaces []*types.Namespace
	for _, ns := range k8sNamespaces.Items {
		if m.clusters.authorizer.Authorize(user, cluster.Name, ns.Name) {
			namespaces = append(namespaces, k8sNamespaceToSCNamespace(&ns))
		}
	}
	return namespaces
}

func (m *NamespaceManager) Get(ctx *resource.Context) resource.Resource {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Resource.(*types.Namespace)
	if m.clusters.authorizer.Authorize(getCurrentUser(ctx), cluster.Name, namespace.GetID()) == false {
		return nil
	}

	k8sNamespace, err := getNamespace(cluster.KubeClient, namespace.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("get namespace info failed:%s", err.Error())
		}
		return nil
	}

	return k8sNamespaceToSCNamespace(k8sNamespace)
}

func (m *NamespaceManager) Delete(ctx *resource.Context) *resterror.APIError {
	if isAdmin(getCurrentUser(ctx)) == false {
		return resterror.NewAPIError(resterror.PermissionDenied, "only admin can delete namespace")
	}

	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.(*types.Namespace)
	exits, err := IsExistsNamespaceInDB(m.clusters.GetDB(), storage.GenTableName(UserQuotaTable), namespace.GetID())
	if err != nil {
		return resterror.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("check exist for namespace %s failed %s", namespace.GetID(), err.Error()))
	}

	if exits {
		return resterror.NewAPIError(resterror.PermissionDenied,
			fmt.Sprintf("can`t delete namespace %s for other user using", namespace.GetID()))
	}

	if err := clearApplications(m.clusters.GetDB(), cluster.KubeClient, cluster.Name, namespace.GetID()); err != nil {
		if apierrors.IsNotFound(err) == false {
			return resterror.NewAPIError(types.ConnectClusterFailed,
				fmt.Sprintf("delete namespace applications failed: %s", err.Error()))
		}
	}

	if err := deleteNamespace(cluster.KubeClient, namespace.GetID()); err != nil {
		if apierrors.IsNotFound(err) {
			return resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("namespace %s desn't exist", namespace.Name))
		} else {
			return resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete namespace failed %s", err.Error()))
		}
	} else {
		/*
			if err := clearTransportLayerIngress(cluster.KubeClient, namespace.GetID(), types.IngressProtocolUDP); err != nil {
				log.Warnf("clean udp ingress for namespace %s failed:%s", namespace.GetID(), err.Error())
			}
		*/
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
	ns.SetCreationTimestamp(k8sNamespace.CreationTimestamp.Time)
	return ns
}
