package handler

import (
	"context"

	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	restresource "github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type StorageClassManager struct {
	clusters *ClusterManager
}

func newStorageClassManager(clusters *ClusterManager) *StorageClassManager {
	return &StorageClassManager{clusters: clusters}
}

func (m *StorageClassManager) List(ctx *restresource.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	k8sStorageClasses, err := getStorageClasses(cluster.GetKubeClient())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("list storageclass info failed:%s", err.Error())
		}
		return nil
	}

	var storageClasses []*types.StorageClass
	for _, item := range k8sStorageClasses.Items {
		storageClasses = append(storageClasses, k8sStorageClassToScStorageClass(&item))
	}
	return storageClasses
}

func getStorageClasses(cli client.Client) (*storagev1.StorageClassList, error) {
	storageClassses := storagev1.StorageClassList{}
	err := cli.List(context.TODO(), nil, &storageClassses)
	return &storageClassses, err
}

func k8sStorageClassToScStorageClass(k8sStorageClass *storagev1.StorageClass) *types.StorageClass {
	storageClass := &types.StorageClass{
		Name: k8sStorageClass.Name,
	}
	storageClass.SetID(k8sStorageClass.Name)
	storageClass.SetCreationTimestamp(k8sStorageClass.CreationTimestamp.Time)
	if k8sStorageClass.GetDeletionTimestamp() != nil {
		storageClass.SetDeletionTimestamp(k8sStorageClass.DeletionTimestamp.Time)
	}
	return storageClass
}

func isStorageClassExist(cli client.Client, name string) bool {
	scs, _ := getStorageClasses(cli)
	for _, sc := range scs.Items {
		if sc.Name == name {
			return true
		}
	}
	return false
}
