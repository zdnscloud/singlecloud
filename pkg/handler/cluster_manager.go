package handler

import (
	"fmt"

	"github.com/zdnscloud/cement/pubsub"
	"github.com/zdnscloud/cement/slice"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gorest"
	resterr "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/authentication"
	"github.com/zdnscloud/singlecloud/pkg/authorization"
	"github.com/zdnscloud/singlecloud/pkg/clusteragent"
	"github.com/zdnscloud/singlecloud/pkg/eventbus"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/pkg/zke"
	"github.com/zdnscloud/singlecloud/storage"

	"github.com/zdnscloud/cement/log"
)

const (
	ZCloudNamespace = "zcloud"
	ZCloudAdmin     = "zcloud-cluster-admin"
	ZCloudReadonly  = "zcloud-cluster-readonly"
)

type ClusterManager struct {
	eventBus      *pubsub.PubSub
	authorizer    *authorization.Authorizer
	authenticator *authentication.Authenticator
	zkeManager    *zke.ZKEManager
	db            storage.DB
	Agent         *clusteragent.AgentManager
}

func newClusterManager(authenticator *authentication.Authenticator, authorizer *authorization.Authorizer, eventBus *pubsub.PubSub, agent *clusteragent.AgentManager, db storage.DB, scVersion string) (*ClusterManager, error) {
	clusterMgr := &ClusterManager{
		authorizer:    authorizer,
		authenticator: authenticator,
		eventBus:      eventBus,
		db:            db,
		Agent:         agent,
	}
	storageNodeListener := &StorageNodeListener{
		clusters: clusterMgr,
	}
	zkeMgr, err := zke.New(db, scVersion, storageNodeListener)
	if err != nil {
		log.Errorf("create zke-manager failed %s", err.Error())
		return nil, err
	}
	clusterMgr.zkeManager = zkeMgr
	go clusterMgr.eventLoop()
	return clusterMgr, nil
}

func (m *ClusterManager) GetDB() storage.DB {
	return m.db
}

func (m *ClusterManager) GetAuthorizer() *authorization.Authorizer {
	return m.authorizer
}

func (m *ClusterManager) GetEventBus() *pubsub.PubSub {
	return m.eventBus
}

func (m *ClusterManager) GetClusterForSubResource(obj restresource.Resource) *zke.Cluster {
	ancestors := restresource.GetAncestors(obj)
	clusterID := ancestors[0].GetID()
	return m.zkeManager.GetReady(clusterID)
}

func (m *ClusterManager) GetClusterByName(name string) *zke.Cluster {
	return m.zkeManager.GetReady(name)
}

func (m *ClusterManager) Create(ctx *restresource.Context) (restresource.Resource, *resterr.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resterr.NewAPIError(resterr.PermissionDenied, "only admin can create cluster")
	}
	return m.zkeManager.Create(ctx)
}

func (m *ClusterManager) Update(ctx *restresource.Context) (restresource.Resource, *resterr.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resterr.NewAPIError(resterr.PermissionDenied, "only admin can update cluster")
	}
	return m.zkeManager.Update(ctx)
}

func (m *ClusterManager) Get(ctx *restresource.Context) restresource.Resource {
	id := ctx.Resource.GetID()
	if m.authorizer.Authorize(getCurrentUser(ctx), id, "") == false {
		return nil
	}
	cluster := m.zkeManager.Get(id)
	if cluster != nil {
		sc := cluster.ToTypesCluster()
		if cluster.IsReady() {
			return getClusterInfo(cluster.KubeClient, sc)
		}
		return sc
	}
	return nil
}

func getClusterInfo(cli client.Client, sc *types.Cluster) *types.Cluster {
	if cli == nil {
		return sc
	}

	nodes, err := getNodes(cli)
	if err != nil {
		return sc
	}
	sc.NodesCount = len(nodes)
	for _, n := range nodes {
		if n.HasRole(types.RoleControlPlane) {
			continue
		}
		sc.Cpu += n.Cpu
		sc.CpuUsed += n.CpuUsed
		sc.Memory += n.Memory
		sc.MemoryUsed += n.MemoryUsed
		sc.Pod += n.Pod
		sc.PodUsed += n.PodUsed
	}
	sc.CpuUsedRatio = fmt.Sprintf("%.2f", float64(sc.CpuUsed)/float64(sc.Cpu))
	sc.MemoryUsedRatio = fmt.Sprintf("%.2f", float64(sc.MemoryUsed)/float64(sc.Memory))
	sc.PodUsedRatio = fmt.Sprintf("%.2f", float64(sc.PodUsed)/float64(sc.Pod))
	return sc
}

