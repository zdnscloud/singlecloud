package handler

import (
	"context"
	"fmt"
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	resterror "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/kvzoo"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	TopPodCount = 5
)

type NamespaceManager struct {
	clusters *ClusterManager
	db       kvzoo.Table
}

func newNamespaceManager(clusters *ClusterManager) (*NamespaceManager, error) {
	tn, _ := kvzoo.TableNameFromSegments(UserQuotaTable)
	table, err := clusters.GetDB().CreateOrGetTable(tn)
	if err != nil {
		return nil, fmt.Errorf("new namespace manager when create or get userquota table failed: %s", err.Error())
	}
	return &NamespaceManager{clusters: clusters, db: table}, nil
}

func (m *NamespaceManager) Create(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resterror.NewAPIError(resterror.PermissionDenied, "only admin can create namespace")
	}

	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.(*types.Namespace)
	err := createNamespace(cluster.KubeClient, namespace.Name)
	if err == nil {
		namespace.SetID(namespace.Name)
		return namespace, nil
	}

	if apierrors.IsAlreadyExists(err) {
		return nil, resterror.NewAPIError(resterror.DuplicateResource, fmt.Sprintf("duplicate namespace name %s", namespace.Name))
	} else {
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create namespace failed %s", err.Error()))
	}
}

func (m *NamespaceManager) List(ctx *resource.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	k8sNamespaces, err := getNamespaces(cluster.KubeClient)
	if err != nil {
		log.Warnf("get namespace info failed:%s", err.Error())
		return nil
	}

	user := getCurrentUser(ctx)
	var namespaces []*types.Namespace
	for _, ns := range k8sNamespaces.Items {
		if m.clusters.authorizer.Authorize(user, cluster.Name, ns.Name) {
			namespace := k8sNamespaceToSCNamespace(&ns)
			namespaces = append(namespaces, namespace)
		}
	}
	return namespaces
}

func (m *NamespaceManager) Get(ctx *resource.Context) resource.Resource {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Resource.(*types.Namespace)
	if m.clusters.authorizer.Authorize(getCurrentUser(ctx), cluster.Name, namespace.GetID()) == false {
		return nil
	}

	return getNamespaceInfo(cluster.KubeClient, namespace.GetID())
}

func (m *NamespaceManager) Delete(ctx *resource.Context) *resterror.APIError {
	if isAdmin(getCurrentUser(ctx)) == false {
		return resterror.NewAPIError(resterror.PermissionDenied, "only admin can delete namespace")
	}

	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.(*types.Namespace)
	exits, err := m.isExistsInUserQuotaTable(namespace.GetID())
	if err != nil {
		return resterror.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("check exist for namespace %s failed %s", namespace.GetID(), err.Error()))
	}

	if exits {
		return resterror.NewAPIError(resterror.PermissionDenied,
			fmt.Sprintf("can`t delete namespace %s for other user using", namespace.GetID()))
	}

	if err := clearApplications(m.clusters.GetDB(), cluster.KubeClient, cluster.Name, namespace.GetID()); err != nil {
		if apierrors.IsNotFound(err) == false {
			return resterror.NewAPIError(types.ConnectClusterFailed,
				fmt.Sprintf("delete namespace applications failed: %s", err.Error()))
		}
	}

	if err := deleteNamespace(cluster.KubeClient, namespace.GetID()); err != nil {
		if apierrors.IsNotFound(err) {
			return resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("namespace %s desn't exist", namespace.Name))
		} else {
			return resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete namespace failed %s", err.Error()))
		}
	} else {
		if err := clearTransportLayerIngress(cluster.KubeClient, namespace.GetID()); err != nil {
			log.Warnf("clean udp ingress for namespace %s failed:%s", namespace.GetID(), err.Error())
		}
	}
	return nil
}

func (m *NamespaceManager) isExistsInUserQuotaTable(namespace string) (bool, error) {
	tx, err := m.db.Begin()
	if err != nil {
		return false, err
	}

	value, _ := tx.Get(namespace)
	tx.Commit()
	return value != nil, nil
}

func getNamespaces(cli client.Client) (*corev1.NamespaceList, error) {
	namespaces := corev1.NamespaceList{}
	err := cli.List(context.TODO(), nil, &namespaces)
	return &namespaces, err
}

func createNamespace(cli client.Client, name string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	return cli.Create(context.TODO(), ns)
}

func deleteNamespace(cli client.Client, name string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	return cli.Delete(context.TODO(), ns)
}

func getNamespace(cli client.Client, name string) (*corev1.Namespace, error) {
	ns := corev1.Namespace{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{"", name}, &ns)
	if err != nil {
		return nil, err
	} else {
		return &ns, nil
	}
}

func hasNamespace(cli client.Client, name string) bool {
	_, err := getNamespace(cli, name)
	return err == nil
}

func k8sNamespaceToSCNamespace(k8sNamespace *corev1.Namespace) *types.Namespace {
	ns := &types.Namespace{
		Name: k8sNamespace.Name,
	}
	ns.SetID(k8sNamespace.Name)
	ns.SetCreationTimestamp(k8sNamespace.CreationTimestamp.Time)
	ns.SetDeletionTimestamp(k8sNamespace.DeletionTimestamp.Time)
	return ns
}

