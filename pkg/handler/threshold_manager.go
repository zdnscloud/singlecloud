package handler

import (
	"encoding/json"
	"fmt"
	"strconv"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/zdnscloud/cement/log"
	resterror "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	restresource "github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/kvzoo"
	"github.com/zdnscloud/singlecloud/pkg/alarm"
	"github.com/zdnscloud/singlecloud/pkg/eventbus"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/pkg/zke"
)

const (
	CpuConfigName      = "cpu"
	MemoryConfigName   = "memory"
	StorageConfigName  = "storage"
	PodCountConfigName = "podCount"
	DefaultCpu         = 80
	DefaultMemory      = 80
	DefaultStorage     = 80
	DefaultPodCount    = 80
)

type ThresholdManager struct {
	clusters       *ClusterManager
	clusterEventCh <-chan interface{}
	table          kvzoo.Table
	threshold      *types.Threshold
}

func newThresholdManager(clusters *ClusterManager) (*ThresholdManager, error) {
	m := &ThresholdManager{
		clusters:       clusters,
		clusterEventCh: clusters.GetEventBus().Sub(eventbus.ClusterEvent),
	}
	if err := m.initThreshold(); err != nil {
		return nil, err
	}
	go m.eventLoop()
	return m, nil
}

func (m *ThresholdManager) initThreshold() error {
	tn, _ := kvzoo.TableNameFromSegments(types.ThresholdTable)
	table, err := m.clusters.GetDB().CreateOrGetTable(tn)
	if err != nil {
		return fmt.Errorf("create or get table %s failed: %s", types.ThresholdTable, err.Error())
	}
	m.table = table
	var threshold *types.Threshold
	if threshold, err = getThresholdFromDB(table, types.ThresholdTable); err != nil {
		if err == kvzoo.ErrNotFound {
			if threshold, err = createDefaultThreshold(table, types.ThresholdTable); err != nil {
				return fmt.Errorf("create default threshold failed: %s", err.Error())
			}
		} else {
			return fmt.Errorf("get threshold from DB failed: %s", err.Error())
		}
	}
	m.threshold = threshold
	return nil
}

func getThresholdFromDB(table kvzoo.Table, name string) (*types.Threshold, error) {
	tx, err := table.Begin()
	if err != nil {
		return nil, err
	}

	defer tx.Commit()
	value, err := tx.Get(name)
	if err != nil {
		return nil, err
	}
	var threshold types.Threshold
	if err := json.Unmarshal(value, &threshold); err != nil {
		return nil, err
	}
	return &threshold, nil
}

func createDefaultThreshold(table kvzoo.Table, name string) (*types.Threshold, error) {
	threshold := &types.Threshold{
		Cpu:      DefaultCpu,
		Memory:   DefaultMemory,
		Storage:  DefaultStorage,
		PodCount: DefaultPodCount,
	}
	threshold.SetID(name)
	if err := addOrUpdateThresholdToDB(table, threshold, "add"); err != nil {
		return nil, err
	}
	return threshold, nil
}

func addOrUpdateThresholdToDB(table kvzoo.Table, threshold *types.Threshold, action string) error {
	value, err := json.Marshal(threshold)
	if err != nil {
		return fmt.Errorf("marshal threshold %s failed: %s", threshold.GetID(), err.Error())
	}

	tx, err := table.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction failed: %s", err.Error())
	}

	defer tx.Rollback()
	switch action {
	case "add":
		if err = tx.Add(threshold.GetID(), value); err != nil {
			return err
		}
	case "update":
		if err = tx.Update(threshold.GetID(), value); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (m *ThresholdManager) eventLoop() {
	for {
		event := <-m.clusterEventCh
		switch e := event.(type) {
		case zke.AddCluster:
			cluster := e.Cluster
			if err := createConfigMap(cluster.KubeClient, ZCloudNamespace, thresholdToConfigmap(m.threshold)); err != nil {
				log.Warnf("create configmap in cluster %s failed for threshold: %s", cluster.Name, err.Error())
				alarm.New().
					Cluster(cluster.Name).
					Namespace(ZCloudNamespace).
					Kind("Threshold").
					Name(m.threshold.GetID()).
					Reason(err.Error()).
					Message(fmt.Sprintf("failed to apply threshold to cluster %s", cluster.Name)).
					Publish()
			}
		}
	}
}

func (m *ThresholdManager) Update(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resterror.NewAPIError(resterror.PermissionDenied, "only admin can update threshold")
	}
	m.threshold = ctx.Resource.(*types.Threshold)

	if err := updateThreshold(m.clusters, m.threshold, m.table); err != nil {
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("update threshold failed %s", err.Error()))
	}
	return m.threshold, nil
}

func updateThreshold(clusters *ClusterManager, threshold *types.Threshold, table kvzoo.Table) error {
	if err := addOrUpdateThresholdToDB(table, threshold, "update"); err != nil {
		return err
	}
	for _, c := range clusters.zkeManager.List() {
		sccm := thresholdToConfigmap(threshold)
		sccm.SetID(sccm.Name)
		if err := updateConfigMap(c.KubeClient, ZCloudNamespace, sccm); err != nil {
			if apierrors.IsNotFound(err) {
				if err := createConfigMap(c.KubeClient, ZCloudNamespace, sccm); err != nil {
					return fmt.Errorf("cluster %s doesn't have threshold, create it first but failed: %v", c.Name, err)
				}
			} else {
				return fmt.Errorf("update threshold in cluster %s failed: %v", c.Name, err)
			}
		}
	}
	return nil
}

func (m *ThresholdManager) Get(ctx *restresource.Context) restresource.Resource {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil
	}
	return m.threshold
}

func (m *ThresholdManager) List(ctx *restresource.Context) interface{} {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil
	}
	return []*types.Threshold{m.threshold}
}

func thresholdToConfigmap(threshold *types.Threshold) *types.ConfigMap {
	return &types.ConfigMap{
		Name: types.ThresholdTable,
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
		},
	}
}
