package handler

import (
	"context"
	"fmt"

	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/gok8s/client"
	resterr "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type StorageClassManager struct {
	clusters *ClusterManager
}

func newStorageClassManager(clusters *ClusterManager) *StorageClassManager {
	return &StorageClassManager{clusters: clusters}
}

func (m *StorageClassManager) List(ctx *restresource.Context) (interface{}, *resterr.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, "cluster doesn't exist")
	}

	k8sStorageClasses, err := getStorageClasses(cluster.GetKubeClient())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterr.NewAPIError(resterr.NotFound, "no found storageclasses")
		}
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("list storageclasses failed %s", err.Error()))
	}

	var storageClasses []*types.StorageClass
	for _, item := range k8sStorageClasses.Items {
		storageClasses = append(storageClasses, k8sStorageClassToScStorageClass(&item))
	}
	return storageClasses, nil
}

func getStorageClasses(cli client.Client) (*storagev1.StorageClassList, error) {
	storageClassses := storagev1.StorageClassList{}
	err := cli.List(context.TODO(), nil, &storageClassses)
	return &storageClassses, err
}

func getStorageClass(cli client.Client, name string) (*storagev1.StorageClass, error) {
	storageClass := storagev1.StorageClass{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{"", name}, &storageClass)
	return &storageClass, err
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

func isStorageClassOrDefaultExist(cli client.Client, name string) bool {
	scs, _ := getStorageClasses(cli)
	for _, sc := range scs.Items {
		if sc.Name == name {
			return true
		}
		if _default, ok := sc.Annotations[StorageClassDefaultKey]; ok && strToBool(_default) {
			return true
		}
	}
	return false
}
