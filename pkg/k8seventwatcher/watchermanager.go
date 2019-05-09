package k8seventwatcher

import (
	"sync"

	"github.com/zdnscloud/cement/log"

	"github.com/zdnscloud/cement/pubsub"
	"github.com/zdnscloud/singlecloud/pkg/eventbus"
	"github.com/zdnscloud/singlecloud/pkg/handler"
)

const MaxEventCount = 4096

type WatcherManager struct {
	lock           sync.Mutex
	watchers       map[string]*EventWatcher
	clusterEventCh <-chan interface{}
}

func New(eventBus *pubsub.PubSub) *WatcherManager {
	mgr := &WatcherManager{
		watchers:       make(map[string]*EventWatcher),
		clusterEventCh: eventBus.Sub(eventbus.ClusterEvent),
	}
	go mgr.eventLoop()
	return mgr
}

func (mgr *WatcherManager) eventLoop() *EventWatcher {
	for {
		event := <-mgr.clusterEventCh
		switch e := event.(type) {
		case handler.AddCluster:
			cluster := e.Cluster
			mgr.lock.Lock()
			_, ok := mgr.watchers[cluster.Name]
			if ok {
				log.Warnf("event watcher detect duplicate cluster %s", cluster.Name)
			} else {
				watcher, err := NewEventWatcher(cluster.Cache, MaxEventCount)
				if err != nil {
					log.Warnf("create event watcher for cluster %s failed: %s", cluster.Name, err.Error())
				} else {
					mgr.watchers[cluster.Name] = watcher
				}
			}
			mgr.lock.Unlock()
		case handler.DeleteCluster:
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
