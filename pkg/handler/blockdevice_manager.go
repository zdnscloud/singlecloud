package handler

import (
	"context"
	"encoding/json"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/clusteragent"
	"github.com/zdnscloud/singlecloud/pkg/types"
	corev1 "k8s.io/api/core/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"reflect"
)

type BlockDeviceManager struct {
	api.DefaultHandler
	clusters *ClusterManager
}

func newBlockDeviceManager(clusters *ClusterManager) *BlockDeviceManager {
	return &BlockDeviceManager{
		clusters: clusters,
	}
}

func (m *BlockDeviceManager) List(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
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

func getBlockDevices(cluster string, cli client.Client, agent *clusteragent.AgentManager) ([]types.BlockDevice, error) {
	res := make([]types.BlockDevice, 0)
	all, err := getAllDevices(cluster, agent)
	if err != nil {
		return res, err
	}
	return cutInvalid(cli, all), nil
}

func getAllDevices(cluster string, agent *clusteragent.AgentManager) ([]types.BlockDevice, error) {
	nets := make([]types.BlockDevice, 0)
	url := "/apis/agent.zcloud.cn/v1/blockdevices"
	res, err := agent.GetData(cluster, url)
	if err != nil {
		return nets, err
	}
	if res == nil {
		return nets, err
	}
	s := reflect.ValueOf(res)
	if s.Len() == 0 {
		return nets, nil
	}
	for i := 0; i < s.Len(); i++ {
		newp := new(types.BlockDevice)
		p := s.Index(i).Interface()
		tmp, _ := json.Marshal(&p)
		json.Unmarshal(tmp, newp)
		nets = append(nets, *newp)
	}
	return nets, nil

}

func cutInvalid(cli client.Client, resp []types.BlockDevice) []types.BlockDevice {
	res := make([]types.BlockDevice, 0)
	for _, b := range resp {
		if !isValidHost(cli, b.NodeName) || len(b.BlockDevices) == 0 {
			continue
		}
		host := types.BlockDevice{
			NodeName:     b.NodeName,
			BlockDevices: b.BlockDevices,
		}
		res = append(res, host)
	}
	return res
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
