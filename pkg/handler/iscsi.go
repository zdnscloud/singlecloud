package handler

import (
	"context"
	"errors"
	"fmt"
	"net"
	"reflect"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/cement/set"
	"github.com/zdnscloud/cement/slice"
	"github.com/zdnscloud/gok8s/client"
	storagev1 "github.com/zdnscloud/immense/pkg/apis/zcloud/v1"
	"github.com/zdnscloud/immense/pkg/common"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/pkg/zke"
)

const (
	IscsiInstanceSecretSuffix = "iscsi-secret"
	IscsiDriverSuffix         = "iscsi.storage.zcloud.cn"
)

type IscsiManager struct {
}

func (s *IscsiManager) GetType() types.StorageType {
	return types.IscsiType
}

func getIscsi(cli client.Client, name string) (*storagev1.Iscsi, error) {
	iscsi := storagev1.Iscsi{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{"", name}, &iscsi)
	return &iscsi, err
}

func getIscsis(cli client.Client) (*storagev1.IscsiList, error) {
	iscsis := storagev1.IscsiList{}
	err := cli.List(context.TODO(), nil, &iscsis)
	return &iscsis, err
}

func (s *IscsiManager) HaveStorage(cli client.Client, name string) (bool, error) {
	iscsis, err := getIscsis(cli)
	if err != nil {
		return false, err
	}
	for _, iscsi := range iscsis.Items {
		if iscsi.Name == name {
			return true, nil
		}
	}
	return false, nil
}

func (s *IscsiManager) GetStorages(cli client.Client) ([]*types.Storage, error) {
	iscsis, err := getIscsis(cli)
	if err != nil {
		return nil, err
	}
	var storages []*types.Storage
	for _, iscsi := range iscsis.Items {
		storages = append(storages, iscsiToSCStorage(&iscsi))
	}
	return storages, nil
}

func (s *IscsiManager) GetStorage(cluster *zke.Cluster, name string) (*types.Storage, error) {
	iscsi, err := getIscsi(cluster.GetKubeClient(), name)
	if err != nil {
		return nil, err
	}
	return iscsiToSCStorageDetail(cluster, iscsi)
}

func (s *IscsiManager) Delete(cli client.Client, name string) error {
	iscsi, err := getIscsi(cli, name)
	if err != nil {
		return err
	}
	if iscsi.Status.Phase == storagev1.Creating || iscsi.Status.Phase == storagev1.Updating || iscsi.Status.Phase == storagev1.Deleting {
		return errors.New("storage in Creating, Updating or Deleting, not allowed delete")
	}

	finalizers := iscsi.GetFinalizers()
	if (len(finalizers) == 0) || (len(finalizers) == 1 && slice.SliceIndex(finalizers, common.StoragePrestopHookFinalizer) == 0) {
		if iscsi.Spec.Chap {
			if err := deleteSecret(cli, ZCloudNamespace, fmt.Sprintf("%s-%s", name, IscsiInstanceSecretSuffix)); err != nil {
				return err
			}
		}
		return cli.Delete(context.TODO(), iscsi)
	} else {
		return errors.New(fmt.Sprintf("storage %s is used by some pvcs, you should delete those pvc first", name))
	}
}

func iscsiToSCStorageDetail(cluster *zke.Cluster, iscsi *storagev1.Iscsi) (*types.Storage, error) {
	storage := iscsiToSCStorage(iscsi)
	storage.Nodes = genStorageNodeFromInstances(iscsi.Status.Capacity.Instances)
	pvs, err := genStoragePVFromClusterAgent(cluster, iscsi.Name)
	if err != nil {
		return nil, err
	}
	storage.PVs = pvs
	if iscsi.Spec.Chap {
		secret, err := getIscsiSecret(cluster.GetKubeClient(), ZCloudNamespace, fmt.Sprintf("%s-%s", iscsi.Name, IscsiInstanceSecretSuffix))
		if err != nil {
			return nil, err
		}
		for _, d := range secret.Data {
			if d.Key == "username" {
				storage.Iscsi.Username = d.Value
			}
			if d.Key == "password" {
				storage.Iscsi.Password = d.Value
			}
		}
	}

	return storage, nil
}

func getIscsiSecret(cli client.Client, namespace, name string) (*types.Secret, error) {
	k8sSecret, err := getSecret(cli, namespace, name)
	if err != nil {
		return nil, err
	}
	return k8sSecretToSCSecret(k8sSecret), nil
}

