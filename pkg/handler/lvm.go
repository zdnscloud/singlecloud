package handler

import (
	"context"
	"errors"
	"fmt"

	k8sstorage "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/cement/set"
	"github.com/zdnscloud/cement/slice"
	"github.com/zdnscloud/gok8s/client"
	storagev1 "github.com/zdnscloud/immense/pkg/apis/zcloud/v1"
	"github.com/zdnscloud/immense/pkg/common"
	"github.com/zdnscloud/singlecloud/pkg/clusteragent"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/pkg/zke"
)

const (
	LvmDriverSuffix = "lvm.storage.zcloud.cn"
)

type LvmManager struct {
}

func (s *LvmManager) GetType() types.StorageType {
	return types.LvmType
}

func getStorageCluster(cli client.Client, name string) (*storagev1.Cluster, error) {
	storageCluster := storagev1.Cluster{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{"", name}, &storageCluster)
	return &storageCluster, err
}

func getStorageClusters(cli client.Client) (*storagev1.ClusterList, error) {
	storageclusters := storagev1.ClusterList{}
	err := cli.List(context.TODO(), nil, &storageclusters)
	return &storageclusters, err
}

func checkStorageCluster(cli client.Client, name string, typ types.StorageType) (bool, error) {
	storageClusters, err := getStorageClusters(cli)
	if err != nil {
		return false, err
	}
	for _, storageCluster := range storageClusters.Items {
		if storageCluster.Spec.StorageType != string(typ) {
			continue
		}
		if storageCluster.Name == name {
			return true, nil
		}
	}
	return false, nil
}

func (s *LvmManager) HaveStorage(cli client.Client, name string) (bool, error) {
	return checkStorageCluster(cli, name, s.GetType())
}

func (s *LvmManager) GetStorages(cli client.Client) ([]*types.Storage, error) {
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
		storage.Parameter = types.Parameter{
			Lvm: &types.StorageClusterParameter{
				Hosts: storageCluster.Spec.Hosts,
			}}
		storages = append(storages, storage)
	}
	return storages, nil
}

func (s *LvmManager) GetStorage(cluster *zke.Cluster, name string) (*types.Storage, error) {
	storageCluster, err := getStorageCluster(cluster.GetKubeClient(), name)
	if err != nil {
		return nil, err
	}
	storage, err := storageClusterToSCStorageDetail(cluster, storageCluster)
	if err != nil {
		return nil, err
	}
	storage.Parameter = types.Parameter{
		Lvm: &types.StorageClusterParameter{
			Hosts: storageCluster.Spec.Hosts,
		}}
	return storage, nil
}

func storageClusterToSCStorageDetail(cluster *zke.Cluster, storageCluster *storagev1.Cluster) (*types.Storage, error) {
	storage := storageClusterToSCStorage(storageCluster)
	storage.Nodes = genStorageNodeFromInstances(storageCluster.Status.Capacity.Instances)
	pvs, err := genStoragePVFromClusterAgent(cluster, storageCluster.Name)
	if err != nil {
		return nil, err
	}
	storage.PVs = pvs
	return storage, nil
}

func storageClusterToSCStorage(storageCluster *storagev1.Cluster) *types.Storage {
	storage := &types.Storage{
		Name:     storageCluster.Name,
		Type:     types.StorageType(storageCluster.Spec.StorageType),
		Phase:    string(storageCluster.Status.Phase),
		Size:     byteToGb(sToi(storageCluster.Status.Capacity.Total.Total)),
		UsedSize: byteToGb(sToi(storageCluster.Status.Capacity.Total.Used)),
		FreeSize: byteToGb(sToi(storageCluster.Status.Capacity.Total.Free)),
	}
	storage.SetID(storageCluster.Name)
	storage.SetCreationTimestamp(storageCluster.CreationTimestamp.Time)
	if storageCluster.GetDeletionTimestamp() != nil {
		storage.SetDeletionTimestamp(storageCluster.DeletionTimestamp.Time)
	}
	return storage
}

func (s *LvmManager) Create(cluster *zke.Cluster, storage *types.Storage) error {
	if storage.Lvm != nil {
		return createStorageCluster(cluster, storage.Name, string(storage.Type), storage.Lvm.Hosts)
	}
	return errors.New(fmt.Sprintf(StorageParameterNullErr, storage.Name))
}

func createStorageCluster(cluster *zke.Cluster, name, typ string, hosts []string) error {
	exist, err := checkStorageClusterTypeExist(cluster.GetKubeClient(), typ)
	if err != nil {
		return err
	}
	if exist {
		return errors.New(fmt.Sprintf("the type %s stroage has already exists", typ))
	}
	if err := isHostsValidate(cluster, hosts); err != nil {
		return err
	}

	k8sStorageCluster := &storagev1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: storagev1.ClusterSpec{
			StorageType: typ,
			Hosts:       hosts,
		},
	}
	return cluster.GetKubeClient().Create(context.TODO(), k8sStorageCluster)
}

