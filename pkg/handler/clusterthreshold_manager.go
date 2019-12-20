package handler

import (
	"fmt"
	"strconv"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	resterror "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	restresource "github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/types"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	ClusterThresholdConfigmapName         = "resource-threshold"
	NamespaceThresholdConfigmapNamePrefix = "resource-threshold-"
	ThresholdConfigmapNamespace           = "zcloud"
	CpuConfigName                         = "cpu"
	MemoryConfigName                      = "memory"
	StorageConfigName                     = "storage"
	PodCountConfigName                    = "podCount"
	PodStorageConfigName                  = "podStorage"
	NodeCpuConfigName                     = "nodeCpu"
	NodeMemoryConfigName                  = "nodeMemory"
)

type ClusterThresholdManager struct {
	clusters *ClusterManager
}

func newClusterThresholdManager(clusterMgr *ClusterManager) *ClusterThresholdManager {
	return &ClusterThresholdManager{
		clusters: clusterMgr,
	}
}

func (m *ClusterThresholdManager) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster s doesn't exist")
	}

	threshold := ctx.Resource.(*types.ClusterThreshold)
	if err := createClusterThreshold(cluster.KubeClient, threshold); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resterror.NewAPIError(resterror.DuplicateResource, fmt.Sprintf("duplicate threshold"))
		} else {
			return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create threshold failed %s", err.Error()))
		}
	}
	threshold.SetID(ClusterThresholdConfigmapName)
	return threshold, nil
}

func (m *ClusterThresholdManager) Delete(ctx *restresource.Context) *resterror.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	threshold := ctx.Resource.(*types.ClusterThreshold)
	err := deleteConfigMap(cluster.KubeClient, ThresholdConfigmapNamespace, threshold.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("threshold desn't exist "))
		} else {
			return resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete threshold failed %s", err.Error()))
		}
	}
	return nil
}
func (m *ClusterThresholdManager) Update(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	threshold := ctx.Resource.(*types.ClusterThreshold)
	if err := updateClusterThreshold(cluster.KubeClient, threshold); err != nil {
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("update threshold failed %s", err.Error()))
	} else {
		return threshold, nil
	}
}

func (m *ClusterThresholdManager) Get(ctx *restresource.Context) restresource.Resource {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	threshold := ctx.Resource.(*types.ClusterThreshold)
	threshold, err := getClusterThreshold(cluster.KubeClient, threshold.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("get threshold failed:%s", err.Error())
		}
		return nil
	}
	return threshold
}

func (m *ClusterThresholdManager) List(ctx *resource.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	return getClusterThresholds(cluster.KubeClient)
}

func createClusterThreshold(cli client.Client, threshold *types.ClusterThreshold) error {
	sccm := clusterThresholdToConfigmap(threshold)
	return createConfigMap(cli, ThresholdConfigmapNamespace, sccm)
}

func getClusterThreshold(cli client.Client, name string) (*types.ClusterThreshold, error) {
	cm, err := getConfigMap(cli, ThresholdConfigmapNamespace, name)
	if err != nil {
		return nil, err
	}
	sccm := k8sConfigMapToSCConfigMap(cm)
	threshold := configMapToClusterThreshold(sccm)
	threshold.SetID(sccm.Name)
	threshold.SetCreationTimestamp(cm.CreationTimestamp.Time)
	return threshold, nil
}

func getClusterThresholds(cli client.Client) []*types.ClusterThreshold {
	var thresholds []*types.ClusterThreshold
	threshold, err := getClusterThreshold(cli, ClusterThresholdConfigmapName)
	if err != nil {
		return nil
	}
	thresholds = append(thresholds, threshold)
	return thresholds
}

func updateClusterThreshold(cli client.Client, threshold *types.ClusterThreshold) error {
	cm, err := getConfigMap(cli, ThresholdConfigmapNamespace, threshold.GetID())
	if err != nil {
		return err
	}
	sccm := k8sConfigMapToSCConfigMap(cm)
	cfgs := make([]types.Config, 0)
	for _, cfg := range sccm.Configs {
		var data string
		switch cfg.Name {
		case CpuConfigName:
			data = strconv.Itoa(threshold.Cpu)
		case MemoryConfigName:
			data = strconv.Itoa(threshold.Memory)
		case StorageConfigName:
			data = strconv.Itoa(threshold.Storage)
		case PodCountConfigName:
			data = strconv.Itoa(threshold.PodCount)
		case NodeCpuConfigName:
			data = strconv.Itoa(threshold.NodeCpu)
		case NodeMemoryConfigName:
			data = strconv.Itoa(threshold.NodeMemory)
		}
		cfgs = append(cfgs, types.Config{
			Name: cfg.Name,
			Data: data,
		})
	}
	sccm.Configs = cfgs
	return updateConfigMap(cli, ThresholdConfigmapNamespace, sccm)
}

func clusterThresholdToConfigmap(threshold *types.ClusterThreshold) *types.ConfigMap {
	return &types.ConfigMap{
		Name: ClusterThresholdConfigmapName,
		Configs: []types.Config{
			types.Config{
				Name: CpuConfigName,
				Data: strconv.Itoa(threshold.Cpu),
			},
			types.Config{
				Name: MemoryConfigName,
				Data: strconv.Itoa(threshold.Memory),
			},
			types.Config{
				Name: StorageConfigName,
				Data: strconv.Itoa(threshold.Storage),
			},
			types.Config{
				Name: PodCountConfigName,
				Data: strconv.Itoa(threshold.PodCount),
			},
			types.Config{
				Name: NodeCpuConfigName,
				Data: strconv.Itoa(threshold.NodeCpu),
			},
			types.Config{
				Name: NodeMemoryConfigName,
				Data: strconv.Itoa(threshold.NodeMemory),
			},
		},
	}
}

func configMapToClusterThreshold(cm *types.ConfigMap) *types.ClusterThreshold {
	var threshold types.ClusterThreshold
	for _, cfg := range cm.Configs {
		switch cfg.Name {
		case CpuConfigName:
			cpu, _ := strconv.Atoi(cfg.Data)
			threshold.Cpu = cpu
		case MemoryConfigName:
			memory, _ := strconv.Atoi(cfg.Data)
			threshold.Memory = memory
		case StorageConfigName:
			storage, _ := strconv.Atoi(cfg.Data)
			threshold.Storage = storage
		case PodCountConfigName:
			podcount, _ := strconv.Atoi(cfg.Data)
			threshold.PodCount = podcount
		case NodeCpuConfigName:
			nodeCpu, _ := strconv.Atoi(cfg.Data)
			threshold.NodeCpu = nodeCpu
		case NodeMemoryConfigName:
			nodeMemory, _ := strconv.Atoi(cfg.Data)
			threshold.NodeMemory = nodeMemory
		}
	}
	return &threshold
}
