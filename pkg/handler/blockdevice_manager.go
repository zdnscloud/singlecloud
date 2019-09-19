package handler

import (
	"context"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	resource "github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/clusteragent"
	"github.com/zdnscloud/singlecloud/pkg/types"
	corev1 "k8s.io/api/core/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

type BlockDeviceManager struct {
	clusters *ClusterManager
}

func newBlockDeviceManager(clusters *ClusterManager) *BlockDeviceManager {
	return &BlockDeviceManager{
		clusters: clusters,
	}
}

func (m *BlockDeviceManager) List(ctx *resource.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	resp, err := getBlockDevices(cluster.Name, cluster.KubeClient, m.clusters.Agent)
	if err != nil {
		log.Warnf("get blockdevices info failed:%s", err.Error())
		return nil
	}
	return resp
}

func getBlockDevices(cluster string, cli client.Client, agent *clusteragent.AgentManager) ([]*types.BlockDevice, error) {
	res := make([]*types.BlockDevice, 0)
	all, err := getAllDevices(cluster, agent)
	if err != nil {
		return res, err
	}
	return cutInvalid(cli, all), nil
}

func getAllDevices(cluster string, agent *clusteragent.AgentManager) ([]types.BlockDevice, error) {
	url := "/apis/agent.zcloud.cn/v1/blockdevices"
	res := make([]types.BlockDevice, 0)
	if err := agent.ListResource(cluster, url, &res); err != nil {
		return res, err
	}
	return res, nil
}

func cutInvalid(cli client.Client, res []types.BlockDevice) []*types.BlockDevice {
	blockdevices := make([]*types.BlockDevice, 0)
	for _, b := range res {
		if !isValidHost(cli, b.NodeName) || len(b.BlockDevices) == 0 {
			continue
		}
		blockdevice := &types.BlockDevice{
			NodeName:     b.NodeName,
			BlockDevices: b.BlockDevices,
		}
		blockdevices = append(blockdevices, blockdevice)
	}
	return blockdevices
}

func isValidHost(cli client.Client, name string) bool {
	node := corev1.Node{}
	if err := cli.Get(context.TODO(), k8stypes.NamespacedName{"", name}, &node); err != nil {
		return false
	}
	_, ok1 := node.Labels["node-role.kubernetes.io/storage"]
	_, ok2 := node.Labels["node-role.kubernetes.io/controlplane"]
	_, ok3 := node.Labels["node-role.kubernetes.io/etcd"]
	if ok1 || ok2 || ok3 {
		return false
	}
	return true
}