func (s *IscsiManager) Create(cluster *zke.Cluster, storage *types.Storage) error {
	if storage.Iscsi != nil {
		if err := iscsiValidation(storage); err != nil {
			return err
		}
		ok, err := validateInitiators(cluster.GetKubeClient(), storage.Iscsi.Initiators)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("controlplane or etcd node can not be initiators")
		}
		if storage.Iscsi.Chap {
			secret := genIscsiSecret(storage)
			if err := createSecret(cluster.GetKubeClient(), ZCloudNamespace, secret); err != nil {
				return err
			}
		}

		k8sIscsi := &storagev1.Iscsi{
			ObjectMeta: metav1.ObjectMeta{
				Name: storage.Name,
			},
			Spec: storagev1.IscsiSpec{
				Targets:    storage.Iscsi.Targets,
				Port:       storage.Iscsi.Port,
				Iqn:        storage.Iscsi.Iqn,
				Chap:       storage.Iscsi.Chap,
				Initiators: storage.Iscsi.Initiators,
			},
		}
		return cluster.GetKubeClient().Create(context.TODO(), k8sIscsi)
	}
	return errors.New(fmt.Sprintf(StorageParameterNullErr, storage.Name))
}

func validateInitiators(cli client.Client, initiators []string) (bool, error) {
	for _, name := range initiators {
		node, err := getK8SNode(cli, name)
		if err != nil {
			return false, err
		}
		if _, ok := node.Labels[zkeRoleLabelPrefix+string(types.RoleControlPlane)]; ok {
			return false, nil
		}
		if _, ok := node.Labels[zkeRoleLabelPrefix+string(types.RoleEtcd)]; ok {
			return false, nil
		}
	}
	return true, nil
}

func genIscsiSecret(storage *types.Storage) *types.Secret {
	return &types.Secret{
		Name: fmt.Sprintf("%s-%s", storage.Name, IscsiInstanceSecretSuffix),
		Data: []types.SecretData{
			types.SecretData{
				Key:   "username",
				Value: storage.Iscsi.Username,
			},
			types.SecretData{
				Key:   "password",
				Value: storage.Iscsi.Password,
			},
		},
	}
}

func iscsiToSCStorage(iscsi *storagev1.Iscsi) *types.Storage {
	storage := &types.Storage{
		Name: iscsi.Name,
		Type: types.IscsiType,
		Parameter: types.Parameter{
			Iscsi: &types.IscsiParameter{
				Targets:    iscsi.Spec.Targets,
				Port:       iscsi.Spec.Port,
				Iqn:        iscsi.Spec.Iqn,
				Chap:       iscsi.Spec.Chap,
				Initiators: iscsi.Spec.Initiators,
			}},
		Phase:    string(iscsi.Status.Phase),
		Size:     byteToGb(sToi(iscsi.Status.Capacity.Total.Total)),
		UsedSize: byteToGb(sToi(iscsi.Status.Capacity.Total.Used)),
		FreeSize: byteToGb(sToi(iscsi.Status.Capacity.Total.Free)),
	}
	storage.SetID(iscsi.Name)
	storage.SetCreationTimestamp(iscsi.CreationTimestamp.Time)
	if iscsi.GetDeletionTimestamp() != nil {
		storage.SetDeletionTimestamp(iscsi.DeletionTimestamp.Time)
	}
	return storage
}

func updateIscsiSecret(cli client.Client, storage *types.Storage) error {
	_k8sSecret, err := getSecret(cli, ZCloudNamespace, fmt.Sprintf("%s-%s", storage.Name, IscsiInstanceSecretSuffix))
	if err != nil {
		return err
	}
	k8sSecret, err := scSecretToK8sSecret(genIscsiSecret(storage), ZCloudNamespace)
	if err != nil {
		return err
	}
	_k8sSecret.Data = k8sSecret.Data
	return cli.Update(context.TODO(), _k8sSecret)
}

