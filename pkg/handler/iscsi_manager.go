package handler

import (
	"context"
	"errors"
	"fmt"
	"sort"
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

const (
	IscsiInstanceSecretSuffix = "iscsi-secret"
)

type IscsiManager struct {
	clusters *ClusterManager
}

func newIscsiManager(clusters *ClusterManager) *IscsiManager {
	return &IscsiManager{
		clusters: clusters,
	}
}

func (m *IscsiManager) List(ctx *resource.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	k8sIscsis, err := getIscsis(cluster.GetKubeClient())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("list iscsi failed:%s", err.Error())
		}
		return nil
	}

	var iscsis []*types.Iscsi
	for _, item := range k8sIscsis.Items {
		iscsis = append(iscsis, k8sIscsiToSCIscsi(cluster, &item))
	}
	return iscsis
}

func (m IscsiManager) Get(ctx *resource.Context) resource.Resource {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	iscsi := ctx.Resource.(*types.Iscsi)
	k8sIscsi, err := getIscsi(cluster.GetKubeClient(), iscsi.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("get iscsi info failed:%s", err.Error())
		}
		return nil
	}

	return k8sIscsiToSCIscsiDetail(cluster, clusteragent.GetAgent(), k8sIscsi)
}

func (m IscsiManager) Delete(ctx *resource.Context) *gorestError.APIError {
	if isAdmin(getCurrentUser(ctx)) == false {
		return gorestError.NewAPIError(gorestError.PermissionDenied, "only admin can delete iscsi")
	}
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return gorestError.NewAPIError(gorestError.NotFound, "iscsi doesn't exist")
	}

	iscsi := ctx.Resource.(*types.Iscsi)
	if err := deleteIscsi(cluster.GetKubeClient(), iscsi.GetID()); err != nil {
		if apierrors.IsNotFound(err) {
			return gorestError.NewAPIError(gorestError.NotFound, fmt.Sprintf("iscsi %s doesn't exist", iscsi.GetID()))
		} else if strings.Contains(err.Error(), "is used by") || strings.Contains(err.Error(), "Creating") {
			return gorestError.NewAPIError(types.InvalidClusterConfig, fmt.Sprintf("delete iscsi failed, %s", err.Error()))
		} else {
			return gorestError.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete iscsi failed, %s", err.Error()))
		}
	}
	return nil
}

func (m IscsiManager) Create(ctx *resource.Context) (resource.Resource, *gorestError.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, gorestError.NewAPIError(gorestError.PermissionDenied, "only admin can create iscsi")
	}
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, gorestError.NewAPIError(gorestError.NotFound, "cluster doesn't exist")
	}

	iscsi := ctx.Resource.(*types.Iscsi)
	if err := createIscsi(cluster, clusteragent.GetAgent(), iscsi); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, gorestError.NewAPIError(gorestError.DuplicateResource, fmt.Sprintf("duplicate iscsi name %s", iscsi.Name))
		} else if strings.Contains(err.Error(), "iscsi has already exists") || strings.Contains(err.Error(), "can not be used for") {
			return nil, gorestError.NewAPIError(types.InvalidClusterConfig, fmt.Sprintf("create iscsi failed, %s", err.Error()))
		} else {
			return nil, gorestError.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create iscsi failed, %s", err.Error()))
		}
	}
	iscsi.SetID(iscsi.Name)
	return iscsi, nil
}

func (m IscsiManager) Update(ctx *resource.Context) (resource.Resource, *gorestError.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, gorestError.NewAPIError(gorestError.PermissionDenied, "only admin can update iscsi")
	}
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, gorestError.NewAPIError(gorestError.NotFound, "cluster doesn't exist")
	}

	iscsi := ctx.Resource.(*types.Iscsi)
	if err := updateIscsi(cluster, clusteragent.GetAgent(), iscsi); err != nil {
		if strings.Contains(err.Error(), "iscsi must keep") || strings.Contains(err.Error(), "is used by") || strings.Contains(err.Error(), "can not be used for") || strings.Contains(err.Error(), "Creating") {
			return nil, gorestError.NewAPIError(types.InvalidClusterConfig, fmt.Sprintf("update iscsi failed, %s", err.Error()))
		} else {
			return nil, gorestError.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("update iscsi failed, %s", err.Error()))
		}
	}
	return iscsi, nil
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

