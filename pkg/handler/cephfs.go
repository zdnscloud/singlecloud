package handler

import (
	"errors"
	"fmt"

	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/pkg/zke"
)

const (
	CephFsDriverSuffix = "cephfs.storage.zcloud.cn"
)

type CephFsManager struct {
}

func (s *CephFsManager) GetType() types.StorageType {
	return types.CephfsType
}

func (s *CephFsManager) GetStorages(cli client.Client) ([]*types.Storage, error) {
	storageClusters, err := getStorageClusters(cli)
	if err != nil {
		return nil, err
	}
	var storages []*types.Storage
	for _, storageCluster := range storageClusters.Items {
		if storageCluster.Spec.StorageType != string(s.GetType()) {
			continue
		}
		storage := storageClusterToSCStorage(&storageCluster)
		storages = append(storages, storage)
	}
	return storages, nil
}

func (s *CephFsManager) GetStorage(cluster *zke.Cluster, name string) (*types.Storage, error) {
	storageCluster, err := getStorageCluster(cluster.GetKubeClient(), name)
	if err != nil {
		return nil, err
	}
	storage, err := storageClusterToSCStorageDetail(cluster, storageCluster)
	if err != nil {
		return nil, err
	}
	return storage, nil
}

func (s *CephFsManager) Create(cluster *zke.Cluster, storage *types.Storage) error {
	if storage.CephFs != nil {
		return createStorageCluster(cluster, storage.Name, string(storage.Type), storage.CephFs.Hosts)
	}
	return errors.New(fmt.Sprintf(StorageParameterNullErr, storage.Name))
}

func (s *CephFsManager) Delete(cli client.Client, name string) error {
	return deleteStorageCluster(cli, name)
}

func (s *CephFsManager) Update(cluster *zke.Cluster, storage *types.Storage) error {
	return updateStorageCluster(cluster, storage.Name, storage.Type, storage.CephFs.Hosts)
}
