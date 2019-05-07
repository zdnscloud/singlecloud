package globaldns

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	corev1 "k8s.io/api/core/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/gok8s/cache"
)

const (
	KubeSystemNamespace = "kube-system"
	FullClusterState    = "full-cluster-state"
)

var (
	gGlobalDNS                *GlobalDNS
	GetFullClusterStateOption = k8stypes.NamespacedName{KubeSystemNamespace, FullClusterState}
)

type GlobalDNS struct {
	clusterDNSSyncers map[string]*ClusterDNSSyncer
	proxy             *DnsProxy
	lock              sync.Mutex
}

func Init(httpCmdAddr string) error {
	proxy, err := newDnsProxy(httpCmdAddr)
	if err != nil {
		return err
	}

	gGlobalDNS = &GlobalDNS{
		clusterDNSSyncers: make(map[string]*ClusterDNSSyncer),
		proxy:             proxy,
	}
	return nil
}

func GetGlobalDNS() *GlobalDNS {
	return gGlobalDNS
}

func (g *GlobalDNS) NewClusterDNSSyncer(clusterName string, c cache.Cache) error {
	g.lock.Lock()
	defer g.lock.Unlock()

	if _, ok := g.clusterDNSSyncers[clusterName]; ok {
		return fmt.Errorf("duplicate cluster name %s", clusterName)
	}

	k8sconfigmap := &corev1.ConfigMap{}
	if err := c.Get(context.TODO(), GetFullClusterStateOption, k8sconfigmap); err != nil {
		return fmt.Errorf("get full-cluster-state configmap failed: %s", err.Error())
	}

	fullState := &FullState{}
	if err := json.Unmarshal([]byte(k8sconfigmap.Data[FullClusterState]), fullState); err != nil {
		return fmt.Errorf("unmarshal full-cluster-state configmap failed: %s", err.Error())
	}

	zoneName := fullState.DesiredState.ZKEConfig.Services.Kubelet.ClusterDomain
	if zoneName == "" {
		return fmt.Errorf("cluster %s zone should not be empty", clusterName)
	}

	for cluster, syncer := range g.clusterDNSSyncers {
		if syncer.GetZoneName() == zoneName {
			return fmt.Errorf("duplicate cluster zone %v, the zone has been belongs to cluster %v", zoneName, cluster)
		}
	}

	syncer, err := newClusterDNSSyncer(zoneName, c, g.proxy)
	if err != nil {
		return err
	}

	g.clusterDNSSyncers[clusterName] = syncer
	return nil
}
