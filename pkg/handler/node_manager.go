package handler

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8stypes "k8s.io/apimachinery/pkg/types"
	metricsapi "k8s.io/metrics/pkg/apis/metrics"

	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/helper"
	resterr "github.com/zdnscloud/gorest/error"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	nodeDrainedNoExecuteTaintKey = "node.zcloud.cn/unexecutable"
)

type NodeManager struct {
	clusters *ClusterManager
}

func newNodeManager(clusters *ClusterManager) *NodeManager {
	return &NodeManager{clusters: clusters}
}

func (m *NodeManager) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	node := ctx.Resource.(*types.Node)
	cli := cluster.GetKubeClient()
	k8sNode, err := getK8SNode(cli, node.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterror.NewAPIError(resterr.ServerError, fmt.Sprintf("get node %s failed:%s", node.GetID(), err.Error()))
		}
		return nil, nil
	}

	name := node.GetID()
	return k8sNodeToSCNode(k8sNode, getNodeMetrics(cli, name), getPodCountOnNode(cli, name)), nil
}

func (m *NodeManager) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	nodes, err := getNodes(cluster.GetKubeClient())
	if err != nil {
		return nil, resterror.NewAPIError(resterr.ServerError, fmt.Sprintf("list nodes failed:%s", err.Error()))
	}

	return nodes, nil
}

func getNodes(cli client.Client) ([]*types.Node, error) {
	k8sNodes, err := getK8SNodes(cli)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		} else {
			return nil, err
		}
	}

	podCountOnNode := getPodCountOnNode(cli, "")
	nodeMetrics := getNodeMetrics(cli, "")
	var nodes []*types.Node
	for _, k8sNode := range k8sNodes.Items {
		nodes = append(nodes, k8sNodeToSCNode(&k8sNode, nodeMetrics, podCountOnNode))
	}
	return nodes, nil
}

func getK8SNode(cli client.Client, name string) (*corev1.Node, error) {
	node := corev1.Node{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{"", name}, &node)
	return &node, err
}

func getK8SNodes(cli client.Client) (*corev1.NodeList, error) {
	nodes := corev1.NodeList{}
	err := cli.List(context.TODO(), nil, &nodes)
	return &nodes, err
}

func k8sNodeToSCNode(k8sNode *corev1.Node, nodeMetrics map[string]metricsapi.NodeMetrics, podCountOnNode map[string]int) *types.Node {
	status := &k8sNode.Status

	var address, host string
	for _, addr := range status.Addresses {
		if addr.Type == corev1.NodeHostName {
			host = addr.Address
		} else if addr.Type == corev1.NodeInternalIP || addr.Type == corev1.NodeExternalIP {
			if address == "" {
				address = addr.Address
			}
		}
	}

	cpuAva := status.Allocatable.Cpu().MilliValue()
	memoryAva := status.Allocatable.Memory().Value()
	podAva := status.Allocatable.Pods().Value()

	usageMetrics := nodeMetrics[k8sNode.Name]
	cpuUsed := usageMetrics.Usage.Cpu().MilliValue()
	memoryUsed := usageMetrics.Usage.Memory().Value()
	podUsed := int64(podCountOnNode[k8sNode.Name])

	cpuRatio := fmt.Sprintf("%.2f", float64(cpuUsed)/float64(cpuAva))
	memoryRatio := fmt.Sprintf("%.2f", float64(memoryUsed)/float64(memoryAva))
	podRatio := fmt.Sprintf("%.2f", float64(podUsed)/float64(podAva))

	nodeInfo := &status.NodeInfo
	os := nodeInfo.OperatingSystem + " " + nodeInfo.KernelVersion
	osImage := nodeInfo.OSImage
	dockderVersion := nodeInfo.ContainerRuntimeVersion
	nodeStatus := types.NSReady
	if isNodeDrained(k8sNode) {
		nodeStatus = types.NSDrained
	} else if isNodeCordoned(k8sNode) {
		nodeStatus = types.NSCordoned
	} else if !helper.IsNodeReady(k8sNode) {
		nodeStatus = types.NSNotReady
	}

	node := &types.Node{
		Name:                 host,
		Status:               nodeStatus,
		Address:              address,
		Roles:                getRoleFromLabels(k8sNode.Labels),
		Labels:               k8sNode.Labels,
		Annotations:          k8sNode.Annotations,
		OperatingSystem:      os,
		OperatingSystemImage: osImage,
		DockerVersion:        dockderVersion,
		Cpu:                  cpuAva,
		CpuUsed:              cpuUsed,
		CpuUsedRatio:         cpuRatio,
		Memory:               memoryAva,
		MemoryUsed:           memoryUsed,
		MemoryUsedRatio:      memoryRatio,
		Pod:                  podAva,
		PodUsed:              podUsed,
		PodUsedRatio:         podRatio,
	}
	node.SetID(node.Name)
	node.SetCreationTimestamp(k8sNode.CreationTimestamp.Time)
	if k8sNode.GetDeletionTimestamp() != nil {
		node.SetDeletionTimestamp(k8sNode.DeletionTimestamp.Time)
	}
	return node
}

func getPodCountOnNode(cli client.Client, name string) map[string]int {
	podCountOnNode := make(map[string]int)

	pods := corev1.PodList{}
	err := cli.List(context.TODO(), nil, &pods)
	if err == nil {
		for _, p := range pods.Items {
			if p.Status.Phase != corev1.PodRunning {
				continue
			}

			n := p.Spec.NodeName
			if name != "" && n != name {
				continue
			}
			podCountOnNode[n] = podCountOnNode[n] + 1
		}
	}
	return podCountOnNode
}

