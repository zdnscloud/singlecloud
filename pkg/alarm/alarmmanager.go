package alarm

import (
	"sort"
	"sync"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/pubsub"
	gorestError "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/eventbus"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/pkg/zke"
)

var eventBus *pubsub.PubSub

const MaxEventCount = 100

type AlarmManager struct {
	lock        sync.Mutex
	cache       *AlarmCache
	clusters    map[string]*zke.Cluster
	eventCaches map[string]*EventCache
}

func NewAlarmManager(eBus *pubsub.PubSub) *AlarmManager {
	eventBus = eBus
	mgr := &AlarmManager{
		clusters:    make(map[string]*zke.Cluster),
		eventCaches: make(map[string]*EventCache),
	}
	cache := NewAlarmCache(MaxEventCount, mgr.clusters)
	mgr.cache = cache
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
			mgr.clusters[cluster.Name] = cluster
			mgr.eventCaches[cluster.Name] = NewEventCache(cluster.Name, cluster.Cache, mgr.cache)
			mgr.lock.Unlock()
		case zke.DeleteCluster:
			cluster := e.Cluster
			mgr.lock.Lock()
			_, ok := mgr.clusters[cluster.Name]
			if ok {
				delete(mgr.clusters, cluster.Name)
			} else {
				log.Warnf("event watcher unknown cluster %s", cluster.Name)
			}
			eventCacher, ok := mgr.eventCaches[cluster.Name]
			if ok {
				eventCacher.Stop()
				delete(mgr.eventCaches, cluster.Name)
			}
			mgr.lock.Unlock()
		}
	}
}

func (m *AlarmManager) List(ctx *resource.Context) interface{} {
	var alarms types.Alarms
	elem := m.cache.alarmList.Back()
	if elem == nil {
		return alarms
	}
	for i := 0; i < m.cache.alarmList.Len(); i++ {
		alarms = append(alarms, elem.Value.(*types.Alarm))
		elem = elem.Prev()
	}
	sort.Sort(sort.Reverse(alarms))
	return alarms
}

func (m *AlarmManager) Update(ctx *resource.Context) (resource.Resource, *gorestError.APIError) {
	alarm := ctx.Resource.(*types.Alarm)
	m.cache.lock.Lock()
	defer m.cache.lock.Unlock()
	elem := m.cache.alarmList.Back()
	if elem == nil {
		return nil, nil
	}
	for i := 0; i < m.cache.alarmList.Len(); i++ {
		newAlarm := elem.Value.(*types.Alarm)
		if newAlarm.ID == alarm.ID {
			newAlarm.Acknowledged = true
			m.cache.SetUnAck(-1)
			break
		}
		elem = elem.Prev()
	}
	return alarm, nil
}

func (m *AlarmManager) Get(ctx *resource.Context) resource.Resource {
	alarm := ctx.Resource.(*types.Alarm)
	elem := m.cache.alarmList.Back()
	if elem == nil {
		return nil
	}
	for i := 0; i < m.cache.alarmList.Len(); i++ {
		newAlarm := elem.Value.(*types.Alarm)
		if newAlarm.ID == alarm.ID {
			return newAlarm
		}
	}
	return nil
}
