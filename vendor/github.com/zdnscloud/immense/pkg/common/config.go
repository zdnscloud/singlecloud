package common

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/zdnscloud/cement/set"
	"github.com/zdnscloud/gok8s/client"
	storagev1 "github.com/zdnscloud/immense/pkg/apis/zcloud/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

func AssembleCreateConfig(cli client.Client, cluster *storagev1.Cluster) (*storagev1.Cluster, error) {
	storagecluster, err := GetStorage(cli, cluster.Name)
	if err != nil {
		return cluster, err
	}
	infos := make([]storagev1.HostInfo, 0)
	for _, h := range cluster.Spec.Hosts {
		devs := make([]string, 0)
		exist, devstmp := isExist(h, storagecluster.Status.Config)
		if !exist {
			devstmp, err := GetBlocksFromClusterAgent(cli, h)
			if err != nil {
				return cluster, err
			}
			devs = append(devs, devstmp...)
		} else {
			devs = append(devs, devstmp...)
		}

		info := storagev1.HostInfo{
			NodeName:     h,
			BlockDevices: devs,
		}
		infos = append(infos, info)
	}
	if err := UpdateStorageclusterConfig(cli, cluster.Name, "add", infos); err != nil {
		return cluster, err
	}
	cluster.Status.Config = infos
	return cluster, nil
}

func AssembleDeleteConfig(cli client.Client, cluster *storagev1.Cluster) (*storagev1.Cluster, error) {
	storagecluster, err := GetStorage(cli, cluster.Name)
	if err != nil {
		return cluster, err
	}
	infos := make([]storagev1.HostInfo, 0)
	for _, h := range cluster.Spec.Hosts {
		for _, info := range storagecluster.Status.Config {
			if info.NodeName != h {
				continue
			}
			infos = append(infos, info)
			break
		}
	}
	if err := UpdateStorageclusterConfig(cli, cluster.Name, "del", infos); err != nil {
		return cluster, err
	}
	cluster.Status.Config = infos
	return cluster, nil
}

func AssembleUpdateConfig(cli client.Client, oldc, newc *storagev1.Cluster) (*storagev1.Cluster, *storagev1.Cluster, error) {
	del, add := hostsDiff(oldc.Spec.Hosts, newc.Spec.Hosts)
	delc := &storagev1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: newc.Name,
		},
		Spec: storagev1.ClusterSpec{
			StorageType: newc.Spec.StorageType,
			Hosts:       del,
		},
	}
	addc := &storagev1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: newc.Name,
		},
		Spec: storagev1.ClusterSpec{
			StorageType: newc.Spec.StorageType,
			Hosts:       add,
		},
	}

	dels, err := AssembleDeleteConfig(cli, delc)
	if err != nil {
		return oldc, newc, err
	}
	adds, err := AssembleCreateConfig(cli, addc)
	if err != nil {
		return oldc, newc, err
	}
	return dels, adds, nil
}

func GetBlocksFromClusterAgent(cli client.Client, name string) ([]string, error) {
	devs := make([]string, 0)
	service := corev1.Service{}
	if err := cli.Get(ctx, k8stypes.NamespacedName{StorageNamespace, "cluster-agent"}, &service); err != nil {
		return devs, err
	}
	url := "/apis/agent.zcloud.cn/v1/blockdevices"
	newurl := "http://" + service.Spec.ClusterIP + url
	req, err := http.NewRequest("GET", newurl, nil)
	if err != nil {
		return devs, err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return devs, err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var info Data
	json.Unmarshal(body, &info)
	for _, h := range info.Data {
		if h.NodeName != name {
			continue
		}
		for _, d := range h.BlockDevices {
			if d.Parted || d.Filesystem || d.Mount {
				continue
			}
			devs = append(devs, d.Name)
		}
	}
	return devs, nil
}

func isExist(h string, infos []storagev1.HostInfo) (bool, []string) {
	for _, info := range infos {
		if info.NodeName == h {
			return true, info.BlockDevices
		}
	}
	return false, []string{}
}

func UpdateStorageclusterConfig(cli client.Client, name, action string, infos []storagev1.HostInfo) error {
	storagecluster, err := GetStorage(cli, name)
	if err != nil {
		return err
	}
	oldinfos := storagecluster.Status.Config
	newinfos := make([]storagev1.HostInfo, 0)
	if action == "add" {
		for _, h := range infos {
			var exist bool
			for _, v := range oldinfos {
				if h.NodeName == v.NodeName {
					exist = true
					break
				}
			}
			if !exist {
				newinfos = append(newinfos, h)
			}
		}
	}
	if action == "del" {
		for _, h := range infos {
			for i, v := range oldinfos {
				if h.NodeName != v.NodeName {
					continue
				}
				oldinfos = append(oldinfos[:i], oldinfos[i+1:]...)
				break
			}
		}
	}
	newinfos = append(newinfos, oldinfos...)
	storagecluster.Status.Config = newinfos
	return cli.Update(ctx, &storagecluster)
}

func hostsDiff(oldcfg, newcfg []string) ([]string, []string) {
	oldhosts := set.StringSetFromSlice(oldcfg)
	newhosts := set.StringSetFromSlice(newcfg)
	del := oldhosts.Difference(newhosts).ToSlice()
	add := newhosts.Difference(oldhosts).ToSlice()
	return del, add
}
