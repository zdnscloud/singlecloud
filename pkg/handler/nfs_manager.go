package handler

import (
	"context"
	"errors"
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/slice"
	"github.com/zdnscloud/gok8s/client"
	gorestError "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	storagev1 "github.com/zdnscloud/immense/pkg/apis/zcloud/v1"
	"github.com/zdnscloud/immense/pkg/common"
	"github.com/zdnscloud/singlecloud/pkg/clusteragent"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/pkg/zke"
)

type NfsManager struct {
	clusters *ClusterManager
}

func newNfsManager(clusters *ClusterManager) *NfsManager {
	return &NfsManager{
		clusters: clusters,
	}
}

func (m *NfsManager) List(ctx *resource.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	k8sNfss, err := getNfss(cluster.GetKubeClient())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("list nfs failed:%s", err.Error())
		}
		return nil
	}

	var nfss []*types.Nfs
	for _, item := range k8sNfss.Items {
		nfss = append(nfss, k8sNfsToSCNfs(cluster, &item))
	}
	return nfss
}

func (m NfsManager) Get(ctx *resource.Context) resource.Resource {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	nfs := ctx.Resource.(*types.Nfs)
	k8sNfs, err := getNfs(cluster.GetKubeClient(), nfs.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("get nfs info failed:%s", err.Error())
		}
		return nil
	}

	return k8sNfsToSCNfsDetail(cluster, clusteragent.GetAgent(), k8sNfs)
}

func (m NfsManager) Delete(ctx *resource.Context) *gorestError.APIError {
	if isAdmin(getCurrentUser(ctx)) == false {
		return gorestError.NewAPIError(gorestError.PermissionDenied, "only admin can delete nfs")
	}
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return gorestError.NewAPIError(gorestError.NotFound, "nfs doesn't exist")
	}

	nfs := ctx.Resource.(*types.Nfs)
	if err := deleteNfs(cluster.GetKubeClient(), nfs.GetID()); err != nil {
		if apierrors.IsNotFound(err) {
			return gorestError.NewAPIError(gorestError.NotFound, fmt.Sprintf("nfs %s doesn't exist", nfs.GetID()))
		} else if strings.Contains(err.Error(), "is used by") || strings.Contains(err.Error(), "Creating") {
			return gorestError.NewAPIError(types.InvalidClusterConfig, fmt.Sprintf("delete nfs failed, %s", err.Error()))
		} else {
			return gorestError.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete nfs failed, %s", err.Error()))
		}
	}
	return nil
}

func (m NfsManager) Create(ctx *resource.Context) (resource.Resource, *gorestError.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, gorestError.NewAPIError(gorestError.PermissionDenied, "only admin can create nfs")
	}
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, gorestError.NewAPIError(gorestError.NotFound, "cluster doesn't exist")
	}

	nfs := ctx.Resource.(*types.Nfs)
	if err := createNfs(cluster, clusteragent.GetAgent(), nfs); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, gorestError.NewAPIError(gorestError.DuplicateResource, fmt.Sprintf("duplicate nfs name %s", nfs.Name))
		} else if strings.Contains(err.Error(), "nfs has already exists") || strings.Contains(err.Error(), "can not be used for") {
			return nil, gorestError.NewAPIError(types.InvalidClusterConfig, fmt.Sprintf("create nfs failed, %s", err.Error()))
		} else {
			return nil, gorestError.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create nfs failed, %s", err.Error()))
		}
	}
	nfs.SetID(nfs.Name)
	return nfs, nil
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

func deleteNfs(cli client.Client, name string) error {
	k8sNfs, err := getNfs(cli, name)
	if err != nil {
		return err
	}
	if k8sNfs.Status.Phase == storagev1.Creating || k8sNfs.Status.Phase == storagev1.Updating || k8sNfs.Status.Phase == storagev1.Deleting {
		return errors.New("nfs in Creating, Updating or Deleting, not allowed delete")
	}

	if err := checkNfsFinalizers(cli, name); err != nil {
		return err
	}

	nfs := &storagev1.Nfs{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	return cli.Delete(context.TODO(), nfs)
}

func createNfs(cluster *zke.Cluster, agent *clusteragent.AgentManager, nfs *types.Nfs) error {
	if err := checkNfsExist(cluster.GetKubeClient(), nfs.Name); err != nil {
		return err
	}

	k8sNfs := scNfsToK8sNfs(nfs)
	return cluster.GetKubeClient().Create(context.TODO(), k8sNfs)
}

func scNfsToK8sNfs(nfs *types.Nfs) *storagev1.Nfs {
	return &storagev1.Nfs{
		ObjectMeta: metav1.ObjectMeta{
			Name: nfs.Name,
		},
		Spec: storagev1.NfsSpec{
			Server: nfs.Server,
			Path:   nfs.Path,
		},
	}
}

func k8sNfsToSCNfs(cluster *zke.Cluster, k8sNfs *storagev1.Nfs) *types.Nfs {
	tSize := byteToGb(sToi(k8sNfs.Status.Capacity.Total.Total))
	uSize := byteToGb(sToi(k8sNfs.Status.Capacity.Total.Used))
	fSize := byteToGb(sToi(k8sNfs.Status.Capacity.Total.Free))
	nfs := &types.Nfs{
		Name:     k8sNfs.Name,
		Server:   k8sNfs.Spec.Server,
		Path:     k8sNfs.Spec.Path,
		Phase:    string(k8sNfs.Status.Phase),
		Size:     tSize,
		UsedSize: uSize,
		FreeSize: fSize,
	}
	nfs.SetID(k8sNfs.Name)
	nfs.SetCreationTimestamp(k8sNfs.CreationTimestamp.Time)
	if k8sNfs.GetDeletionTimestamp() != nil {
		nfs.SetDeletionTimestamp(k8sNfs.DeletionTimestamp.Time)
	}
	return nfs
}

func k8sNfsToSCNfsDetail(cluster *zke.Cluster, agent *clusteragent.AgentManager, k8sNfs *storagev1.Nfs) *types.Nfs {
	nfs := k8sNfsToSCNfs(cluster, k8sNfs)
	var info types.Nfs
	if err := agent.GetResource(cluster.Name, "/storages/"+k8sNfs.Name, &info); err != nil {
		log.Warnf("get storages from clusteragent failed:%s", err.Error())
	} else {
		nfs.PVs = info.PVs
	}
	return nfs
}

func checkNfsExist(cli client.Client, name string) error {
	nfss := storagev1.NfsList{}
	err := cli.List(context.TODO(), nil, &nfss)
	if err != nil {
		return err
	}
	for _, nfs := range nfss.Items {
		if nfs.Name == name {
			return errors.New(fmt.Sprintf("The name of %s nfs has already exists", name))
		}
	}
	return nil
}

func checkNfsFinalizers(cli client.Client, name string) error {
	var obj runtime.Object
	obj, err := getNfs(cli, name)
	if err != nil {
		return err
	}
	metaObj := obj.(metav1.Object)
	finalizers := metaObj.GetFinalizers()
	if (len(finalizers) == 0) || (len(finalizers) == 1 && slice.SliceIndex(finalizers, common.StoragePrestopHookFinalizer) == 0) {
		return nil
	} else {
		return errors.New(fmt.Sprintf("The nfs %s is used by some pods, you should stop those pods first", name))
	}
}