func checkStorageClusterTypeExist(cli client.Client, typ string) (bool, error) {
	storageclusters := storagev1.ClusterList{}
	err := cli.List(context.TODO(), nil, &storageclusters)
	if err != nil {
		return false, err
	}
	for _, storage := range storageclusters.Items {
		if storage.Spec.StorageType == typ {
			return true, nil
		}
	}
	return false, nil
}

func isHostsValidate(cluster *zke.Cluster, hosts []string) error {
	resp, err := getBlockDevices(cluster.Name, cluster.GetKubeClient(), clusteragent.GetAgent())
	if err != nil {
		return err
	}
	var invalidHosts []string
	for _, host := range hosts {
		if !checkUsed(resp, host) {
			continue
		}
		invalidHosts = append(invalidHosts, host)
	}
	if len(invalidHosts) > 0 {
		return errors.New(fmt.Sprintf("hosts %s can not be used for storage", invalidHosts))
	}
	return nil
}

func checkUsed(blockinfo []*types.BlockDevice, node string) bool {
	for _, info := range blockinfo {
		if info.NodeName == node && info.UsedBy == "" {
			return false
		}
	}
	return true
}

func (s *LvmManager) Delete(cli client.Client, name string) error {
	return deleteStorageCluster(cli, name)
}

func deleteStorageCluster(cli client.Client, name string) error {
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

func (s *LvmManager) Update(cluster *zke.Cluster, storage *types.Storage) error {
	return updateStorageCluster(cluster, storage.Name, storage.Type, storage.Lvm.Hosts)
}

func updateStorageCluster(cluster *zke.Cluster, name string, typ types.StorageType, hosts []string) error {
	k8sStorageCluster, err := getStorageCluster(cluster.GetKubeClient(), name)
	if err != nil {
		return err
	}
	if k8sStorageCluster.Status.Phase == storagev1.Creating ||
		k8sStorageCluster.Status.Phase == storagev1.Updating ||
		k8sStorageCluster.Status.Phase == storagev1.Deleting ||
		k8sStorageCluster.GetDeletionTimestamp() != nil {
		return errors.New(fmt.Sprintf("%s in Creating, Updating or Deleting, not allowed update", name))
	}

	s1 := set.StringSetFromSlice(k8sStorageCluster.Spec.Hosts)
	s2 := set.StringSetFromSlice(hosts)
	addHosts := s2.Difference(s1).ToSlice()
	delHosts := s1.Difference(s2).ToSlice()
	if typ == types.LvmType {
		if err := isDelHostsUsed(cluster.GetKubeClient(), name, typ, delHosts); err != nil {
			return err
		}
	}
	if err := isHostsValidate(cluster, addHosts); err != nil {
		return err
	}

	k8sStorageCluster.Spec.Hosts = hosts
	return cluster.GetKubeClient().Update(context.TODO(), k8sStorageCluster)
}

func isDelHostsUsed(cli client.Client, name string, typ types.StorageType, hosts []string) error {
	var suffix string
	if typ == types.LvmType {
		suffix = LvmDriverSuffix
	} else if typ == types.CephfsType {
		suffix = CephFsDriverSuffix
	} else {
		return errors.New(fmt.Sprintf("unknow storage type %s", typ))
	}
	usedHosts, err := getAttachedHosts(cli, fmt.Sprintf("%s.%s", name, suffix), hosts)
	if err != nil {
		return err
	}
	if len(usedHosts) > 0 {
		return errors.New(fmt.Sprintf("the storagehosts %s is used by some pods, you should stop those pods first", usedHosts))
	}
	return nil
}

func getAttachedHosts(cli client.Client, driverName string, nodes []string) ([]string, error) {
	var hosts []string
	volumeAttachments := k8sstorage.VolumeAttachmentList{}
	err := cli.List(context.TODO(), nil, &volumeAttachments)
	if err != nil {
		return hosts, err
	}
	for _, volumeAttachment := range volumeAttachments.Items {
		if driverName != volumeAttachment.Spec.Attacher {
			continue
		}
		if slice.SliceIndex(nodes, volumeAttachment.Spec.NodeName) >= 0 {
			if slice.SliceIndex(hosts, volumeAttachment.Spec.NodeName) >= 0 {
				continue
			}
			hosts = append(hosts, volumeAttachment.Spec.NodeName)
		}
	}
	return hosts, nil
}
