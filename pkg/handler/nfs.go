package handler

import (
	"context"
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/cement/slice"
	"github.com/zdnscloud/gok8s/client"
	storagev1 "github.com/zdnscloud/immense/pkg/apis/zcloud/v1"
	"github.com/zdnscloud/immense/pkg/common"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/pkg/zke"
)

const (
	NfsDriverSuffix = "nfs.storage.zcloud.cn"
)

type NfsManager struct {
}

func newNfsManager() *NfsManager {
	return &NfsManager{}
}

func (s *NfsManager) GetType() types.StorageType {
	return types.NfsType
}

func getNfs(cli client.Client, name string) (*storagev1.Nfs, error) {
	nfs := storagev1.Nfs{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{"", name}, &nfs)
	return &nfs, err
}

func getNfss(cli client.Client) (*storagev1.NfsList, error) {
	nfss := storagev1.NfsList{}
	err := cli.List(context.TODO(), nil, &nfss)
	return &nfss, err
}

func (s *NfsManager) GetStorages(cli client.Client) ([]*types.Storage, error) {
	nfss, err := getNfss(cli)
	if err != nil {
		return nil, err
	}
	var storages []*types.Storage
	for _, nfs := range nfss.Items {
		storages = append(storages, nfsToSCStorage(&nfs))
	}
	return storages, nil
}

func nfsToSCStorage(nfs *storagev1.Nfs) *types.Storage {
	storage := &types.Storage{
		Name: nfs.Name,
		Type: types.NfsType,
		Parameter: types.Parameter{
			Nfs: &types.NfsParameter{
				Server: nfs.Spec.Server,
				Path:   nfs.Spec.Path,
			}},
		Phase:    string(nfs.Status.Phase),
		Size:     byteToGb(sToi(nfs.Status.Capacity.Total.Total)),
		UsedSize: byteToGb(sToi(nfs.Status.Capacity.Total.Used)),
		FreeSize: byteToGb(sToi(nfs.Status.Capacity.Total.Free)),
	}
	storage.SetID(nfs.Name)
	storage.SetCreationTimestamp(nfs.CreationTimestamp.Time)
	if nfs.GetDeletionTimestamp() != nil {
		storage.SetDeletionTimestamp(nfs.DeletionTimestamp.Time)
	}
	return storage
}

func (s *NfsManager) GetStorage(cluster *zke.Cluster, name string) (*types.Storage, error) {
	nfs, err := getNfs(cluster.GetKubeClient(), name)
	if err != nil {
		return nil, err
	}
	return nfsToSCStorageDetail(cluster, nfs)
}

func nfsToSCStorageDetail(cluster *zke.Cluster, nfs *storagev1.Nfs) (*types.Storage, error) {
	storage := nfsToSCStorage(nfs)
	storage.Nodes = genStorageNodeFromInstances(nfs.Status.Capacity.Instances)
	pvs, err := genStoragePVFromClusterAgent(cluster, nfs.Name)
	if err != nil {
		return nil, err
	}
	storage.PVs = pvs
	return storage, nil
}

func (s *NfsManager) Create(cluster *zke.Cluster, storage *types.Storage) error {
	if storage.Nfs != nil {
		if !isIpv4(storage.Nfs.Server) {
			return errors.New("must be an ipv4 address")
		}
		k8sNfs := &storagev1.Nfs{
			ObjectMeta: metav1.ObjectMeta{
				Name: storage.Name,
			},
			Spec: storagev1.NfsSpec{
				Server: storage.Nfs.Server,
				Path:   storage.Nfs.Path,
			},
		}
		return cluster.GetKubeClient().Create(context.TODO(), k8sNfs)
	}
	return errors.New(fmt.Sprintf(StorageParameterNullErr, storage.Name))
}

func (s *NfsManager) Delete(cli client.Client, name string) error {
	nfs, err := getNfs(cli, name)
	if err != nil {
		return err
	}
	if nfs.Status.Phase == storagev1.Creating || nfs.Status.Phase == storagev1.Updating || nfs.Status.Phase == storagev1.Deleting {
		return errors.New("storage in Creating, Updating or Deleting, not allowed delete")
	}

	finalizers := nfs.GetFinalizers()
	if (len(finalizers) == 0) || (len(finalizers) == 1 && slice.SliceIndex(finalizers, common.StoragePrestopHookFinalizer) == 0) {
		return cli.Delete(context.TODO(), nfs)
	} else {
		return errors.New(fmt.Sprintf("storage %s is used by some pvcs, you should delete those pvc first", name))
	}
}

func (s *NfsManager) Update(cluster *zke.Cluster, storage *types.Storage) error {
	if !isIpv4(storage.Nfs.Server) {
		return errors.New("must be an ipv4 address")
	}
	k8sNfs, err := getNfs(cluster.GetKubeClient(), storage.Name)
	if err != nil {
		return err
	}
	if k8sNfs.Status.Phase == storagev1.Creating || k8sNfs.Status.Phase == storagev1.Updating || k8sNfs.Status.Phase == storagev1.Deleting {
		return errors.New("iscsi in Creating, Updating or Deleting, not allowed update")
	}
	if k8sNfs.GetDeletionTimestamp() != nil {
		return errors.New("nfs in Deleting, not allowed update")
	}
	finalizers := k8sNfs.GetFinalizers()
	if (len(finalizers) == 0) || (len(finalizers) == 1 && slice.SliceIndex(finalizers, common.StoragePrestopHookFinalizer) == 0) {
	} else {
		return errors.New(fmt.Sprintf("storage %s is used by some pvcs, you should delete those pvc first", storage.Name))
	}

	k8sNfs.Spec.Server = storage.Nfs.Server
	k8sNfs.Spec.Path = storage.Nfs.Path
	return cluster.GetKubeClient().Update(context.TODO(), k8sNfs)
}
