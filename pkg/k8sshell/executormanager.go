package k8sshell

import (
	"sync"

	"github.com/zdnscloud/cement/log"

	"github.com/zdnscloud/cement/pubsub"
	"github.com/zdnscloud/gok8s/exec"
	"github.com/zdnscloud/singlecloud/pkg/eventbus"
	"github.com/zdnscloud/singlecloud/pkg/zke"
)

type ExecutorManager struct {
	lock           sync.Mutex
	executors      map[string]*exec.Executor
	clusterEventCh <-chan interface{}
}

func New(eventBus *pubsub.PubSub) *ExecutorManager {
	mgr := &ExecutorManager{
		executors:      make(map[string]*exec.Executor),
		clusterEventCh: eventBus.Sub(eventbus.ClusterEvent),
	}
	go mgr.eventLoop()
	return mgr
}

func (mgr *ExecutorManager) eventLoop() {
	for {
		event := <-mgr.clusterEventCh
		switch e := event.(type) {
		case zke.AddCluster:
			cluster := e.Cluster
			mgr.lock.Lock()
			_, ok := mgr.executors[cluster.Name]
			if ok {
				log.Warnf("shell executor detect duplicate cluster %s", cluster.Name)
			} else {
				executor, err := exec.New(cluster.K8sConfig, cluster.KubeClient, cluster.Cache)
				if err != nil {
					log.Warnf("create executor for cluster %s failed: %s", cluster.Name, err.Error())
				} else {
					mgr.executors[cluster.Name] = executor
				}
			}
			mgr.lock.Unlock()
		case zke.DeleteCluster:
			cluster := e.Cluster
			mgr.lock.Lock()
			executor, ok := mgr.executors[cluster.Name]
			if ok {
				executor.Stop()
				delete(mgr.executors, cluster.Name)
			} else {
				log.Warnf("event watcher unknown cluster %s", cluster.Name)
			}
			mgr.lock.Unlock()
		}
	}
}
