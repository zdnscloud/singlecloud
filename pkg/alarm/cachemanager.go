package alarm

import (
	"sync"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/pubsub"
	"github.com/zdnscloud/singlecloud/pkg/eventbus"
	"github.com/zdnscloud/singlecloud/pkg/zke"
)

var eventBus *pubsub.PubSub

const MaxEventCount = 100

type AlarmManager struct {
	lock   sync.Mutex
	caches map[string]*AlarmCache
}

func NewAlarmManager(eBus *pubsub.PubSub) *AlarmManager {
	mgr := &AlarmManager{
		caches: make(map[string]*AlarmCache),
	}
	eventBus = eBus
	go mgr.eventLoop()
	return mgr
}

func (mgr *AlarmManager) eventLoop() {
	clusterEventCh := eventBus.Sub(eventbus.ClusterEvent)
	for {
		event := <-clusterEventCh
		switch e := event.(type) {
		case zke.AddCluster:
			cluster := e.Cluster
			mgr.lock.Lock()
			_, ok := mgr.caches[cluster.Name]
			if ok {
				log.Warnf("event watcher detect duplicate cluster %s", cluster.Name)
			} else {
				cache, err := NewAlarmCache(cluster.Cache, MaxEventCount)
				if err != nil {
					log.Warnf("create event watcher for cluster %s failed: %s", cluster.Name, err.Error())
				} else {
					mgr.caches[cluster.Name] = cache
				}
			}
			mgr.lock.Unlock()
		case zke.DeleteCluster:
			cluster := e.Cluster
			mgr.lock.Lock()
			cache, ok := mgr.caches[cluster.Name]
			if ok {
				cache.Stop()
				delete(mgr.caches, cluster.Name)
			} else {
				log.Warnf("event watcher unknown cluster %s", cluster.Name)
			}
			mgr.lock.Unlock()
		}
	}
}
