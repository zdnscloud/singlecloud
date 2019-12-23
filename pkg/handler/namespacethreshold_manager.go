package handler

import (
	"encoding/json"
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

type NamespaceThresholdManager struct {
	clusters *ClusterManager
}

func newNamespaceThresholdManager(clusterMgr *ClusterManager) *NamespaceThresholdManager {
	return &NamespaceThresholdManager{
		clusters: clusterMgr,
	}
}

func (m *NamespaceThresholdManager) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster s doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	name := NamespaceThresholdConfigmapNamePrefix + namespace
	threshold := ctx.Resource.(*types.NamespaceThreshold)
	if err := createNamespaceThreshold(cluster.KubeClient, name, threshold); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resterror.NewAPIError(resterror.DuplicateResource, fmt.Sprintf("duplicate threshold"))
		} else {
			return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create threshold failed %s", err.Error()))
		}
	}
	threshold.SetID(name)
	return threshold, nil
}

func (m *NamespaceThresholdManager) Delete(ctx *restresource.Context) *resterror.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	threshold := ctx.Resource.(*types.NamespaceThreshold)
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
func (m *NamespaceThresholdManager) Update(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	//namespace := ctx.Resource.GetParent().GetID()
	threshold := ctx.Resource.(*types.NamespaceThreshold)
	if err := updateNamespaceThreshold(cluster.KubeClient, threshold); err != nil {
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("update threshold failed %s", err.Error()))
	} else {
		return threshold, nil
	}
}

func (m *NamespaceThresholdManager) Get(ctx *restresource.Context) restresource.Resource {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	//namespace := ctx.Resource.GetParent().GetID()
	threshold := ctx.Resource.(*types.NamespaceThreshold)
	threshold, err := getNamespaceThreshold(cluster.KubeClient, threshold.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("get threshold failed:%s", err.Error())
		}
		return nil
	}
	return threshold
}

func (m *NamespaceThresholdManager) List(ctx *resource.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}
	namespace := ctx.Resource.GetParent().GetID()
	name := NamespaceThresholdConfigmapNamePrefix + namespace
	return getNamespaceThresholds(cluster.KubeClient, name)
}

func createNamespaceThreshold(cli client.Client, name string, threshold *types.NamespaceThreshold) error {
	sccm := namespaceThresholdToConfigmap(threshold, name)
	return createConfigMap(cli, ThresholdConfigmapNamespace, sccm)
}

func getNamespaceThreshold(cli client.Client, name string) (*types.NamespaceThreshold, error) {
	cm, err := getConfigMap(cli, ThresholdConfigmapNamespace, name)
	if err != nil {
		return nil, err
	}
	sccm := k8sConfigMapToSCConfigMap(cm)
	threshold := configMapToNamespaceThreshold(sccm)
	threshold.SetID(sccm.Name)
	threshold.SetCreationTimestamp(cm.CreationTimestamp.Time)
	return threshold, nil
}

func getNamespaceThresholds(cli client.Client, name string) []*types.NamespaceThreshold {
	var thresholds []*types.NamespaceThreshold
	threshold, err := getNamespaceThreshold(cli, name)
	if err != nil {
		return nil
	}
	thresholds = append(thresholds, threshold)
	return thresholds
}
func updateNamespaceThreshold(cli client.Client, threshold *types.NamespaceThreshold) error {
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
		case PodStorageConfigName:
			data = strconv.Itoa(threshold.PodStorage)
		case MailToConfigName:
			tmp, _ := json.Marshal(threshold.MailTo)
			data = string(tmp)
		}
		cfgs = append(cfgs, types.Config{
			Name: cfg.Name,
			Data: data,
		})
	}
	sccm.Configs = cfgs
	return updateConfigMap(cli, ThresholdConfigmapNamespace, sccm)
}

func namespaceThresholdToConfigmap(threshold *types.NamespaceThreshold, name string) *types.ConfigMap {
	mailTo, _ := json.Marshal(threshold.MailTo)
	return &types.ConfigMap{
		Name: name,
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
				Name: PodStorageConfigName,
				Data: strconv.Itoa(threshold.PodStorage),
			},
			types.Config{
				Name: MailToConfigName,
				Data: string(mailTo),
			},
		},
	}
}

func configMapToNamespaceThreshold(cm *types.ConfigMap) *types.NamespaceThreshold {
	var threshold types.NamespaceThreshold
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
		case PodStorageConfigName:
			podStorage, _ := strconv.Atoi(cfg.Data)
			threshold.PodStorage = podStorage
		case MailToConfigName:
			var mailTo []string
			json.Unmarshal([]byte(cfg.Data), &mailTo)
			threshold.MailTo = mailTo
		}
	}
	return &threshold
}
