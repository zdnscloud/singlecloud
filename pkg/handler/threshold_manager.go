package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	//"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	resterror "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	restresource "github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/kvzoo"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	ThresholdTable              = "threshold"
	ThresholdConfigmapName      = "resource-threshold"
	ThresholdConfigmapNamespace = "zcloud"
	CpuConfigName               = "cpu"
	MemoryConfigName            = "memory"
	StorageConfigName           = "storage"
	PodCountConfigName          = "podCount"
	MailFromConfigName          = "mailFrom"
	MailToConfigName            = "mailTo"
)

type ThresholdManager struct {
	clusters *ClusterManager
}

func newThresholdManager(clusterMgr *ClusterManager) *ThresholdManager {
	return &ThresholdManager{
		clusters: clusterMgr,
	}
}

func (m *ThresholdManager) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resterror.NewAPIError(resterror.PermissionDenied, "only admin can create threshold")
	}
	threshold := ctx.Resource.(*types.Threshold)
	if err := createThreshold(m.clusters, threshold); err != nil {
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create threshold failed %s", err.Error()))
	}

	return threshold, nil
}

func (m *ThresholdManager) Delete(ctx *restresource.Context) *resterror.APIError {
	if isAdmin(getCurrentUser(ctx)) == false {
		return resterror.NewAPIError(resterror.PermissionDenied, "only admin can delete threshold")
	}
	if err := deleteThreshold(m.clusters, ctx.Resource.GetID()); err != nil {
		return resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete threshold failed %s", err.Error()))
	}
	return nil
}

func (m *ThresholdManager) Update(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resterror.NewAPIError(resterror.PermissionDenied, "only admin can update threshold")
	}
	threshold := ctx.Resource.(*types.Threshold)

	if err := updateThreshold(m.clusters, threshold); err != nil {
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("update threshold failed %s", err.Error()))
	}
	return threshold, nil
}

func (m *ThresholdManager) Get(ctx *restresource.Context) restresource.Resource {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil
	}
	threshold, err := getThreshold(m.clusters, ctx.Resource.GetID())
	if err != nil {
		return nil
	}
	return threshold
}

func (m *ThresholdManager) List(ctx *resource.Context) interface{} {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil
	}
	threshold, err := getThreshold(m.clusters, ThresholdConfigmapName)
	if err != nil {
		return nil
	}
	return []*types.Threshold{threshold}
}

func createThreshold(clusters *ClusterManager, threshold *types.Threshold) error {
	if !checkPort(threshold.MailFrom.Port) {
		return errors.New("port must be numer")
	}
	table, _, err := createOrGetThresholdTable(clusters.GetDB())
	if err != nil {
		return err
	}
	threshold.Status = types.ThresholdActive
	threshold.SetID(ThresholdConfigmapName)
	threshold.SetCreationTimestamp(time.Now())
	if err := addThresholdToDB(table, threshold); err != nil {
		return err
	}
	for _, c := range clusters.zkeManager.List() {
		if err := createThresholdConfigMap(c.KubeClient, threshold); err != nil {
			go updateThresholdStatusToInactiveInDB(table, threshold.GetID())
			return fmt.Errorf("create threshold in cluster %s failed: %v", c.Name, err)
		}
	}
	return nil
}

func createOrGetThresholdTable(db kvzoo.DB) (kvzoo.Table, kvzoo.TableName, error) {
	tn, _ := kvzoo.TableNameFromSegments(ThresholdTable)
	table, err := db.CreateOrGetTable(tn)
	if err != nil {
		return nil, tn, fmt.Errorf("create or get table %s failed: %s", tn, err.Error())
	}

	return table, tn, nil
}

func addThresholdToDB(table kvzoo.Table, threshold *types.Threshold) error {
	value, err := json.Marshal(threshold)
	if err != nil {
		return fmt.Errorf("marshal threshold %s failed: %s", threshold.GetID(), err.Error())
	}

	tx, err := table.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction failed: %s", err.Error())
	}

	defer tx.Rollback()
	if err = tx.Add(threshold.GetID(), value); err != nil {
		return err
	}

	return tx.Commit()
}

func createThresholdConfigMap(cli client.Client, threshold *types.Threshold) error {
	sccm, err := thresholdToConfigmap(threshold)
	if err != nil {
		return err
	}
	return createConfigMap(cli, ThresholdConfigmapNamespace, sccm)
}

