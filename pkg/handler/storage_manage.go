package handler

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	resterr "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	storagev1 "github.com/zdnscloud/immense/pkg/apis/zcloud/v1"
	"github.com/zdnscloud/singlecloud/pkg/clusteragent"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/pkg/zke"
)

const (
	StorageClassDefaultKey  = "storageclass.kubernetes.io/is-default-class"
	StorageTable            = "storage"
	StorageNotFoundErr      = "can not found storage %s"
	StorageParameterNullErr = "parameter can not be null for storage %s"
)

type StorageHandle interface {
	GetType() types.StorageType
	GetStorages(cli client.Client) ([]*types.Storage, error)
	GetStorage(cluster *zke.Cluster, name string) (*types.Storage, error)
	Delete(cli client.Client, name string) error
	Create(cluster *zke.Cluster, storage *types.Storage) error
	Update(cluster *zke.Cluster, storage *types.Storage) error
}

type StorageManager struct {
	clusters       *ClusterManager
	storageHandles []StorageHandle
}

func newStorageManager(clusters *ClusterManager) *StorageManager {
	return &StorageManager{
		clusters: clusters,
		storageHandles: []StorageHandle{
			&LvmManager{},
			&CephFsManager{},
			&IscsiManager{},
			&NfsManager{}},
	}
}

func (m *StorageManager) List(ctx *resource.Context) (interface{}, *resterr.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, "cluster doesn't exist")
	}

	storages, err := m.getStorages(cluster.GetKubeClient())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterr.NewAPIError(resterr.NotFound, "no found storage")
		}
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("list storag failed %s", err.Error()))
	}
	return storages, nil
}

func (m *StorageManager) getStorages(cli client.Client) ([]*types.Storage, error) {
	var storages types.Storages
	for _, handle := range m.storageHandles {
		_storages, err := handle.GetStorages(cli)
		if err != nil {
			return nil, err
		}
		storages = append(storages, _storages...)
	}
	sort.Sort(storages)
	return storages, nil
}

func (m *StorageManager) getHandleFromType(typ types.StorageType) (StorageHandle, error) {
	for _, handle := range m.storageHandles {
		if typ == handle.GetType() {
			return handle, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("undefiend storage typ %s", string(typ)))
}

func (m *StorageManager) Get(ctx *resource.Context) (resource.Resource, *resterr.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, "cluster doesn't exist")
	}
	name := ctx.Resource.(*types.Storage).GetID()
	storage, err := m.getStorage(cluster, name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterr.NewAPIError(resterr.NotFound, fmt.Sprintf("no found storage %s", name))
		}
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("get storage failed %s", err.Error()))
	}
	return storage, nil
}

func (m *StorageManager) getStorage(cluster *zke.Cluster, name string) (*types.Storage, error) {
	for _, handle := range m.storageHandles {
		storage, err := handle.GetStorage(cluster, name)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return nil, err
		}
		return storage, nil
	}
	return nil, errors.New(fmt.Sprintf(StorageNotFoundErr, name))
}

func (m *StorageManager) Delete(ctx *resource.Context) *resterr.APIError {
	if isAdmin(getCurrentUser(ctx)) == false {
		return resterr.NewAPIError(resterr.PermissionDenied, "only admin can delete nfs")
	}
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterr.NewAPIError(resterr.NotFound, "storage doesn't exist")
	}
	storage := ctx.Resource.(*types.Storage)
	if err := m.deleteStorage(cluster, storage.GetID()); err != nil {
		if apierrors.IsNotFound(err) {
			return resterr.NewAPIError(resterr.NotFound, fmt.Sprintf("storage %s doesn't exist", storage.GetID()))
		} else {
			return resterr.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete storage failed, %s", err.Error()))
		}
	}
	return nil
}

func (m *StorageManager) deleteStorage(cluster *zke.Cluster, name string) error {
	for _, handle := range m.storageHandles {
		_, err := handle.GetStorage(cluster, name)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return err
		}
		return handle.Delete(cluster.GetKubeClient(), name)
	}
	return errors.New(fmt.Sprintf(StorageNotFoundErr, name))
}

