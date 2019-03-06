package k8smanager

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/zdnscloud/gok8s/client"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/logger"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type NodeManager struct {
	cluster *types.Cluster
}

func newNodeManager(cluster *types.Cluster) NodeManager {
	return NodeManager{cluster: cluster}
}

func (m NodeManager) List() interface{} {
	k8sNodes, err := getNodes(m.cluster.KubeClient)
	if err != nil {
		logger.Warn("get node info failed:%s", err.Error())
		return nil
	}

	var nodes []*types.Node
	for _, k8sNode := range k8sNodes.Items {
		nodes = append(nodes, k8sNodeToSCNode(&k8sNode))
	}
	return nodes
}

func getNodes(cli client.Client) (*corev1.NodeList, error) {
	nodes := corev1.NodeList{}
	err := cli.List(context.TODO(), nil, &nodes)
	return &nodes, err
}

func k8sNodeToSCNode(k8sNode *corev1.Node) *types.Node {
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

	var cpu, memory, storage string
	var podCount int
	for typ, c := range status.Capacity {
		if typ == corev1.ResourceCPU {
			cpu = c.String()
		} else if typ == corev1.ResourceMemory {
			memory = c.String()
		} else if typ == corev1.ResourceEphemeralStorage {
			storage = c.String()
		} else if typ == corev1.ResourcePods {
			v, _ := c.AsInt64()
			podCount = int(v)
		}
	}

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
		Cpu:                  cpu,
		Memory:               memory,
		Storage:              storage,
		PodCount:             podCount,
		CreationTimestamp:    k8sNode.CreationTimestamp.String(),
	}
	node.SetID(node.Name)
	node.SetType(resttypes.GetResourceType(node))
	return node
}
