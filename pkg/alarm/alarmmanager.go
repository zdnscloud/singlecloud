package alarm

import (
	"fmt"
	"sync"

	"github.com/zdnscloud/cement/log"
	gorestError "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	eb "github.com/zdnscloud/singlecloud/pkg/eventbus"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

var alarmManager *AlarmManager

const (
	MaxAlarmCount = 1000
)

func GetAlarmManager() *AlarmManager {
	return alarmManager
}

type AlarmManager struct {
	lock              sync.Mutex
	cache             *AlarmCache
	clusterEventCache map[string]*EventCache
}

func NewAlarmManager() error {
	alarmCache, err := NewAlarmCache()
	if err != nil {
		return err
	}
	alarmManager = &AlarmManager{
		cache:             alarmCache,
		clusterEventCache: make(map[string]*EventCache),
	}
	go alarmManager.eventLoop()
	return nil
}

func (mgr *AlarmManager) eventLoop() {
	clusterEventCh := eb.SubscribeResourceEvent(types.Cluster{})
	for {
		event := <-clusterEventCh
		switch e := event.(type) {
		case eb.ResourceCreateEvent:
			cluster := e.Resource.(*types.Cluster)
			mgr.lock.Lock()
			mgr.clusterEventCache[cluster.Name] = NewEventCache(cluster.Name, cluster.KubeProvider.GetCache(), mgr.cache)
			mgr.lock.Unlock()
		case eb.ResourceDeleteEvent:
			clusterName := e.Resource.GetID()
			mgr.lock.Lock()
			if cache, ok := mgr.clusterEventCache[clusterName]; ok {
				cache.Stop()
				delete(mgr.clusterEventCache, clusterName)
			} else {
				log.Warnf("can not found event cache for cluster %s", clusterName)
			}
			mgr.lock.Unlock()
		}
	}
}

func (m *AlarmManager) List(ctx *resource.Context) interface{} {
	alarms := make([]*types.Alarm, 0)
	m.cache.lock.RLock()
	for elem := m.cache.alarmList.Back(); elem != nil; elem = elem.Prev() {
		alarms = append(alarms, elem.Value.(*types.Alarm))
	}
	m.cache.lock.RUnlock()
	return alarms
}

func (m *AlarmManager) Update(ctx *resource.Context) (resource.Resource, *gorestError.APIError) {
	alarm := ctx.Resource.(*types.Alarm)
	if err := m.cache.Update(alarm); err != nil {
		return nil, gorestError.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("update alarm id %d to table failed: %s", alarm.UID, err.Error()))
	}
	return alarm, nil
}
