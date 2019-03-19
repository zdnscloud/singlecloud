package handler

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	k8stypes "k8s.io/apimachinery/pkg/types"
	metricsapi "k8s.io/metrics/pkg/apis/metrics"

	"github.com/zdnscloud/gok8s/client"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/logger"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type NodeManager struct {
	DefaultHandler
	clusters *ClusterManager
}

func newNodeManager(clusters *ClusterManager) *NodeManager {
	return &NodeManager{clusters: clusters}
}

func (m *NodeManager) Get(obj resttypes.Object) interface{} {
	cluster := m.clusters.GetClusterForSubResource(obj)
	if cluster == nil {
		return nil
	}

	node := obj.(*types.Node)
	cli := cluster.KubeClient
	k8sNode, err := getNode(cli, node.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Warn("get node info failed:%s", err.Error())
		}
		return nil
	}

	name := node.GetID()
	return k8sNodeToSCNode(k8sNode, getNodeMetrics(cli, name), getPodCountOnNode(cli, name))
}

func (m *NodeManager) List(obj resttypes.Object) interface{} {
	cluster := m.clusters.GetClusterForSubResource(obj)
	if cluster == nil {
		return nil
	}

	cli := cluster.KubeClient
	k8sNodes, err := getNodes(cli)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Warn("get node info failed:%s", err.Error())
		}
		return nil
	}

	podCountOnNode := getPodCountOnNode(cli, "")
	nodeMetrics := getNodeMetrics(cli, "")
	var nodes []*types.Node
	for _, k8sNode := range k8sNodes.Items {
		nodes = append(nodes, k8sNodeToSCNode(&k8sNode, nodeMetrics, podCountOnNode))
	}
	return nodes
}

func getNode(cli client.Client, name string) (*corev1.Node, error) {
	node := corev1.Node{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{"", name}, &node)
	return &node, err
}

func getNodes(cli client.Client) (*corev1.NodeList, error) {
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

	cpuAva := status.Allocatable.Cpu()
	memoryAva := status.Allocatable.Memory()
	podAva := status.Allocatable.Pods()

	usageMetrics := nodeMetrics[k8sNode.Name]
	cpuUsed := float64(usageMetrics.Usage.Cpu().MilliValue()) / 1000
	memoryUsed := float64(usageMetrics.Usage.Memory().MilliValue()) / 1000
	podUsed := float64(podCountOnNode[k8sNode.Name])

	cpuRatio := fmt.Sprintf("%.2f", cpuUsed/float64(cpuAva.Value()))
	memoryRatio := fmt.Sprintf("%.2f", memoryUsed/(float64(memoryAva.MilliValue())/1000))
	podRatio := fmt.Sprintf("%.2f", podUsed/float64(podAva.Value()))

	nodeInfo := &status.NodeInfo
	os := nodeInfo.OperatingSystem + " " + nodeInfo.KernelVersion
	osImage := nodeInfo.OSImage
	dockderVersion := nodeInfo.ContainerRuntimeVersion

	var roles []string
	v, ok := k8sNode.Labels["node-role.kubernetes.io/controlplane"]
	if ok && (v == "true") {
		roles = append(roles, "controlplane")
	}

	v, ok = k8sNode.Labels["node-role.kubernetes.io/etcd"]
	if ok && (v == "true") {
		roles = append(roles, "etcd")
	}

	v, ok = k8sNode.Labels["node-role.kubernetes.io/worker"]
	if ok && (v == "true") {
		roles = append(roles, "worker")
	}

	node := &types.Node{
		Name:                 host,
		Address:              address,
		Role:                 strings.Join(roles, ","),
		Labels:               k8sNode.Labels,
		Annotations:          k8sNode.Annotations,
		OperatingSystem:      os,
		OperatingSystemImage: osImage,
		DockerVersion:        dockderVersion,
		Cpu:                  cpuAva.String(),
		CpuUsedRatio:         cpuRatio,
		Memory:               memoryAva.String(),
		MemoryUsedRatio:      memoryRatio,
		Pod:                  podAva.String(),
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
