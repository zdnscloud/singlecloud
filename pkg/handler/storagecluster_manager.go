package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/zdnscloud/cement/set"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/runtime"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/helper"
	gorestError "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	storagev1 "github.com/zdnscloud/immense/pkg/apis/zcloud/v1"
	"github.com/zdnscloud/singlecloud/pkg/clusteragent"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/pkg/zke"
)

type StorageClusterManager struct {
	clusters *ClusterManager
}

func newStorageClusterManager(clusters *ClusterManager) *StorageClusterManager {
	return &StorageClusterManager{
		clusters: clusters,
	}
}

func (m *StorageClusterManager) List(ctx *resource.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	k8sStorageClusters, err := getStorageClusters(cluster.KubeClient)
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("list storagecluster info failed:%s", err.Error())
		}
		return nil
	}

	var storageclusters []*types.StorageCluster
	for _, item := range k8sStorageClusters.Items {
		storageclusters = append(storageclusters, k8sStorageToSCStorage(cluster, &item))
	}
	return storageclusters
}

func (m StorageClusterManager) Get(ctx *resource.Context) resource.Resource {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	storagecluster := ctx.Resource.(*types.StorageCluster)
	k8sStorageCluster, err := getStorageCluster(cluster.KubeClient, storagecluster.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("get storagecluster info failed:%s", err.Error())
		}
		return nil
	}

	return k8sStorageToSCStorageDetail(cluster, m.clusters.Agent, k8sStorageCluster)
}

func (m StorageClusterManager) Delete(ctx *resource.Context) *gorestError.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return gorestError.NewAPIError(gorestError.NotFound, "storagecluster doesn't exist")
	}

	storagecluster := ctx.Resource.(*types.StorageCluster)
	if err := deleteStorageCluster(cluster.KubeClient, storagecluster.GetID()); err != nil {
		if apierrors.IsNotFound(err) {
			return gorestError.NewAPIError(gorestError.NotFound, fmt.Sprintf("storagecluster %s doesn't exist", storagecluster.GetID()))
		} else if strings.Contains(err.Error(), "is used by") {
			return gorestError.NewAPIError(types.InvalidClusterConfig, fmt.Sprintf("delete storagecluster failed, %s", err.Error()))
		} else {
			return gorestError.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete storagecluster failed, %s", err.Error()))
		}
	}
	return nil
}

func (m StorageClusterManager) Create(ctx *resource.Context) (resource.Resource, *gorestError.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, gorestError.NewAPIError(gorestError.NotFound, "cluster doesn't exist")
	}

	storagecluster := ctx.Resource.(*types.StorageCluster)
	if err := createStorageCluster(cluster.KubeClient, storagecluster); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, gorestError.NewAPIError(gorestError.DuplicateResource, fmt.Sprintf("duplicate storagecluster name %s", storagecluster.Name))
		} else if strings.Contains(err.Error(), "storagecluster has already exists") {
			return nil, gorestError.NewAPIError(types.InvalidClusterConfig, fmt.Sprintf("create storagecluster failed %s", err.Error()))
		} else {
			return nil, gorestError.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create storagecluster failed %s", err.Error()))
		}
	}
	storagecluster.SetID(storagecluster.Name)
	return storagecluster, nil
}

func (m StorageClusterManager) Update(ctx *resource.Context) (resource.Resource, *gorestError.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, gorestError.NewAPIError(gorestError.NotFound, "cluster doesn't exist")
	}

	storagecluster := ctx.Resource.(*types.StorageCluster)
	if err := updateStorageCluster(cluster.KubeClient, storagecluster); err != nil {
		if strings.Contains(err.Error(), "storagecluster must keep") || strings.Contains(err.Error(), "is used by") {
			return nil, gorestError.NewAPIError(types.InvalidClusterConfig, fmt.Sprintf("update storagecluster failed, %s", err.Error()))
		} else {
			return nil, gorestError.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("update storagecluster failed, %s", err.Error()))
		}
	}
	return storagecluster, nil
}

func getStorageCluster(cli client.Client, name string) (*storagev1.Cluster, error) {
	storagecluster := storagev1.Cluster{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{"", name}, &storagecluster)
	return &storagecluster, err
}

func getStorageClusters(cli client.Client) (*storagev1.ClusterList, error) {
	storageclusters := storagev1.ClusterList{}
	err := cli.List(context.TODO(), nil, &storageclusters)
	return &storageclusters, err
}

func deleteStorageCluster(cli client.Client, name string) error {
	storagecluster := &storagev1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	err1 := checkFinalizers(cli, name)
	err2 := cli.Delete(context.TODO(), storagecluster)
	if err2 != nil {
		return err2
	}
	if err1 != nil {
		return err1
	}
	return nil
}

func createStorageCluster(cli client.Client, storagecluster *types.StorageCluster) error {
	if err := checkStorageClusterExist(cli, storagecluster.StorageType); err != nil {
		return err
	}
	k8sStorageCluster := scStorageToK8sStorage(storagecluster)
	return cli.Create(context.TODO(), k8sStorageCluster)
}

func updateStorageCluster(cli client.Client, storagecluster *types.StorageCluster) error {
	if len(storagecluster.Hosts) == 0 {
		return errors.New("update storagecluster failed, storagecluster must keep at least one node,suggest delete the storagecluster")
	}
	if err := checkFinalizerWithHost(cli, storagecluster); err != nil {
		return err
	}

	k8sStorageCluster, err := getStorageCluster(cli, storagecluster.GetID())
	if err != nil {
		return err
	}
	k8sStorageCluster.Spec.Hosts = storagecluster.Hosts
	return cli.Update(context.TODO(), k8sStorageCluster)
}

