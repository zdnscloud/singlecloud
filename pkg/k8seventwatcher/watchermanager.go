package k8seventwatcher

import (
	"sync"

	"github.com/zdnscloud/cement/log"
	eb "github.com/zdnscloud/singlecloud/pkg/eventbus"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const MaxEventCount = 4096

type WatcherManager struct {
	lock           sync.Mutex
	watchers       map[string]*EventWatcher
	clusterEventCh <-chan interface{}
}

func New() *WatcherManager {
	mgr := &WatcherManager{
		watchers:       make(map[string]*EventWatcher),
		clusterEventCh: eb.SubscribeResourceEvent(types.Cluster{}),
	}
	go mgr.eventLoop()
	return mgr
}

func (mgr *WatcherManager) eventLoop() *EventWatcher {
	for {
		event := <-mgr.clusterEventCh
		switch e := event.(type) {
		case eb.ResourceCreateEvent:
			cluster := e.Resource.(*types.Cluster)
			mgr.lock.Lock()
			_, ok := mgr.watchers[cluster.Name]
			if ok {
				log.Warnf("event watcher detect duplicate cluster %s", cluster.Name)
			} else {
				watcher, err := NewEventWatcher(cluster.KubeProvider.GetCache(), MaxEventCount)
				if err != nil {
					log.Warnf("create event watcher for cluster %s failed: %s", cluster.Name, err.Error())
				} else {
					mgr.watchers[cluster.Name] = watcher
				}
			}
			mgr.lock.Unlock()
		case eb.ResourceDeleteEvent:
			clusterName := e.Resource.GetID()
			mgr.lock.Lock()
			watcher, ok := mgr.watchers[clusterName]
			if ok {
				watcher.Stop()
				delete(mgr.watchers, clusterName)
			} else {
				log.Warnf("event watcher unknown cluster %s", clusterName)
			}
			mgr.lock.Unlock()
		}
	}
}
