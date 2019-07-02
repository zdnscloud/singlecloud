package handler

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	storagev1 "github.com/zdnscloud/immense/pkg/apis/zcloud/v1"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	StorageClusterNamespace = "zcloud"
)

type StorageClusterManager struct {
	api.DefaultHandler
	clusters *ClusterManager
}

func newStorageClusterManager(clusters *ClusterManager) *StorageClusterManager {
	return &StorageClusterManager{clusters: clusters}
}

func (m *StorageClusterManager) List(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Object.GetParent().GetID()
	k8sStorageClusters, err := getStorageClusters(cluster.KubeClient, namespace)
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("list storagecluster info failed:%s", err.Error())
		}
		return nil
	}

	var storageclusters []*types.StorageCluster
	for _, item := range k8sStorageClusters.Items {
		storageclusters = append(storageclusters, k8sStorageToSCStorage(&item))
	}
	return storageclusters
}

func (m StorageClusterManager) Get(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Object.GetParent().GetID()
	storagecluster := ctx.Object.(*types.StorageCluster)
	k8sStorageCluster, err := getStorageCluster(cluster.KubeClient, namespace, storagecluster.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("get storagecluster info failed:%s", err.Error())
		}
		return nil
	}

	return k8sStorageToSCStorage(k8sStorageCluster)
}

func (m StorageClusterManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	//namespace := ctx.Object.GetParent().GetID()
	storagecluster := ctx.Object.(*types.StorageCluster)
	err := deleteStorageCluster(cluster.KubeClient, storagecluster.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return resttypes.NewAPIError(resttypes.NotFound,
				fmt.Sprintf("storagecluster %s with namespace %s doesn't exist", storagecluster.GetID(), StorageClusterNamespace))
		} else {
			return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete storagecluster failed %s", err.Error()))
		}
	}
	return nil
}

func (m StorageClusterManager) Create(ctx *resttypes.Context, yamlConf []byte) (interface{}, *resttypes.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	storagecluster := ctx.Object.(*types.StorageCluster)
	if err := createStorageCluster(cluster.KubeClient, namespace, storagecluster); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resttypes.NewAPIError(resttypes.DuplicateResource, fmt.Sprintf("duplicate storagecluster name %s", storagecluster.Name))
		} else {
			return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create storagecluster failed %s", err.Error()))
		}
	}
	storagecluster.SetID(storagecluster.Name)
	return storagecluster, nil
}

func (m StorageClusterManager) Update(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	storagecluster := ctx.Object.(*types.StorageCluster)
	if err := updateStorageCluster(cluster.KubeClient, namespace, storagecluster); err != nil {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("update storagecluster failed %s", err.Error()))
	} else {
		return storagecluster, nil
	}
}

func getStorageCluster(cli client.Client, namespace, name string) (*storagev1.Cluster, error) {
	storagecluster := storagev1.Cluster{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, name}, &storagecluster)
	return &storagecluster, err
}

func getStorageClusters(cli client.Client, namespace string) (*storagev1.ClusterList, error) {
	storageclusters := storagev1.ClusterList{}
	err := cli.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &storageclusters)
	return &storageclusters, err
}

func deleteStorageCluster(cli client.Client, name string) error {
	storagecluster := &storagev1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: StorageClusterNamespace},
	}
	return cli.Delete(context.TODO(), storagecluster)
}

func createStorageCluster(cli client.Client, namespace string, storagecluster *types.StorageCluster) error {
	k8sStorageCluster := scStorageToK8sStorage(storagecluster)
	return cli.Create(context.TODO(), k8sStorageCluster)
}

func updateStorageCluster(cli client.Client, namespace string, storagecluster *types.StorageCluster) error {
	k8sStorageCluster, err := getStorageCluster(cli, namespace, storagecluster.GetID())
	if err != nil {
		return err
	}
	hosts := make([]storagev1.HostSpec, 0)
	for _, h := range storagecluster.Hosts {
		host := storagev1.HostSpec{
			NodeName:     h.NodeName,
			BlockDevices: h.BlockDevices,
		}
		hosts = append(hosts, host)
	}
	k8sStorageCluster.Spec.Hosts = hosts
	return cli.Update(context.TODO(), k8sStorageCluster)
}

func scStorageToK8sStorage(storagecluster *types.StorageCluster) *storagev1.Cluster {
	hosts := make([]storagev1.HostSpec, 0)
	for _, h := range storagecluster.Hosts {
		host := storagev1.HostSpec{
			NodeName:     h.NodeName,
			BlockDevices: h.BlockDevices,
		}
		hosts = append(hosts, host)
	}
	return &storagev1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      storagecluster.Name,
			Namespace: StorageClusterNamespace,
		},
		Spec: storagev1.ClusterSpec{
			StorageType: storagecluster.StorageType,
			Hosts:       hosts,
		},
	}
}

func k8sStorageToSCStorage(k8sStorageCluster *storagev1.Cluster) *types.StorageCluster {
	hosts := make([]types.HostSpec, 0)
	for _, h := range k8sStorageCluster.Spec.Hosts {
		host := types.HostSpec{
			NodeName:     h.NodeName,
			BlockDevices: h.BlockDevices,
		}
		hosts = append(hosts, host)
	}

	storagecluster := &types.StorageCluster{
		Name: k8sStorageCluster.Name,
		//Namespace:   StorageClusterNamespace,
		StorageType: k8sStorageCluster.Spec.StorageType,
		Hosts:       hosts,
		Status:      k8sStorageCluster.Status.CephStatus.Health,
	}
	storagecluster.SetID(k8sStorageCluster.Name)
	storagecluster.SetType(types.StorageClusterType)
	storagecluster.SetCreationTimestamp(k8sStorageCluster.CreationTimestamp.Time)
	return storagecluster
}
