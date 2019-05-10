package globaldns

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	corev1 "k8s.io/api/core/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/pubsub"
	"github.com/zdnscloud/g53"
	"github.com/zdnscloud/gok8s/cache"

	"github.com/zdnscloud/singlecloud/pkg/eventbus"
	"github.com/zdnscloud/singlecloud/pkg/handler"
)

const (
	KubeSystemNamespace = "kube-system"
	FullClusterState    = "full-cluster-state"
)

var (
	GetFullClusterStateOption = k8stypes.NamespacedName{KubeSystemNamespace, FullClusterState}
)

type GlobalDNS struct {
	clusterEventCh    <-chan interface{}
	clusterDNSSyncers map[string]*ClusterDNSSyncer
	proxy             *DnsProxy
	lock              sync.Mutex
}

func New(httpCmdAddr string, eventBus *pubsub.PubSub) error {
	proxy, err := newDnsProxy(httpCmdAddr)
	if err != nil {
		return err
	}

	gdns := &GlobalDNS{
		clusterEventCh:    eventBus.Sub(eventbus.ClusterEvent),
		clusterDNSSyncers: make(map[string]*ClusterDNSSyncer),
		proxy:             proxy,
	}

	go gdns.eventLoop()
	return nil
}

func (g *GlobalDNS) eventLoop() {
	for {
		event := <-g.clusterEventCh
		switch e := event.(type) {
		case handler.AddCluster:
			cluster := e.Cluster
			g.lock.Lock()
			err := g.newClusterDNSSyncer(cluster.Name, cluster.Cache)
			if err != nil {
				log.Warnf("create globaldns syncer for cluster %s failed: %s", cluster.Name, err.Error())
			}
			g.lock.Unlock()
		case handler.DeleteCluster:
			cluster := e.Cluster
			g.lock.Lock()
			syncer, ok := g.clusterDNSSyncers[cluster.Name]
			if ok {
				syncer.Stop()
				delete(g.clusterDNSSyncers, cluster.Name)
			} else {
				log.Warnf("globaldns syncer is unknown cluster %s", cluster.Name)
			}
			g.lock.Unlock()
		}
	}
}

func (g *GlobalDNS) newClusterDNSSyncer(clusterName string, c cache.Cache) error {
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

	if fullState.DesiredState.ZKEConfig.Services.Kubelet.ClusterDomain == "" {
		return fmt.Errorf("cluster %s zone should not be empty", clusterName)
	}

	zoneNameStr := fullState.DesiredState.ZKEConfig.Services.Kubelet.ClusterDomain
	zoneName, err := g53.NameFromString(zoneNameStr)
	if err != nil {
		return fmt.Errorf("parse cluster %s zone name %s failed: %v", clusterName, zoneNameStr, err.Error())
	}

	for cluster, syncer := range g.clusterDNSSyncers {
		if syncer.GetZoneName().Equals(zoneName) {
			return fmt.Errorf("duplicate cluster zone %v, the zone has been belongs to cluster %v", zoneNameStr, cluster)
		}
	}

	syncer, err := newClusterDNSSyncer(zoneName, c, g.proxy)
	if err != nil {
		return err
	}

	g.clusterDNSSyncers[clusterName] = syncer
	return nil
}