func (m *StorageManager) Create(ctx *resource.Context) (resource.Resource, *resterr.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resterr.NewAPIError(resterr.PermissionDenied, "only admin can create storage")
	}
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, "cluster doesn't exist")
	}
	storage := ctx.Resource.(*types.Storage)
	if err := m.createStorage(cluster, storage); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resterr.NewAPIError(resterr.DuplicateResource, fmt.Sprintf("duplicate storage name %s", storage.Name))
		} else {
			return nil, resterr.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create storage failed, %s", err.Error()))
		}
	}
	storage.SetID(storage.Name)
	return storage, nil
}

func (m *StorageManager) createStorage(cluster *zke.Cluster, storage *types.Storage) error {
	exist, err := m.checkStorageExist(cluster.GetKubeClient(), storage.Name)
	if err != nil {
		return err
	}
	if exist {
		return errors.New(fmt.Sprintf("the name %s of storage has already exists", storage.Name))
	}
	handle, err := m.getHandleFromType(storage.Type)
	if err != nil {
		return err
	}
	return handle.Create(cluster, storage)
}

func (m *StorageManager) checkStorageExist(cli client.Client, name string) (bool, error) {
	storages, err := m.getStorages(cli)
	if err != nil {
		return false, err
	}
	for _, storage := range storages {
		if storage.Name == name {
			return true, nil
		}
	}
	return false, nil
}

func (m *StorageManager) Update(ctx *resource.Context) (resource.Resource, *resterr.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resterr.NewAPIError(resterr.PermissionDenied, "only admin can update storage")
	}
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, "cluster doesn't exist")
	}

	storage := ctx.Resource.(*types.Storage)
	if err := m.updateStorage(cluster, storage); err != nil {
		return nil, resterr.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("update storage failed, %s", err.Error()))
	}
	return storage, nil
}

func (m *StorageManager) updateStorage(cluster *zke.Cluster, storage *types.Storage) error {
	handle, err := m.getHandleFromType(storage.Type)
	if err != nil {
		return err
	}
	return handle.Update(cluster, storage)
}

func sToi(size string) int64 {
	num, _ := strconv.ParseInt(size, 10, 64)
	return num
}

func byteToGb(num int64) string {
	f := float64(num) / (1024 * 1024 * 1024)
	return fmt.Sprintf("%.2f", math.Trunc(f*1e2)*1e-2)
}

func strToBool(str string) bool {
	if str == "true" {
		return true
	} else {
		return false
	}
}

func genStoragePVFromClusterAgent(cluster *zke.Cluster, name string) ([]types.PV, error) {
	var info types.PVInfo
	if err := clusteragent.GetAgent().GetResource(cluster.Name, "/storages/"+name, &info); err != nil {
		log.Warnf("get storages from clusteragent failed:%s", err.Error())
		return nil, err
	}
	return info.PVs, nil
}
func genStorageNodeFromInstances(instances []storagev1.Instance) []types.StorageNode {
	var nodes types.StorageNodes
	ns := make(map[string]map[string]int64)
	nodestat := make(map[string]bool)
	stat := true
	for _, i := range instances {
		if !i.Stat {
			stat = false
		}
		nodestat[i.Host] = stat
		v, ok := ns[i.Host]
		if ok {
			v["Total"] += sToi(i.Info.Total)
			v["Used"] += sToi(i.Info.Used)
			v["Free"] += sToi(i.Info.Free)
		} else {
			info := make(map[string]int64)
			info["Total"] = sToi(i.Info.Total)
			info["Used"] = sToi(i.Info.Used)
			info["Free"] = sToi(i.Info.Free)
			ns[i.Host] = info
		}
	}
	for k, v := range ns {
		node := types.StorageNode{
			Name:     k,
			Size:     byteToGb(v["Total"]),
			UsedSize: byteToGb(v["Used"]),
			FreeSize: byteToGb(v["Free"]),
			Stat:     nodestat[k],
		}
		nodes = append(nodes, node)
	}
	sort.Sort(nodes)
	return nodes
}