func updateThresholdStatusToInactiveInDB(table kvzoo.Table, name string) error {
	tx, err := table.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()
	threshold, err := getThresholdFromDBTx(tx, name)
	if err != nil {
		return err
	}

	threshold.Status = types.ThresholdInActive
	value, err := json.Marshal(threshold)
	if err != nil {
		return err
	}

	if err := tx.Update(name, value); err != nil {
		return err
	}

	return tx.Commit()
}

func getThresholdFromDBTx(tx kvzoo.Transaction, name string) (*types.Threshold, error) {
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

func deleteThreshold(clusters *ClusterManager, name string) error {
	table, _, err := createOrGetThresholdTable(clusters.GetDB())
	if err != nil {
		return err
	}
	for _, c := range clusters.zkeManager.List() {
		if err := deleteConfigMap(c.KubeClient, ThresholdConfigmapNamespace, name); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			go updateThresholdStatusToInactiveInDB(table, name)
			return fmt.Errorf("delete threshold in cluster %s failed: %v", c.Name, err)
		}
	}
	return delThresholdFromDB(table, name)
}

func delThresholdFromDB(table kvzoo.Table, name string) error {
	tx, err := table.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()
	if err := tx.Delete(name); err != nil {
		return err
	}

	return tx.Commit()
}

func updateThreshold(clusters *ClusterManager, threshold *types.Threshold) error {
	if !checkPort(threshold.MailFrom.Port) {
		return errors.New("port must be numer")
	}
	table, _, err := createOrGetThresholdTable(clusters.GetDB())
	if err != nil {
		return err
	}
	for _, c := range clusters.zkeManager.List() {
		if _, err := getConfigMap(c.KubeClient, ThresholdConfigmapNamespace, threshold.GetID()); apierrors.IsNotFound(err) {
			if err := createThresholdConfigMap(c.KubeClient, threshold); err != nil {
				return fmt.Errorf("cluster %s doesn't have threshold, create it first but failed: %v", c.Name, err)
			}
			continue
		}
		if err := updateThresholdConfigMap(c.KubeClient, threshold); err != nil {
			go updateThresholdStatusToInactiveInDB(table, threshold.GetID())
			return fmt.Errorf("update threshold in cluster %s failed: %v", c.Name, err)
		}
	}
	threshold.Status = types.ThresholdActive
	return updateThresholdInDB(table, threshold)
}

func updateThresholdConfigMap(cli client.Client, threshold *types.Threshold) error {
	sccm, err := thresholdToConfigmap(threshold)
	if err != nil {
		return err
	}
	sccm.SetID(sccm.Name)
	return updateConfigMap(cli, ThresholdConfigmapNamespace, sccm)
}

func updateThresholdInDB(table kvzoo.Table, threshold *types.Threshold) error {
	tx, err := table.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction failed: %s", err.Error())
	}
	defer tx.Rollback()
	name := threshold.GetID()
	oldThreshold, err := getThresholdFromDBTx(tx, name)
	if err != nil {
		return err
	}
	threshold.SetCreationTimestamp(time.Time(oldThreshold.CreationTimestamp))
	threshold.SetDeletionTimestamp(time.Time(oldThreshold.DeletionTimestamp))
	value, err := json.Marshal(threshold)
	if err != nil {
		return fmt.Errorf("marshal threshold %s failed: %s", threshold.GetID(), err.Error())
	}

	if err = tx.Update(name, value); err != nil {
		return err
	}

	return tx.Commit()
}

func getThreshold(clusters *ClusterManager, name string) (*types.Threshold, error) {
	table, _, err := createOrGetThresholdTable(clusters.GetDB())
	if err != nil {
		return nil, err
	}
	return getThresholdFromDB(table, name)
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

func thresholdToConfigmap(threshold *types.Threshold) (*types.ConfigMap, error) {
	mailFrom, err := json.Marshal(threshold.MailFrom)
	if err != nil {
		return nil, err
	}
	mailTo, err := json.Marshal(threshold.MailTo)
	if err != nil {
		return nil, err
	}
	return &types.ConfigMap{
		Name: ThresholdConfigmapName,
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

func checkPort(port string) bool {
	if len(port) == 0 {
		return true
	}
	pattern := "^(\\d+)$"
	result, _ := regexp.MatchString(pattern, port)
	return result
}