func (s *IscsiManager) Update(cluster *zke.Cluster, storage *types.Storage) error {
	if err := iscsiValidation(storage); err != nil {
		return err
	}
	k8sIscsi, err := getIscsi(cluster.GetKubeClient(), storage.Name)
	if err != nil {
		return err
	}
	if k8sIscsi.Status.Phase == storagev1.Creating || k8sIscsi.Status.Phase == storagev1.Updating || k8sIscsi.Status.Phase == storagev1.Deleting {
		return errors.New("iscsi in Creating, Updating or Deleting, not allowed update")
	}
	if k8sIscsi.GetDeletionTimestamp() != nil {
		return errors.New("iscsi in Deleting, not allowed update")
	}
	if !reflect.DeepEqual(k8sIscsi.Spec.Initiators, storage.Iscsi.Initiators) {
		s1 := set.StringSetFromSlice(k8sIscsi.Spec.Initiators)
		s2 := set.StringSetFromSlice(storage.Iscsi.Initiators)
		addHosts := s2.Difference(s1).ToSlice()
		delHosts := s1.Difference(s2).ToSlice()
		if err := isDelHostsUsed(cluster.GetKubeClient(), k8sIscsi.Name, types.IscsiType, delHosts); err != nil {
			return err
		}

		ok, err := validateInitiators(cluster.GetKubeClient(), addHosts)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("controlplane or etcd node can not be initiators")
		}
	}
	var chapChange bool
	if storage.Iscsi.Chap && k8sIscsi.Spec.Chap == storage.Iscsi.Chap {
		secret, err := getIscsiSecret(cluster.GetKubeClient(), ZCloudNamespace, fmt.Sprintf("%s-%s", k8sIscsi.Name, IscsiInstanceSecretSuffix))
		if err != nil {
			return err
		}
		for _, d := range secret.Data {
			if d.Key == "username" && d.Value != storage.Iscsi.Username {
				chapChange = true
			}
			if d.Key == "password" && d.Value != storage.Iscsi.Password {
				chapChange = true
			}
		}
	}

	if !reflect.DeepEqual(k8sIscsi.Spec.Targets, storage.Iscsi.Targets) ||
		k8sIscsi.Spec.Port != storage.Iscsi.Port ||
		k8sIscsi.Spec.Iqn != storage.Iscsi.Iqn ||
		k8sIscsi.Spec.Chap != storage.Iscsi.Chap ||
		chapChange {
		finalizers := k8sIscsi.GetFinalizers()
		if (len(finalizers) == 0) || (len(finalizers) == 1 && slice.SliceIndex(finalizers, common.StoragePrestopHookFinalizer) == 0) {
			if k8sIscsi.Spec.Chap && !storage.Iscsi.Chap {
				if err := deleteSecret(cluster.GetKubeClient(), ZCloudNamespace, fmt.Sprintf("%s-%s", k8sIscsi.Name, IscsiInstanceSecretSuffix)); err != nil {
					return err
				}
			}
			if !k8sIscsi.Spec.Chap && storage.Iscsi.Chap {
				secret := genIscsiSecret(storage)
				if err := createSecret(cluster.GetKubeClient(), ZCloudNamespace, secret); err != nil {
					return err
				}
			}
			if chapChange {
				if err := updateIscsiSecret(cluster.GetKubeClient(), storage); err != nil {
					return err
				}
			}
		} else {
			return errors.New(fmt.Sprintf("storage %s is used by some pvcs, you should delete those pvc first", storage.Name))
		}
	}

	k8sIscsi.Spec.Targets = storage.Iscsi.Targets
	k8sIscsi.Spec.Port = storage.Iscsi.Port
	k8sIscsi.Spec.Iqn = storage.Iscsi.Iqn
	k8sIscsi.Spec.Chap = storage.Iscsi.Chap
	k8sIscsi.Spec.Initiators = storage.Iscsi.Initiators
	return cluster.GetKubeClient().Update(context.TODO(), k8sIscsi)
}

func iscsiValidation(storage *types.Storage) error {
	if strings.Contains(storage.Iscsi.Iqn, " ") {
		return errors.New("iscsi iqn cannot contain Spaces")
	}
	tmp := make(map[string]string)
	for _, target := range storage.Iscsi.Targets {
		if !isIpv4(target) {
			return errors.New("iscsi target must be an ipv4 address")
		}
		if _, ok := tmp[target]; ok {
			return errors.New("iscsi target duplicate")
		} else {
			tmp[target] = ""
		}

	}
	if len(storage.Iscsi.Initiators) == 0 {
		return errors.New("iscsi initiators at lesat one")
	}
	if storage.Iscsi.Chap {
		if storage.Iscsi.Username == "" || storage.Iscsi.Password == "" {
			return errors.New("if chap is checked, fields username and password can not be empty")
		}
	}
	return nil
}

func isIpv4(input string) bool {
	ip := net.ParseIP(input)
	return ip != nil && ip.To4() != nil
}
