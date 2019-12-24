package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	resterror "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	restresource "github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/types"
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
	MailFromConfigName                    = "mailFrom"
	MailToConfigName                      = "mailTo"
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
	threshold, err := getClusterThreshold(cluster.KubeClient, ClusterThresholdConfigmapName)
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("list thresholds failed:%s", err.Error())
		}
		return nil
	}

	return []*types.ClusterThreshold{threshold}
}

func createClusterThreshold(cli client.Client, threshold *types.ClusterThreshold) error {
	if !checkPort(threshold.MailFrom.Port) {
		return errors.New("port must be numer")
	}
	sccm, err := clusterThresholdToConfigmap(threshold)
	if err != nil {
		return err
	}
	return createConfigMap(cli, ThresholdConfigmapNamespace, sccm)
}

func getClusterThreshold(cli client.Client, name string) (*types.ClusterThreshold, error) {
	cm, err := getConfigMap(cli, ThresholdConfigmapNamespace, name)
	if err != nil {
		return nil, err
	}
	sccm := k8sConfigMapToSCConfigMap(cm)
	threshold, err := configMapToClusterThreshold(sccm)
	if err != nil {
		return nil, err
	}
	threshold.SetID(sccm.Name)
	threshold.SetCreationTimestamp(cm.CreationTimestamp.Time)
	return threshold, nil
}

func updateClusterThreshold(cli client.Client, threshold *types.ClusterThreshold) error {
	if !checkPort(threshold.MailFrom.Port) {
		return errors.New("port must be numer")
	}
	sccm, err := clusterThresholdToConfigmap(threshold)
	if err != nil {
		return err
	}
	sccm.SetID(sccm.Name)
	return updateConfigMap(cli, ThresholdConfigmapNamespace, sccm)
}

func clusterThresholdToConfigmap(threshold *types.ClusterThreshold) (*types.ConfigMap, error) {
	mailFrom, err := json.Marshal(threshold.MailFrom)
	if err != nil {
		return nil, err
	}
	mailTo, err := json.Marshal(threshold.MailTo)
	if err != nil {
		return nil, err
	}
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
			types.Config{
				Name: MailFromConfigName,
				Data: string(mailFrom),
			},
			types.Config{
				Name: MailToConfigName,
				Data: string(mailTo),
			},
		},
	}, nil
}

func configMapToClusterThreshold(cm *types.ConfigMap) (*types.ClusterThreshold, error) {
	var threshold types.ClusterThreshold
	for _, cfg := range cm.Configs {
		switch cfg.Name {
		case CpuConfigName:
			cpu, err := strconv.Atoi(cfg.Data)
			if err != nil {
				return nil, err
			}
			threshold.Cpu = cpu
		case MemoryConfigName:
			memory, err := strconv.Atoi(cfg.Data)
			if err != nil {
				return nil, err
			}
			threshold.Memory = memory
		case StorageConfigName:
			storage, err := strconv.Atoi(cfg.Data)
			if err != nil {
				return nil, err
			}
			threshold.Storage = storage
		case PodCountConfigName:
			podcount, err := strconv.Atoi(cfg.Data)
			if err != nil {
				return nil, err
			}
			threshold.PodCount = podcount
		case NodeCpuConfigName:
			nodeCpu, err := strconv.Atoi(cfg.Data)
			if err != nil {
				return nil, err
			}
			threshold.NodeCpu = nodeCpu
		case NodeMemoryConfigName:
			nodeMemory, err := strconv.Atoi(cfg.Data)
			if err != nil {
				return nil, err
			}
			threshold.NodeMemory = nodeMemory
		case MailFromConfigName:
			var mailFrom types.Mail
			if err := json.Unmarshal([]byte(cfg.Data), &mailFrom); err != nil {
				return nil, err
			}
			threshold.MailFrom = mailFrom
		case MailToConfigName:
			var mailTo []string
			if err := json.Unmarshal([]byte(cfg.Data), &mailTo); err != nil {
				return nil, err
			}
			threshold.MailTo = mailTo
		}
	}
	return &threshold, nil
}

func checkPort(port string) bool {
	pattern := "^(\\d+)$"
	result, _ := regexp.MatchString(pattern, port)
	return result
}