func (m *ClusterManager) List(ctx *restresource.Context) interface{} {
	requestFlags := ctx.Request.URL.Query()
	user := getCurrentUser(ctx)
	var readyClusters []*types.Cluster
	var allClusters []*types.Cluster

	for _, c := range m.zkeManager.List() {
		if m.authorizer.Authorize(user, c.Name, "") {
			sc := getClusterInfo(c.KubeClient, c.ToTypesCluster())
			allClusters = append(allClusters, sc)
			if c.IsReady() {
				readyClusters = append(readyClusters, sc)
			}
		}
	}

	if onlyReady := requestFlags.Get("onlyready"); onlyReady == "true" {
		return readyClusters
	}
	return allClusters
}

func (m *ClusterManager) Delete(ctx *restresource.Context) *resterr.APIError {
	if isAdmin(getCurrentUser(ctx)) == false {
		return resterr.NewAPIError(resterr.PermissionDenied, "only admin can delete cluster")
	}
	id := ctx.Resource.GetID()
	return m.zkeManager.Delete(id)
}

func (m *ClusterManager) Action(ctx *restresource.Context) (interface{}, *resterr.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resterr.NewAPIError(resterr.PermissionDenied, "only admin can call cluster action apis")
	}

	action := ctx.Resource.GetAction()
	id := ctx.Resource.GetID()

	switch action.Name {
	case types.CSCancelAction:
		return m.zkeManager.CancelCluster(id)
	case types.CSImportAction:
		return m.zkeManager.Import(ctx)
	default:
		return nil, nil
	}
}

func (m *ClusterManager) authorizationHandler() gorest.HandlerFunc {
	return func(ctx *restresource.Context) *resterr.APIError {
		if _, ok := ctx.Resource.(*types.User); ok {
			action := ctx.Resource.GetAction()
			if action != nil && action.Name == types.ActionLogin {
				return nil
			}
		}

		user := getCurrentUser(ctx)
		if user == "" {
			return resterr.NewAPIError(resterr.Unauthorized, fmt.Sprintf("user is unknowned"))
		}

		if m.authorizer.GetUser(user) == nil {
			newUser := &types.User{Name: user}
			newUser.SetID(user)
			m.authorizer.AddUser(newUser)
		}

		ancestors := restresource.GetAncestors(ctx.Resource)
		if len(ancestors) < 2 {
			return nil
		}

		if _, ok := ancestors[0].(*types.Cluster); ok {
			if _, ok := ancestors[1].(*types.Namespace); ok {
				cluster := ancestors[0].GetID()
				namespace := ancestors[1].GetID()
				if m.authorizer.Authorize(user, cluster, namespace) == false {
					return resterr.NewAPIError(resterr.PermissionDenied, fmt.Sprintf("user %s has no sufficient permission to work on cluster %s namespace %s", user, cluster, namespace))
				}
			}
		}
		return nil
	}
}

func (m *ClusterManager) eventLoop() {
	for {
		obj := <-m.zkeManager.PubEventCh
		m.eventBus.Pub(obj, eventbus.ClusterEvent)
	}
}

type StorageNodeListener struct {
	clusters *ClusterManager
}

func (m StorageNodeListener) IsStorageNode(clusterName, node string) (bool, error) {
	cluster := m.clusters.zkeManager.Get(clusterName)
	if cluster == nil {
		return false, fmt.Errorf("nil cluster %s", clusterName)
	}
	if !cluster.IsReady() && cluster.KubeClient == nil {
		return false, fmt.Errorf("cluster %s kubeClient is nil", clusterName)
	}
	storageClusters, err := getStorageClusters(cluster.KubeClient)
	if err != nil {
		return true, err
	}
	for _, storageCluster := range storageClusters.Items {
		if slice.SliceIndex(storageCluster.Spec.Hosts, node) >= 0 {
			return true, nil
		}
	}
	return false, nil
}

var _ zke.NodeListener = StorageNodeListener{}
