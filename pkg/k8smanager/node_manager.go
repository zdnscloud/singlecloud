package k8smanager

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/gok8s/client"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/logger"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"k8s.io/apimachinery/pkg/api/resource"
)

type NodeManager struct {
	cluster *types.Cluster
}

func newNodeManager(cluster *types.Cluster) NodeManager {
	return NodeManager{cluster: cluster}
}

func (m NodeManager) Get(node *types.Node) interface{} {
	k8sNode, err := getNode(m.cluster.KubeClient, node.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Warn("get node info failed:%s", err.Error())
		}
		return nil
	}

	return k8sNodeToSCNode(k8sNode)
}

func (m NodeManager) List() interface{} {
	k8sNodes, err := getNodes(m.cluster.KubeClient)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Warn("get node info failed:%s", err.Error())
		}
		return nil
	}

	var nodes []*types.Node
	for _, k8sNode := range k8sNodes.Items {
		nodes = append(nodes, k8sNodeToSCNode(&k8sNode))
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

	var cpuCap, memoryCap, storageCap, podCountCap resource.Quantity
	for typ, c := range status.Capacity {
		if typ == corev1.ResourceCPU {
			cpuCap = c
		} else if typ == corev1.ResourceMemory {
			memoryCap = c
		} else if typ == corev1.ResourceEphemeralStorage {
			storageCap = c
		} else if typ == corev1.ResourcePods {
			podCountCap = c
		}
	}

	var cpuAva, memoryAva, storageAva, podCountAva resource.Quantity
	for typ, c := range status.Allocatable {
		if typ == corev1.ResourceCPU {
			cpuAva = c
		} else if typ == corev1.ResourceMemory {
			memoryAva = c
		} else if typ == corev1.ResourceEphemeralStorage {
			storageAva = c
		} else if typ == corev1.ResourcePods {
			podCountAva = c
		}
	}

	cpuRatio := fmt.Sprintf("%.2f", calculateUsedRatio(&cpuCap, &cpuAva))
	memoryRatio := fmt.Sprintf("%.2f", calculateUsedRatio(&memoryCap, &memoryAva))
	storageRatio := fmt.Sprintf("%.2f", calculateUsedRatio(&storageCap, &storageAva))
	podCountRatio := fmt.Sprintf("%.2f", calculateUsedRatio(&podCountCap, &podCountAva))

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
		Cpu:                  cpuCap.String(),
		CpuUsedRatio:         cpuRatio,
		Memory:               memoryCap.String(),
		MemoryUsedRatio:      memoryRatio,
		Storage:              storageCap.String(),
		StorageUserdRatio:    storageRatio,
		PodCount:             int(podCountCap.Value()),
		PodUsedRatio:         podCountRatio,
	}
	node.SetID(node.Name)
	node.SetCreationTimestamp(k8sNode.CreationTimestamp.Time)
	node.SetType(resttypes.GetResourceType(node))
	return node
}

func calculateUsedRatio(capacity, avail *resource.Quantity) float64 {
	used := capacity.Copy()
	used.Sub(*avail)

	return float64(used.Value()*100) / float64(capacity.Value())
}
