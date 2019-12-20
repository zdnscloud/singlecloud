package alarm

import (
	"sync"

	"github.com/zdnscloud/cement/log"

	"github.com/zdnscloud/cement/pubsub"
	"github.com/zdnscloud/singlecloud/pkg/eventbus"
	"github.com/zdnscloud/singlecloud/pkg/zke"
)

const MaxEventCount = 100

type WatcherManager struct {
	lock           sync.Mutex
	watchers       map[string]*AlarmWatcher
	clusterEventCh <-chan interface{}
	zcloudEventCh  <-chan interface{}
}

func New(eventBus *pubsub.PubSub) *WatcherManager {
	mgr := &WatcherManager{
		watchers:       make(map[string]*AlarmWatcher),
		clusterEventCh: eventBus.Sub(eventbus.ClusterEvent),
		zcloudEventCh:  eventBus.Sub(eventbus.ZcloudEvent),
	}
	go mgr.eventLoop()
	return mgr
}

func (mgr *WatcherManager) eventLoop() {
	for {
		event := <-mgr.clusterEventCh
		switch e := event.(type) {
		case zke.AddCluster:
			cluster := e.Cluster
			mgr.lock.Lock()
			_, ok := mgr.watchers[cluster.Name]
			if ok {
				log.Warnf("event watcher detect duplicate cluster %s", cluster.Name)
			} else {
				watcher, err := NewAlarmWatcher(cluster.Cache, MaxEventCount, mgr.zcloudEventCh)
				if err != nil {
					log.Warnf("create event watcher for cluster %s failed: %s", cluster.Name, err.Error())
				} else {
					mgr.watchers[cluster.Name] = watcher
				}
			}
			mgr.lock.Unlock()
		case zke.DeleteCluster:
			cluster := e.Cluster
			mgr.lock.Lock()
			watcher, ok := mgr.watchers[cluster.Name]
			if ok {
				watcher.Stop()
				delete(mgr.watchers, cluster.Name)
			} else {
				log.Warnf("event watcher unknown cluster %s", cluster.Name)
			}
			mgr.lock.Unlock()
		}
	}
}