func scStorageToK8sStorage(storagecluster *types.StorageCluster) *storagev1.Cluster {
	return &storagev1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: types.StorageclusterMap[storagecluster.StorageType],
		},
		Spec: storagev1.ClusterSpec{
			StorageType: storagecluster.StorageType,
			Hosts:       storagecluster.Hosts,
		},
	}
}

func k8sStorageToSCStorage(cluster *zke.Cluster, k8sStorageCluster *storagev1.Cluster) *types.StorageCluster {
	tSize := byteToGb(sToi(k8sStorageCluster.Status.Capacity.Total.Total))
	uSize := byteToGb(sToi(k8sStorageCluster.Status.Capacity.Total.Used))
	fSize := byteToGb(sToi(k8sStorageCluster.Status.Capacity.Total.Free))
	storagecluster := &types.StorageCluster{
		Name:        k8sStorageCluster.Name,
		StorageType: k8sStorageCluster.Spec.StorageType,
		Hosts:       k8sStorageCluster.Spec.Hosts,
		Phase:       k8sStorageCluster.Status.Phase,
		Size:        tSize,
		UsedSize:    uSize,
		FreeSize:    fSize,
	}
	storagecluster.SetID(k8sStorageCluster.Name)
	storagecluster.SetCreationTimestamp(k8sStorageCluster.CreationTimestamp.Time)
	return storagecluster
}

func k8sStorageToSCStorageDetail(cluster *zke.Cluster, agent *clusteragent.AgentManager, k8sStorageCluster *storagev1.Cluster) *types.StorageCluster {
	info, err := getStatusInfo(cluster.Name, agent, k8sStorageCluster.Spec.StorageType)
	if err != nil {
		log.Warnf("get storages from clusteragent failed:%s", err.Error())
	}
	storagecluster := k8sStorageToSCStorage(cluster, k8sStorageCluster)
	storagecluster.Nodes = countSize(k8sStorageCluster)
	storagecluster.PVs = info.PVs
	return storagecluster
}

func getStatusInfo(cluster string, agent *clusteragent.AgentManager, storagetype string) (types.Storage, error) {
	var info types.Storage
	url := "/apis/agent.zcloud.cn/v1/storages/" + storagetype
	req, err := http.NewRequest("GET", clusteragent.ClusterAgentServiceHost+url, nil)
	if err != nil {
		return info, err
	}
	resp, err := agent.ProxyRequest(cluster, req)
	if err != nil {
		return info, err
	}
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &info)
	defer resp.Body.Close()
	return info, nil
}

func checkStorageClusterExist(cli client.Client, storageType string) error {
	storageclusters := storagev1.ClusterList{}
	err := cli.List(context.TODO(), nil, &storageclusters)
	if err != nil {
		return err
	}
	for _, storage := range storageclusters.Items {
		if storage.Spec.StorageType == storageType {
			return errors.New(fmt.Sprintf("The type of %s storagecluster has already exists", storageType))
		}
	}
	return nil
}

func checkFinalizers(cli client.Client, name string) error {
	var obj runtime.Object
	obj, err := getStorageCluster(cli, name)
	if err != nil {
		return err
	}
	metaObj := obj.(metav1.Object)
	if len(metaObj.GetFinalizers()) > 0 {
		return errors.New("The storagecluster is used by some pods, is will be delete until those pods stop")
	}
	return nil
}

func checkFinalizerWithHost(cli client.Client, storagecluster *types.StorageCluster) error {
	hosts := make([]string, 0)
	k8sStoragecluster, err := getStorageCluster(cli, storagecluster.GetID())
	if err != nil {
		return err
	}
	if k8sStoragecluster.Spec.StorageType == "cephfs" {
		return nil
	}
	var obj runtime.Object
	obj = k8sStoragecluster
	metaObj := obj.(metav1.Object)

	ClusterFinalizer := "storage.zcloud.cn/finalizer"
	delhosts := getDelHost(k8sStoragecluster.Spec.Hosts, storagecluster.Hosts)
	for _, host := range delhosts {
		fr := ClusterFinalizer + "-" + host
		if !helper.HasFinalizer(metaObj, fr) {
			continue
		}
		hosts = append(hosts, host)
	}
	if len(hosts) > 0 {
		return errors.New(fmt.Sprintf("The storagehosts %s is used by some pods, you should stop those pods first", hosts))
	}
	return nil
}

func getDelHost(oldhosts, newhosts []string) []string {
	s1 := set.StringSetFromSlice(oldhosts)
	s2 := set.StringSetFromSlice(newhosts)
	return s1.Difference(s2).ToSlice()
}

func sToi(size string) int64 {
	num, _ := strconv.ParseInt(size, 10, 64)
	return num
}

func byteToGb(num int64) string {
	f := float64(num) / (1024 * 1024 * 1024)
	return fmt.Sprintf("%.2f", math.Trunc(f*1e2)*1e-2)
}

func countSize(k8sStorageCluster *storagev1.Cluster) []types.StorageNode {
	var nodes types.StorageNodes
	ns := make(map[string]map[string]int64)
	nodestat := make(map[string]bool)
	stat := true
	for _, i := range k8sStorageCluster.Status.Capacity.Instances {
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
