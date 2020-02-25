package k8sshell

import (
	"sync"

	"github.com/zdnscloud/cement/log"

	"github.com/zdnscloud/gok8s/exec"
	eb "github.com/zdnscloud/singlecloud/pkg/eventbus"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type ExecutorManager struct {
	lock           sync.Mutex
	executors      map[string]*exec.Executor
	clusterEventCh <-chan interface{}
}

func New() *ExecutorManager {
	mgr := &ExecutorManager{
		executors:      make(map[string]*exec.Executor),
		clusterEventCh: eb.SubscribeResourceEvent(types.Cluster{}),
	}
	go mgr.eventLoop()
	return mgr
}

func (mgr *ExecutorManager) eventLoop() {
	for {
		event := <-mgr.clusterEventCh
		switch e := event.(type) {
		case eb.ResourceCreateEvent:
			cluster := e.Resource.(*types.Cluster)
			mgr.lock.Lock()
			_, ok := mgr.executors[cluster.Name]
			if ok {
				log.Warnf("shell executor detect duplicate cluster %s", cluster.Name)
			} else {
				executor, err := exec.New(cluster.KubeProvider.GetKubeRestConfig(), cluster.KubeProvider.GetKubeClient(), cluster.KubeProvider.GetKubeCache())
				if err != nil {
					log.Warnf("create executor for cluster %s failed: %s", cluster.Name, err.Error())
				} else {
					mgr.executors[cluster.Name] = executor
				}
			}
			mgr.lock.Unlock()
		case eb.ResourceDeleteEvent:
			clusterName := e.Resource.GetID()
			mgr.lock.Lock()
			executor, ok := mgr.executors[clusterName]
			if ok {
				executor.Stop()
				delete(mgr.executors, clusterName)
			} else {
				log.Warnf("event watcher unknown cluster %s", clusterName)
			}
			mgr.lock.Unlock()
		}
	}
}
