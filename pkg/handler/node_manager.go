package handler

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	k8stypes "k8s.io/apimachinery/pkg/types"
	metricsapi "k8s.io/metrics/pkg/apis/metrics"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/helper"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/pkg/zke"
)

type NodeManager struct {
	api.DefaultHandler
	clusters *ClusterManager
}

func newNodeManager(clusters *ClusterManager) *NodeManager {
	return &NodeManager{clusters: clusters}
}

func (m *NodeManager) Get(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	node := ctx.Object.(*types.Node)
	cli := cluster.KubeClient
	k8sNode, err := getK8SNode(cli, node.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Warnf("get node info failed:%s", err.Error())
		}
		return nil
	}

	name := node.GetID()
	return k8sNodeToSCNode(k8sNode, getNodeMetrics(cli, name), getPodCountOnNode(cli, name))
}

func (m *NodeManager) List(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	nodes, _ := getNodes(cluster.KubeClient)
	return nodes
}

func (m *NodeManager) Create(ctx *resttypes.Context, yaml []byte) (interface{}, *resttypes.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}
	inner := ctx.Object.(*types.Node)
	if len(inner.Name) == 0 || len(inner.Roles) == 0 || len(inner.Address) == 0 {
		return nil, resttypes.NewAPIError(resttypes.NotNullable, "node name address and roles can't be null")
	}

	if err := m.clusters.ZKE.UpdateForAddNode(cluster.Name, inner); err != nil {
		return nil, resttypes.NewAPIError(resttypes.InvalidOption, fmt.Sprintf("zke err %s", err))
	}

	inner.SetID(inner.Name)
	inner.SetCreationTimestamp(time.Now())
	return inner, nil
}

func (m *NodeManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}
	zkeCluster := m.clusters.ZKE.Get(cluster.Name)
	if zkeCluster.Status == zke.ClusterCreateing || zkeCluster.Status == zke.ClusterUpateing {
		return resttypes.NewAPIError(resttypes.PermissionDenied, "cluster is createing or updateing")
	}
	target := ctx.Object.(*types.Node).GetID()

	if err := m.clusters.ZKE.UpdateForDeleteNode(cluster.Name, target); err != nil {
		return resttypes.NewAPIError(resttypes.InvalidOption, fmt.Sprintf("zke err %s", err))
	}
	return nil
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
	if helper.IsNodeReady(k8sNode) == false {
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
	node.SetType(resttypes.GetResourceType(node))
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
	zkeRoles           = []string{
		"controlplane", "etcd", "worker", "edge", "storage",
	}
)

func getRoleFromLabels(labels map[string]string) []string {
	hasLabel := func(lbs map[string]string, lb string) bool {
		v, ok := lbs[lb]
		return ok && v == "true"
	}

	var roles []string
	for _, r := range zkeRoles {
		if hasLabel(labels, zkeRoleLabelPrefix+r) {
			roles = append(roles, r)
		}
	}
	return roles
}