func deleteIscsi(cli client.Client, name string) error {
	k8sIscsi, err := getIscsi(cli, name)
	if err != nil {
		return err
	}
	if k8sIscsi.Status.Phase == storagev1.Creating || k8sIscsi.Status.Phase == storagev1.Updating || k8sIscsi.Status.Phase == storagev1.Deleting {
		return errors.New("iscsi in Creating, Updating or Deleting, not allowed delete")
	}

	if err := checkIscsiFinalizers(cli, name); err != nil {
		return err
	}
	if err := deleteSecret(cli, ZCloudNamespace, fmt.Sprintf("%s-%s", name, IscsiInstanceSecretSuffix)); err != nil {
		return err
	}

	iscsi := &storagev1.Iscsi{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	return cli.Delete(context.TODO(), iscsi)
}

func createIscsi(cluster *zke.Cluster, agent *clusteragent.AgentManager, iscsi *types.Iscsi) error {
	if err := checkIscsiExist(cluster.GetKubeClient(), iscsi.Name); err != nil {
		return err
	}

	if iscsi.Chap {
		if iscsi.Username == "" || iscsi.Password == "" {
			return errors.New("if chap is checked, fields username and password can not be empty")
		}
		if err := createIscsiSecret(cluster.GetKubeClient(), ZCloudNamespace, fmt.Sprintf("%s-%s", iscsi.Name, IscsiInstanceSecretSuffix), iscsi.Username, iscsi.Password); err != nil {
			return err
		}
	}

	k8sIscsi := scIscsiToK8sIscsi(iscsi)
	return cluster.GetKubeClient().Create(context.TODO(), k8sIscsi)
}

func createIscsiSecret(cli client.Client, namespace, name, username, password string) error {
	secret := &types.Secret{
		Name: name,
		Data: []types.SecretData{
			types.SecretData{
				Key:   "username",
				Value: username,
			},
			types.SecretData{
				Key:   "password",
				Value: password,
			},
		},
	}
	return createSecret(cli, namespace, secret)
}

func getIscsiSecret(cli client.Client, namespace, name string) (*types.Secret, error) {
	k8sSecret, err := getSecret(cli, namespace, name)
	if err != nil {
		return nil, err
	}
	return k8sSecretToSCSecret(k8sSecret), nil
}

func updateIscsi(cluster *zke.Cluster, agent *clusteragent.AgentManager, iscsi *types.Iscsi) error {
	k8sIscsi, err := getIscsi(cluster.GetKubeClient(), iscsi.GetID())
	if err != nil {
		return err
	}
	if k8sIscsi.Status.Phase == storagev1.Creating || k8sIscsi.Status.Phase == storagev1.Updating || k8sIscsi.Status.Phase == storagev1.Deleting {
		return errors.New("iscsi in Creating, Updating or Deleting, not allowed update")
	}
	if k8sIscsi.GetDeletionTimestamp() != nil {
		return errors.New("iscsi in Deleting, not allowed update")
	}
	if k8sIscsi.Spec.Target != iscsi.Target || k8sIscsi.Spec.Port != iscsi.Port || k8sIscsi.Spec.Iqn != iscsi.Iqn || k8sIscsi.Spec.Chap != iscsi.Chap {
		return errors.New(fmt.Sprintf("iscsi %s only initiators can be update", iscsi.Name))
	}

	k8sIscsi.Spec.Initiators = iscsi.Initiators
	return cluster.GetKubeClient().Update(context.TODO(), k8sIscsi)
}

func scIscsiToK8sIscsi(iscsi *types.Iscsi) *storagev1.Iscsi {
	return &storagev1.Iscsi{
		ObjectMeta: metav1.ObjectMeta{
			Name: iscsi.Name,
		},
		Spec: storagev1.IscsiSpec{
			Target:     iscsi.Target,
			Port:       iscsi.Port,
			Iqn:        iscsi.Iqn,
			Chap:       iscsi.Chap,
			Initiators: iscsi.Initiators,
		},
	}
}

func k8sIscsiToSCIscsi(cluster *zke.Cluster, k8sIscsi *storagev1.Iscsi) *types.Iscsi {
	tSize := byteToGb(sToi(k8sIscsi.Status.Capacity.Total.Total))
	uSize := byteToGb(sToi(k8sIscsi.Status.Capacity.Total.Used))
	fSize := byteToGb(sToi(k8sIscsi.Status.Capacity.Total.Free))
	iscsi := &types.Iscsi{
		Name:       k8sIscsi.Name,
		Target:     k8sIscsi.Spec.Target,
		Port:       k8sIscsi.Spec.Port,
		Iqn:        k8sIscsi.Spec.Iqn,
		Chap:       k8sIscsi.Spec.Chap,
		Initiators: k8sIscsi.Spec.Initiators,
		Phase:      string(k8sIscsi.Status.Phase),
		Size:       tSize,
		UsedSize:   uSize,
		FreeSize:   fSize,
	}
	iscsi.SetID(k8sIscsi.Name)
	iscsi.SetCreationTimestamp(k8sIscsi.CreationTimestamp.Time)
	if k8sIscsi.GetDeletionTimestamp() != nil {
		iscsi.SetDeletionTimestamp(k8sIscsi.DeletionTimestamp.Time)
	}
	return iscsi
}

func k8sIscsiToSCIscsiDetail(cluster *zke.Cluster, agent *clusteragent.AgentManager, k8sIscsi *storagev1.Iscsi) *types.Iscsi {
	iscsi := k8sIscsiToSCIscsi(cluster, k8sIscsi)
	iscsi.Nodes = genStorageNodeFromInstances(k8sIscsi.Status.Capacity.Instances)
	var info types.Iscsi
	if err := agent.GetResource(cluster.Name, "/storages/"+k8sIscsi.Name, &info); err != nil {
		log.Warnf("get storages from clusteragent failed:%s", err.Error())
	} else {
		iscsi.PVs = info.PVs
	}
	if secret, err := getIscsiSecret(cluster.GetKubeClient(), ZCloudNamespace, fmt.Sprintf("%s-%s", k8sIscsi.Name, IscsiInstanceSecretSuffix)); err != nil {
		log.Warnf("get iscsi secret %s failed:%s", fmt.Sprintf("%s-%s", k8sIscsi.Name, IscsiInstanceSecretSuffix), err.Error())
	} else {
		for _, d := range secret.Data {
			if d.Key == "username" {
				iscsi.Username = d.Value
			}
			if d.Key == "password" {
				iscsi.Password = d.Value
			}
		}
	}

	return iscsi
}

func checkIscsiExist(cli client.Client, name string) error {
	iscsis := storagev1.IscsiList{}
	err := cli.List(context.TODO(), nil, &iscsis)
	if err != nil {
		return err
	}
	for _, iscsi := range iscsis.Items {
		if iscsi.Name == name {
			return errors.New(fmt.Sprintf("The name of %s iscsi has already exists", name))
		}
	}
	return nil
}

func checkIscsiFinalizers(cli client.Client, name string) error {
	var obj runtime.Object
	obj, err := getIscsi(cli, name)
	if err != nil {
		return err
	}
	metaObj := obj.(metav1.Object)
	finalizers := metaObj.GetFinalizers()
	if (len(finalizers) == 0) || (len(finalizers) == 1 && slice.SliceIndex(finalizers, common.StoragePrestopHookFinalizer) == 0) {
		return nil
	} else {
		return errors.New(fmt.Sprintf("The iscsi %s is used by some pods, you should stop those pods first", name))
	}
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