func getNamespaceInfo(cli client.Client, name string) *types.Namespace {
	ns, err := getNamespace(cli, name)
	if err != nil {
		log.Warnf("get namespace failed:%s", err.Error())
		return nil
	}

	namespace := k8sNamespaceToSCNamespace(ns)

	nodes, err := getNodes(cli)
	if err != nil {
		log.Warnf("get node info failed:%s", err.Error())
		return namespace
	}

	for _, n := range nodes {
		if n.HasRole(types.RoleControlPlane) {
			continue
		}
		namespace.Cpu += n.Cpu
		namespace.Memory += n.Memory
		namespace.Pod += n.Pod
	}

	podMetricsList, err := cli.GetPodMetrics(name, "", labels.Everything())
	if err != nil {
		log.Warnf("get pod metrcis failed:%s", err.Error())
		return namespace
	}

	var podsWithCpuInfo []*types.PodCpuInfo
	var podsWithMemoryInfo []*types.PodMemoryInfo
	for _, pod := range podMetricsList.Items {
		cpuUsed := int64(0)
		memoryUsed := int64(0)
		for _, container := range pod.Containers {
			cpuUsed += container.Usage.Cpu().MilliValue()
			memoryUsed += container.Usage.Memory().Value()
		}
		podsWithCpuInfo = append(podsWithCpuInfo, &types.PodCpuInfo{
			Name:    pod.Name,
			CpuUsed: cpuUsed,
		})

		podsWithMemoryInfo = append(podsWithMemoryInfo, &types.PodMemoryInfo{
			Name:       pod.Name,
			MemoryUsed: memoryUsed,
		})
		namespace.CpuUsed += cpuUsed
		namespace.MemoryUsed += memoryUsed
	}

	sort.Sort(types.PodByCpuUsage(podsWithCpuInfo))
	sort.Sort(types.PodByMemoryUsage(podsWithMemoryInfo))
	if len(podsWithCpuInfo) > 5 {
		podsWithCpuInfo = podsWithCpuInfo[:5]
	}
	if len(podsWithMemoryInfo) > 5 {
		podsWithMemoryInfo = podsWithMemoryInfo[:5]
	}
	namespace.PodsUseMostCPU = podsWithCpuInfo
	namespace.PodsUseMostMemory = podsWithMemoryInfo

	namespace.PodUsed = int64(len(podMetricsList.Items))
	if namespace.Cpu > 0 {
		namespace.CpuUsedRatio = fmt.Sprintf("%.2f", float64(namespace.CpuUsed)/float64(namespace.Cpu))
	}

	if namespace.Memory > 0 {
		namespace.MemoryUsedRatio = fmt.Sprintf("%.2f", float64(namespace.MemoryUsed)/float64(namespace.Memory))
	}

	if namespace.Pod > 0 {
		namespace.PodUsedRatio = fmt.Sprintf("%.2f", float64(namespace.PodUsed)/float64(namespace.Pod))
	}
	return namespace
}

func (m *NamespaceManager) Action(ctx *resource.Context) (interface{}, *resterror.APIError) {
	action := ctx.Resource.GetAction()
	switch action.Name {
	case types.ActionSearchPod:
		return m.searchPod(ctx)
	default:
		return nil, nil
	}
}

func (m *NamespaceManager) searchPod(ctx *resource.Context) (interface{}, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetID()
	if m.clusters.authorizer.Authorize(getCurrentUser(ctx), cluster.Name, namespace) == false {
		return nil, resterror.NewAPIError(resterror.Unauthorized, "user has no permission to access the namespace")
	}

	action := ctx.Resource.GetAction()
	target, ok := action.Input.(*types.PodToSearch)
	if ok == false {
		return nil, resterror.NewAPIError(resterror.InvalidFormat, "search pod param not valid")
	}

	if target.Name == "" {
		return nil, resterror.NewAPIError(resterror.NotNullable, "empty pod name")
	}

	pod := corev1.Pod{}
	err := cluster.KubeClient.Get(context.TODO(), k8stypes.NamespacedName{namespace, target.Name}, &pod)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("search pod get err:%s", err.Error()))
	}

	if len(pod.OwnerReferences) != 1 {
		return map[string]string{
			"kind": "pod",
			"name": pod.Name,
		}, nil
	}

	owner := pod.OwnerReferences[0]
	if owner.Kind != "ReplicaSet" {
		return map[string]string{
			"kind": owner.Kind,
			"name": owner.Name,
		}, nil
	}

	var rs appsv1.ReplicaSet
	err = cluster.KubeClient.Get(context.TODO(), k8stypes.NamespacedName{namespace, owner.Name}, &rs)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get replicaset failed:%s", err.Error()))
	}

	if len(rs.OwnerReferences) != 1 {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("replicaset has %d owners", len(rs.OwnerReferences)))
	}

	owner = rs.OwnerReferences[0]
	return map[string]string{
		"kind": owner.Kind,
		"name": owner.Name,
	}, nil
}
