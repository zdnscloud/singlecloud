package handler

import (
	"fmt"

	"github.com/zdnscloud/singlecloud/pkg/authentication"
	"github.com/zdnscloud/singlecloud/pkg/authorization"
	"github.com/zdnscloud/singlecloud/pkg/eventbus"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/pkg/zke"
	"github.com/zdnscloud/singlecloud/storage"

	"github.com/zdnscloud/cement/pubsub"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
)

const (
	ZCloudNamespace = "zcloud"
	ZCloudAdmin     = "zcloud-cluster-admin"
	ZCloudReadonly  = "zcloud-cluster-readonly"
)

type ClusterManager struct {
	api.DefaultHandler

	eventBus      *pubsub.PubSub
	authorizer    *authorization.Authorizer
	authenticator *authentication.Authenticator
	zkeManager    *zke.ZKEManager
	db            storage.DB
}

func newClusterManager(authenticator *authentication.Authenticator, authorizer *authorization.Authorizer, eventBus *pubsub.PubSub, db storage.DB) *ClusterManager {

	clusterMgr := &ClusterManager{
		authorizer:    authorizer,
		authenticator: authenticator,
		eventBus:      eventBus,
		db:            db,
	}
	zkeMgr, err := zke.New(db)
	if err != nil {
		return clusterMgr
	}
	clusterMgr.zkeManager = zkeMgr
	go clusterMgr.eventLoop()
	return clusterMgr
}

func (m *ClusterManager) GetDB() storage.DB {
	return m.db
}

func (m *ClusterManager) GetAuthorizer() *authorization.Authorizer {
	return m.authorizer
}

func (m *ClusterManager) GetClusterForSubResource(obj resttypes.Object) *zke.Cluster {
	ancestors := resttypes.GetAncestors(obj)
	clusterID := ancestors[0].GetID()
	return m.zkeManager.GetReady(clusterID)
}

func (m *ClusterManager) GetClusterByName(name string) *zke.Cluster {
	return m.zkeManager.GetReady(name)
}

func (m *ClusterManager) Create(ctx *resttypes.Context, yamlConf []byte) (interface{}, *resttypes.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resttypes.NewAPIError(resttypes.PermissionDenied, "only admin can create cluster")
	}
	return m.zkeManager.Create(ctx)
}

func (m *ClusterManager) Update(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resttypes.NewAPIError(resttypes.PermissionDenied, "only admin can create cluster")
	}
	return m.zkeManager.Update(ctx)
}

func (m *ClusterManager) Get(ctx *resttypes.Context) interface{} {
	id := ctx.Object.GetID()
	if m.authorizer.Authorize(getCurrentUser(ctx), id, "") == false {
		return nil
	}
	cluster := m.zkeManager.Get(id)
	if cluster != nil {
		return getClusterInfo(cluster.Client, cluster.Cluster)
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

func (m *ClusterManager) List(ctx *resttypes.Context) interface{} {
	requestFlags := ctx.Request.URL.Query()
	user := getCurrentUser(ctx)
	var clusters []*types.Cluster

	if onlyReady := requestFlags.Get("onlyready"); onlyReady == "true" {
		for _, c := range m.zkeManager.ListReady() {
			if m.authorizer.Authorize(user, c.Cluster.Name, "") {
				sc := getClusterInfo(c.Client, c.Cluster)
				clusters = append(clusters, sc)
			}
		}
		return clusters
	}

	for _, c := range m.zkeManager.ListAll() {
		if m.authorizer.Authorize(user, c.Cluster.Name, "") {
			sc := getClusterInfo(c.Client, c.Cluster)
			clusters = append(clusters, sc)
		}
	}
	return clusters
}

func (m *ClusterManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	if isAdmin(getCurrentUser(ctx)) == false {
		return resttypes.NewAPIError(resttypes.PermissionDenied, "only admin can delete cluster")
	}
	id := ctx.Object.(*types.Cluster).GetID()
	return m.zkeManager.Delete(id)
}

func (m *ClusterManager) Action(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	if ctx.Action.Name == types.ClusterCancel {
		id := ctx.Object.(*types.Cluster).GetID()
		return m.zkeManager.Cancel(id)
	}
	return nil, resttypes.NewAPIError(resttypes.InvalidAction, fmt.Sprintf("action %s is unknown", ctx.Action.Name))
}

func (m *ClusterManager) authorizationHandler() api.HandlerFunc {
	return func(ctx *resttypes.Context) *resttypes.APIError {
		if ctx.Object.GetType() == types.UserType {
			if ctx.Action != nil && ctx.Action.Name == types.ActionLogin {
				return nil
			}
		}

		user := getCurrentUser(ctx)
		if user == "" {
			return resttypes.NewAPIError(resttypes.Unauthorized, fmt.Sprintf("user is unknowned"))
		}

		if m.authorizer.GetUser(user) == nil {
			newUser := &types.User{Name: user}
			newUser.SetID(user)
			m.authorizer.AddUser(newUser)
		}

		ancestors := resttypes.GetAncestors(ctx.Object)
		if len(ancestors) < 2 {
			return nil
		}

		if ancestors[0].GetType() == types.ClusterType && ancestors[1].GetType() == types.NamespaceType {
			cluster := ancestors[0].GetID()
			namespace := ancestors[1].GetID()
			if m.authorizer.Authorize(user, cluster, namespace) == false {
				return resttypes.NewAPIError(resttypes.PermissionDenied, fmt.Sprintf("user %s has no sufficient permission to work on cluster %s namespace %s", user, cluster, namespace))
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
