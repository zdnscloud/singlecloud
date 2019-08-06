package handler

import (
	"context"
	"fmt"

	"encoding/json"
	"io/ioutil"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"net/http"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	storagev1 "github.com/zdnscloud/immense/pkg/apis/zcloud/v1"
	"github.com/zdnscloud/singlecloud/pkg/clusteragent"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type StorageClusterManager struct {
	api.DefaultHandler
	clusters *ClusterManager
}

func newStorageClusterManager(clusters *ClusterManager) *StorageClusterManager {
	return &StorageClusterManager{
		clusters: clusters,
	}
}

func (m *StorageClusterManager) List(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	k8sStorageClusters, err := getStorageClusters(cluster.KubeClient)
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("list storagecluster info failed:%s", err.Error())
		}
		return nil
	}

	var storageclusters []*types.StorageCluster
	for _, item := range k8sStorageClusters.Items {
		storageclusters = append(storageclusters, k8sStorageToSCStorage(cluster, m.clusters.Agent, &item))
	}
	return storageclusters
}

func (m StorageClusterManager) Get(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	storagecluster := ctx.Object.(*types.StorageCluster)
	k8sStorageCluster, err := getStorageCluster(cluster.KubeClient, storagecluster.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("get storagecluster info failed:%s", err.Error())
		}
		return nil
	}

	return k8sStorageToSCStorage(cluster, m.clusters.Agent, k8sStorageCluster)
}

func (m StorageClusterManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	storagecluster := ctx.Object.(*types.StorageCluster)
	err := deleteStorageCluster(cluster.KubeClient, storagecluster.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("storagecluster %s doesn't exist", storagecluster.GetID()))
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

	storagecluster := ctx.Object.(*types.StorageCluster)
	if err := createStorageCluster(cluster.KubeClient, storagecluster); err != nil {
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

	storagecluster := ctx.Object.(*types.StorageCluster)
	if err := updateStorageCluster(cluster.KubeClient, storagecluster); err != nil {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("update storagecluster failed %s", err.Error()))
	} else {
		return storagecluster, nil
	}
}

func getStorageCluster(cli client.Client, name string) (*storagev1.Cluster, error) {
	storagecluster := storagev1.Cluster{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{"", name}, &storagecluster)
	return &storagecluster, err
}

func getStorageClusters(cli client.Client) (*storagev1.ClusterList, error) {
	storageclusters := storagev1.ClusterList{}
	err := cli.List(context.TODO(), nil, &storageclusters)
	return &storageclusters, err
}

func deleteStorageCluster(cli client.Client, name string) error {
	storagecluster := &storagev1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	return cli.Delete(context.TODO(), storagecluster)
}

func createStorageCluster(cli client.Client, storagecluster *types.StorageCluster) error {
	k8sStorageCluster := scStorageToK8sStorage(storagecluster)
	return cli.Create(context.TODO(), k8sStorageCluster)
}

func updateStorageCluster(cli client.Client, storagecluster *types.StorageCluster) error {
	k8sStorageCluster, err := getStorageCluster(cli, storagecluster.GetID())
	if err != nil {
		return err
	}
	k8sStorageCluster.Spec.Hosts = storagecluster.Hosts
	return cli.Update(context.TODO(), k8sStorageCluster)
}

func scStorageToK8sStorage(storagecluster *types.StorageCluster) *storagev1.Cluster {
	return &storagev1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: storagecluster.Name,
		},
		Spec: storagev1.ClusterSpec{
			StorageType: storagecluster.StorageType,
			Hosts:       storagecluster.Hosts,
		},
	}
}

func k8sStorageToSCStorage(cluster *Cluster, agent *clusteragent.AgentManager, k8sStorageCluster *storagev1.Cluster) *types.StorageCluster {
	info, err := getStatusInfo(cluster.Name, agent, k8sStorageCluster.Spec.StorageType)
	if err != nil {
		log.Warnf("get clusterinfo from clusteragent failed:%s", err.Error())
	}
	storagecluster := &types.StorageCluster{
		Name:        k8sStorageCluster.Name,
		StorageType: k8sStorageCluster.Spec.StorageType,
		Hosts:       k8sStorageCluster.Spec.Hosts,
		Phase:       k8sStorageCluster.Status.Phase,
		Size:        info.Size,
		UsedSize:    info.UsedSize,
		FreeSize:    info.FreeSize,
		Nodes:       info.Nodes,
		PVs:         info.PVs,
	}
	storagecluster.SetID(k8sStorageCluster.Name)
	storagecluster.SetType(types.StorageClusterType)
	storagecluster.SetCreationTimestamp(k8sStorageCluster.CreationTimestamp.Time)
	return storagecluster
}

func getStatusInfo(cluster string, agent *clusteragent.AgentManager, storagetype string) (types.Storage, error) {
	var info types.Storage
	url := "/apis/agent.zcloud.cn/v1/storages/"
	req, err := http.NewRequest("GET", clusteragent.ClusterAgentServiceHost+url+storagetype, nil)
	if err != nil {
		return info, err
	}
	resp, err := agent.ProxyRequest(cluster, req)
	if err != nil {
		return info, err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &info)
	return info, nil
}
