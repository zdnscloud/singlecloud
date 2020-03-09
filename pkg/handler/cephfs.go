package handler

import (
	"context"
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/zdnscloud/cement/set"
	"github.com/zdnscloud/cement/slice"
	"github.com/zdnscloud/gok8s/client"
	storagev1 "github.com/zdnscloud/immense/pkg/apis/zcloud/v1"
	"github.com/zdnscloud/immense/pkg/common"
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

func (s *CephFsManager) GetStorage(cli client.Client, name string) (*types.Storage, error) {
	storageCluster, err := getStorageCluster(cli, name)
	if err != nil {
		return nil, err
	}
	storage := storageClusterToSCStorage(storageCluster)
	storage.Parameter = types.Parameter{
		CephFs: types.StorageClusterParameter{
			Hosts: storageCluster.Spec.Hosts,
		}}
	return storage, nil
}

func (s *CephFsManager) GetStorageDetail(cluster *zke.Cluster, name string) (*types.Storage, error) {
	storageCluster, err := getStorageCluster(cluster.GetKubeClient(), name)
	if err != nil {
		return nil, err
	}
	storage, err := storageClusterToSCStorageDetail(cluster, storageCluster)
	if err != nil {
		return nil, err
	}

	storage.Parameter = types.Parameter{
		CephFs: types.StorageClusterParameter{
			Hosts: storageCluster.Spec.Hosts,
		}}
	return storage, nil
}

func (s *CephFsManager) Create(cluster *zke.Cluster, storage *types.Storage) error {
	exist, err := checkStorageClusterTypeExist(cluster.GetKubeClient(), storage.Type)
	if err != nil {
		return err
	}
	if exist {
		return errors.New(fmt.Sprintf("the type %s stroage has already exists", string(storage.Type)))
	}
	if err := isHostsValidate(cluster, storage.CephFs.Hosts); err != nil {
		return err
	}

	k8sStorageCluster := &storagev1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: storage.Name,
		},
		Spec: storagev1.ClusterSpec{
			StorageType: string(storage.Type),
			Hosts:       storage.CephFs.Hosts,
		},
	}
	return cluster.GetKubeClient().Create(context.TODO(), k8sStorageCluster)
}

func (s *CephFsManager) Delete(cli client.Client, name string) error {
	storageCluster, err := getStorageCluster(cli, name)
	if err != nil {
		return err
	}
	if storageCluster.Status.Phase == storagev1.Creating ||
		storageCluster.Status.Phase == storagev1.Updating ||
		storageCluster.Status.Phase == storagev1.Deleting {
		return errors.New("storage in Creating, Updating or Deleting, not allowed delete")
	}

	finalizers := storageCluster.GetFinalizers()
	if (len(finalizers) == 0) || (len(finalizers) == 1 && slice.SliceIndex(finalizers, common.StoragePrestopHookFinalizer) == 0) {
		return cli.Delete(context.TODO(), storageCluster)
	} else {
		return errors.New(fmt.Sprintf("storage %s is used by some pods, you should stop those pods first", name))
	}
}

func (s *CephFsManager) Update(cluster *zke.Cluster, storage *types.Storage) error {
	k8sStorageCluster, err := getStorageCluster(cluster.GetKubeClient(), storage.Name)
	if err != nil {
		return err
	}
	if k8sStorageCluster.Status.Phase == storagev1.Creating ||
		k8sStorageCluster.Status.Phase == storagev1.Updating ||
		k8sStorageCluster.Status.Phase == storagev1.Deleting ||
		k8sStorageCluster.GetDeletionTimestamp() != nil {
		return errors.New("storage in Creating, Updating or Deleting, not allowed update")
	}

	s1 := set.StringSetFromSlice(k8sStorageCluster.Spec.Hosts)
	s2 := set.StringSetFromSlice(storage.CephFs.Hosts)
	addHosts := s2.Difference(s1).ToSlice()

	if err := isHostsValidate(cluster, addHosts); err != nil {
		return err
	}

	k8sStorageCluster.Spec.Hosts = storage.CephFs.Hosts
	return cluster.GetKubeClient().Update(context.TODO(), k8sStorageCluster)
}