func getNodeMetrics(cli client.Client, name string) map[string]metricsapi.NodeMetrics {
	nodeMetricsByName := make(map[string]metricsapi.NodeMetrics)
	nodeMetricsList, err := cli.GetNodeMetrics(name, labels.Everything())
	if err == nil {
		for _, metrics := range nodeMetricsList.Items {
			nodeMetricsByName[metrics.Name] = metrics
		}
	}
	return nodeMetricsByName
}

var (
	zkeRoleLabelPrefix = "node-role.kubernetes.io/"
	zkeRoles           = []types.NodeRole{
		types.RoleControlPlane, types.RoleWorker, types.RoleEdge, types.RoleStorage,
	}
)

func getRoleFromLabels(labels map[string]string) []types.NodeRole {
	hasLabel := func(lbs map[string]string, lb string) bool {
		v, ok := lbs[lb]
		return ok && v == "true"
	}

	var roles []types.NodeRole
	for _, r := range zkeRoles {
		if hasLabel(labels, zkeRoleLabelPrefix+string(r)) {
			roles = append(roles, r)
		}
	}
	return roles
}

func (m *NodeManager) Action(ctx *restresource.Context) (interface{}, *resterr.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resterr.NewAPIError(resterr.PermissionDenied, "only admin can call node action api")
	}

	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, "cluster doesn't exist")
	}
	action := ctx.Resource.GetAction()
	node := ctx.Resource.GetID()

	switch action.Name {
	case types.NodeCordon:
		return nil, cordonNode(cluster.GetKubeClient(), node)
	case types.NodeUnCordon:
		return nil, uncordonNode(cluster.GetKubeClient(), node)
	case types.NodeDrain:
		return nil, drainNode(cluster.GetKubeClient(), node)
	default:
		return nil, nil
	}
}

func cordonNode(cli client.Client, name string) *resterr.APIError {
	node, err := getK8sNodeIfNotControlplaneOrStorage(cli, name)
	if err != nil {
		return err
	}

	if isNodeCordoned(node) || isNodeDrained(node) {
		return resterr.NewAPIError(resterr.PermissionDenied, fmt.Sprintf("node %s is already cordoned or drained", name))
	}

	node.Spec.Unschedulable = true
	if err := cli.Update(context.TODO(), node); err != nil {
		return resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("update node %s failed %s", name, err.Error()))
	}
	return nil
}

func drainNode(cli client.Client, name string) *resterr.APIError {
	node, err := getK8sNodeIfNotControlplaneOrStorage(cli, name)
	if err != nil {
		return err
	}

	if isNodeDrained(node) {
		return resterr.NewAPIError(resterr.PermissionDenied, fmt.Sprintf("node %s is already drained", name))
	}

	node.Spec.Taints = append(node.Spec.Taints, corev1.Taint{
		Key:    nodeDrainedNoExecuteTaintKey,
		Effect: corev1.TaintEffectNoExecute,
		TimeAdded: &metav1.Time{
			Time: time.Now(),
		},
	})

	node.Spec.Unschedulable = true
	if err := cli.Update(context.TODO(), node); err != nil {
		return resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("update node %s failed %s", name, err.Error()))
	}
	return nil
}

func uncordonNode(cli client.Client, name string) *resterr.APIError {
	node, err := getK8sNodeIfNotControlplaneOrStorage(cli, name)
	if err != nil {
		return err
	}

	if isNodeDrained(node) {
		for i, t := range node.Spec.Taints {
			if t.Key == nodeDrainedNoExecuteTaintKey {
				node.Spec.Taints = append(node.Spec.Taints[:i], node.Spec.Taints[i+1:]...)
				break
			}
		}
	}

	if isNodeCordoned(node) {
		node.Spec.Unschedulable = false
	} else {
		return resterr.NewAPIError(resterr.PermissionDenied, fmt.Sprintf("node %s isn't cordoned", name))
	}

	if err := cli.Update(context.TODO(), node); err != nil {
		return resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("update node %s failed %s", name, err.Error()))
	}
	return nil
}

func isNodeCordoned(node *corev1.Node) bool {
	return node.Spec.Unschedulable
}

func isNodeDrained(node *corev1.Node) bool {
	for _, t := range node.Spec.Taints {
		if t.Key == nodeDrainedNoExecuteTaintKey {
			return true
		}
	}
	return false
}

func getK8sNodeIfNotControlplaneOrStorage(cli client.Client, name string) (*corev1.Node, *resterr.APIError) {
	node, err := getK8SNode(cli, name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterr.NewAPIError(resterr.NotFound, fmt.Sprintf("node %s desn't exist", name))
		}
		return nil, resterr.NewAPIError(resterr.ServerError, err.Error())
	}

	if checkNodeByRole(node, types.RoleControlPlane) {
		return nil, resterr.NewAPIError(resterr.PermissionDenied, "can not cordon|drain|uncordon a controlplane node")
	}

	if checkNodeByRole(node, types.RoleStorage) {
		return nil, resterr.NewAPIError(resterr.PermissionDenied, "can not cordon|drain|uncordon a storage node")
	}
	return node, nil
}

func checkNodeByRole(node *corev1.Node, role types.NodeRole) bool {
	n := types.Node{
		Roles: getRoleFromLabels(node.Labels),
	}
	if n.HasRole(role) {
		return true
	}
	return false
}
