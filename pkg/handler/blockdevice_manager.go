package handler

import (
	"context"
	"encoding/json"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cluster-agent/blockdevice"
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

func getAllDevices(cluster string, agent *clusteragent.AgentManager) ([]blockdevice.BlockDevice, error) {
	nets := make([]blockdevice.BlockDevice, 0)
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
		newp := new(blockdevice.BlockDevice)
		p := s.Index(i).Interface()
		tmp, _ := json.Marshal(&p)
		json.Unmarshal(tmp, newp)
		nets = append(nets, *newp)
	}
	return nets, nil

}

func cutInvalid(cli client.Client, resp []blockdevice.BlockDevice) []types.BlockDevice {
	res := make([]types.BlockDevice, 0)
	infos := getStorageUsed(cli)
	for _, h := range resp {
		if !isValidHost(cli, h.NodeName) || len(h.BlockDevices) == 0 {
			continue
		}
		var usedby string
		devs := make([]types.Dev, 0)
		for _, d := range h.BlockDevices {
			used := getUsed(h.NodeName, d, infos)
			if used == "other" {
				continue
			}
			if used != "" {
				usedby = used
			}
			dev := types.Dev{
				Name: d.Name,
				Size: d.Size,
			}
			devs = append(devs, dev)
		}
		if len(devs) == 0 {
			continue
		}
		host := types.BlockDevice{
			NodeName:     h.NodeName,
			BlockDevices: devs,
			UsedBy:       usedby,
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
	_, ok1 := node.Labels["node-role.kubernetes.io/controlplane"]
	_, ok2 := node.Labels["node-role.kubernetes.io/etcd"]
	if ok1 || ok2 {
		return false
	}
	return true
}

func getUsed(host string, dev blockdevice.Dev, infos map[string][]string) string {
	var used string
	info, ok := infos[host]
	if !ok {
		if dev.Parted || dev.Filesystem || dev.Mount {
			return "other"
		}
		return used
	}
	used = "other"
	for _, d := range info {
		if dev.Name != d {
			continue
		}
		used = info[0]
	}
	return used
}

func getStorageUsed(cli client.Client) map[string][]string {
	infos := make(map[string][]string)
	scs, _ := getStorageClusters(cli)
	for _, sc := range scs.Items {
		for _, info := range sc.Status.Config {
			devs := make([]string, 0)
			devs = append(devs, sc.Name)
			for _, d := range info.BlockDevices {
				devs = append(devs, d)
			}
			infos[info.NodeName] = devs
		}
	}
	return infos
}
