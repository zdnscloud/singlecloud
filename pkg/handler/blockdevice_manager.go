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
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"net/http"
	"sort"
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
	var res types.BlockDeviceSlice
	url := "/apis/agent.zcloud.cn/v1/blockinfos"
	req, err := http.NewRequest("GET", clusteragent.ClusterAgentServiceHost+url, nil)
	if err != nil {
		return res, err
	}
	resp, err := agent.ProxyRequest(cluster, req)
	if err != nil {
		return res, err
	}
	defer resp.Body.Close()
	var info types.Data
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &info)
	for _, h := range info.Data[0].Hosts {
		blockdevice := types.BlockDevice{
			Host: h,
		}
		res = append(res, blockdevice)
	}
	sort.Sort(res)
	return res, nil
}

func cutInvalid(cli client.Client, resp []types.BlockDevice) []types.BlockDevice {
	res := make([]types.BlockDevice, 0)
	for _, b := range resp {
		if !isValidHost(cli, b.Host.NodeName) {
			continue
		}
		dev := make([]types.Dev, 0)
		for _, d := range b.Host.BlockDevices {
			if !isValidBlockDevice(d) {
				continue
			}
			dev = append(dev, d)
		}
		if len(dev) == 0 {
			continue
		}
		host := types.BlockDevice{
			Host: types.Host{
				NodeName:     b.Host.NodeName,
				BlockDevices: dev,
			},
		}
		res = append(res, host)
	}
	return res
}

func isValidBlockDevice(dev types.Dev) bool {
	if dev.Parted || dev.Filesystem || dev.Mount {
		return false
	}
	return true
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
