package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"k8s.io/client-go/rest"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/pubsub"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/client/config"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/authentication"
	"github.com/zdnscloud/singlecloud/pkg/authorization"
	"github.com/zdnscloud/singlecloud/pkg/eventbus"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/pkg/zke"
	zkecore "github.com/zdnscloud/zke/core"
	zkepki "github.com/zdnscloud/zke/core/pki"
	zketypes "github.com/zdnscloud/zke/types"
)

const (
	ZCloudNamespace = "zcloud"
	ZCloudAdmin     = "zcloud-cluster-admin"
	ZCloudReadonly  = "zcloud-cluster-readonly"
)

type Cluster struct {
	Name           string
	CreateTime     time.Time
	KubeClient     client.Client
	Cache          cache.Cache
	K8sConfig      *rest.Config
	Status         string
	stopCh         chan struct{}
	State          *zkecore.FullState
	Config         *zketypes.ZKEConfig
	CancelFunction context.CancelFunc
}

type AddCluster struct {
	Cluster *Cluster
}

type DeleteCluster struct {
	Cluster *Cluster
}

type UpdateCluster struct {
	Cluster *Cluster
}
type ClusterManager struct {
	api.DefaultHandler

	lock          sync.Mutex
	clusters      []*Cluster
	eventBus      *pubsub.PubSub
	authorizer    *authorization.Authorizer
	authenticator *authentication.Authenticator
	zkeMsgCh      chan zke.ZKEMsg
}

func newClusterManager(authenticator *authentication.Authenticator, authorizer *authorization.Authorizer, eventBus *pubsub.PubSub) *ClusterManager {

	clusterMgr := &ClusterManager{
		authorizer:    authorizer,
		authenticator: authenticator,
		eventBus:      eventBus,
		zkeMsgCh:      make(chan zke.ZKEMsg),
	}
	go clusterMgr.zkeEventLoop()
	return clusterMgr
}

func (m *ClusterManager) GetClusterForSubResource(obj resttypes.Object) *Cluster {
	ancestors := resttypes.GetAncestors(obj)
	clusterID := ancestors[0].GetID()
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.get(clusterID)
}

func (m *ClusterManager) Create(ctx *resttypes.Context, yamlConf []byte) (interface{}, *resttypes.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resttypes.NewAPIError(resttypes.PermissionDenied, "only admin can create cluster")
	}

	if len(yamlConf) > 0 {
		return m.importExternalCluster(ctx, yamlConf)
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	inner := ctx.Object.(*types.Cluster)
	if c := m.get(inner.Name); c != nil {
		return nil, resttypes.NewAPIError(resttypes.DuplicateResource, "duplicate cluster name")
	}

	cluster := &Cluster{
		Name:       inner.Name,
		CreateTime: time.Now(),
		Status:     zke.ClusterCreateing,
	}

	ctxWithCancel, cancel := context.WithCancel(context.Background())
	cluster.CancelFunction = cancel

	stopCh := make(chan struct{})
	cluster.stopCh = stopCh
	m.clusters = append(m.clusters, cluster)

	zkeEventCh := make(chan zke.ZKEEvent)
	go zke.CreateCluster(ctxWithCancel, zkeEventCh, m.zkeMsgCh)
	zkeEvent := zke.ZKEEvent{
		Config: zke.ScClusterToZKEConfig(inner),
	}
	zkeEventCh <- zkeEvent

	inner.SetID(inner.Name)
	inner.SetType(types.ClusterType)
	inner.Status = types.CSCreateing
	inner.SetCreationTimestamp(cluster.CreateTime)
	return inner, nil
}

func (m *ClusterManager) importExternalCluster(ctx *resttypes.Context, yamlConf []byte) (interface{}, *resttypes.APIError) {
	m.lock.Lock()
	defer m.lock.Unlock()

	inner := ctx.Object.(*types.Cluster)
	if c := m.get(inner.Name); c != nil {
		return nil, resttypes.NewAPIError(resttypes.DuplicateResource, "duplicate cluster name")
	}

	cluster := &Cluster{
		Name:       inner.Name,
		CreateTime: time.Now(),
		Status:     zke.ClusterCreateComplete,
		State:      &zkecore.FullState{},
	}

	if err := json.Unmarshal(yamlConf, cluster.State); err != nil {
		return nil, resttypes.NewAPIError(types.InvalidClusterConfig, fmt.Sprintf("invalid cluster state:%s", err.Error()))
	}
	cluster.State.DesiredState.CertificatesBundle = zkepki.TransformPEMToObject(cluster.State.DesiredState.CertificatesBundle)
	cluster.State.CurrentState.CertificatesBundle = zkepki.TransformPEMToObject(cluster.State.CurrentState.CertificatesBundle)

	cluster.Config = cluster.State.CurrentState.ZKEConfig.DeepCopy()
	k8sConfYaml := cluster.State.CurrentState.CertificatesBundle[zkepki.KubeAdminCertName].Config

	k8sconf, err := config.BuildConfig([]byte(k8sConfYaml))
	if err != nil {
		return nil, resttypes.NewAPIError(types.InvalidClusterConfig, fmt.Sprintf("invalid cluster config:%s", err.Error()))
	}

	cli, err := client.New(k8sconf, client.Options{})
	if err != nil {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("connect to cluster failed:%s", err.Error()))
	}
	cluster.KubeClient = cli

	stopCh := make(chan struct{})
	cache, err := cache.New(k8sconf, cache.Options{})
	if err != nil {
		return nil, resttypes.NewAPIError(types.InvalidClusterConfig, fmt.Sprintf("create cache failed:%s", err.Error()))
	}
	go cache.Start(stopCh)
	cache.WaitForCacheSync(stopCh)

	cluster.Cache = cache
	cluster.K8sConfig = k8sconf
	cluster.stopCh = stopCh
	m.clusters = append(m.clusters, cluster)

	c, err := getClusterInfo(cluster)
	if err != nil {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("get cluster info:%s", err.Error()))
	}

	m.eventBus.Pub(AddCluster{Cluster: cluster}, eventbus.ClusterEvent)
	return c, nil
}

func getClusterInfo(c *Cluster) (*types.Cluster, error) {
	cluster := &types.Cluster{}
	cluster.SetID(c.Name)
	cluster.SetType(types.ClusterType)
	cluster.Name = c.Name
	cluster.SetCreationTimestamp(c.CreateTime)

	switch c.Status {
	case zke.ClusterCreateing:
		cluster.Status = types.CSCreateing
	case zke.ClusterCreateFailed:
		cluster.Status = types.CSCreateFailed
	case zke.ClusterUpateFailed:
		cluster.Status = types.CSUpdateFailed
	case zke.ClusterUpateing:
		cluster.Status = types.CSUpdateing
	default:
		cluster.Status = types.CSUnreachable
	}

	if c.Status == zke.ClusterCreateFailed || c.Status == zke.ClusterCreateing {
		return cluster, fmt.Errorf("cluster %s not yet created", c.Name)
	}

	version, err := c.KubeClient.ServerVersion()
	if err != nil {
		return cluster, err
	}

	cluster.Version = version.GitVersion

	nodes, err := getNodes(c.KubeClient)
	if err != nil {
		return cluster, err
	}
	cluster.NodesCount = len(nodes)
	for _, n := range nodes {
		cluster.Cpu += n.Cpu
		cluster.CpuUsed += n.CpuUsed
		cluster.Memory += n.Memory
		cluster.MemoryUsed += n.MemoryUsed
		cluster.Pod += n.Pod
		cluster.PodUsed += n.PodUsed
	}
	cluster.CpuUsedRatio = fmt.Sprintf("%.2f", float64(cluster.CpuUsed)/float64(cluster.Cpu))
	cluster.MemoryUsedRatio = fmt.Sprintf("%.2f", float64(cluster.MemoryUsed)/float64(cluster.Memory))
	cluster.PodUsedRatio = fmt.Sprintf("%.2f", float64(cluster.PodUsed)/float64(cluster.Pod))
	cluster.Status = types.CSRunning
	return cluster, nil
}

func (m *ClusterManager) Get(ctx *resttypes.Context) interface{} {
	target := ctx.Object.GetID()
	if m.authorizer.Authorize(getCurrentUser(ctx), target, "") == false {
		return nil
	}

	m.lock.Lock()
	cluster := m.get(target)
	m.lock.Unlock()
	if cluster == nil {
		return nil
	}
	info, _ := getClusterInfo(cluster)
	return info
}

func (m *ClusterManager) get(id string) *Cluster {
	for _, c := range m.clusters {
		if c.Name == id {
			return c
		}
	}
	return nil
}

func (m *ClusterManager) List(ctx *resttypes.Context) interface{} {
	user := getCurrentUser(ctx)
	var clusters []*types.Cluster

	m.lock.Lock()
	defer m.lock.Unlock()
	for _, c := range m.clusters {
		if m.authorizer.Authorize(user, c.Name, "") {
			info, _ := getClusterInfo(c)
			clusters = append(clusters, info)
		}
	}
	return clusters
}

func (m *ClusterManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	if isAdmin(getCurrentUser(ctx)) == false {
		return resttypes.NewAPIError(resttypes.PermissionDenied, "only admin can create cluster")
	}

	target := ctx.Object.(*types.Cluster).GetID()
	m.lock.Lock()
	var cluster *Cluster
	for i, c := range m.clusters {
		if c.Name == target {
			if c.Status == zke.ClusterCreateing || c.Status == zke.ClusterUpateing {
				c.CancelFunction()
			}
			cluster = c
			m.clusters = append(m.clusters[:i], m.clusters[i+1:]...)
			break
		}
	}
	m.lock.Unlock()

	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("cluster %s desn't exist", target))
	}
	m.eventBus.Pub(DeleteCluster{Cluster: cluster}, eventbus.ClusterEvent)
	close(cluster.stopCh)
	return nil
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

func (m *ClusterManager) zkeEventLoop() {
	for {
		msg := <-m.zkeMsgCh
		if msg.Error != nil {
			log.Errorf("ZKE:%s", msg.Error)
		}
		m.setClusterAfterCreatedOrUpdated(msg)
	}
}

func (m *ClusterManager) setClusterAfterCreatedOrUpdated(zkeMsg zke.ZKEMsg) error {
	for _, c := range m.clusters {
		if c.Name == zkeMsg.ClusterName {
			m.lock.Lock()
			defer m.lock.Unlock()
			switch zkeMsg.Status {
			case zke.ClusterCreateComplete:
				c.KubeClient = zkeMsg.KubeClient
				c.K8sConfig = zkeMsg.KubeConfig
				c.Status = zkeMsg.Status
				c.Config = zkeMsg.ClusterConfig
				c.State = zkeMsg.ClusterState
				cache, err := cache.New(c.K8sConfig, cache.Options{})
				if err != nil {
					return err
				}
				go cache.Start(c.stopCh)
				cache.WaitForCacheSync(c.stopCh)
				c.Cache = cache
				m.eventBus.Pub(AddCluster{Cluster: c}, eventbus.ClusterEvent)
			case zke.ClusterUpateComplete:
				c.KubeClient = zkeMsg.KubeClient
				c.K8sConfig = zkeMsg.KubeConfig
				c.Status = zkeMsg.Status
				c.Config = zkeMsg.ClusterConfig
				c.State = zkeMsg.ClusterState
				m.eventBus.Pub(UpdateCluster{Cluster: c}, eventbus.ClusterEvent)
			default:
				c.Status = zkeMsg.Status
			}
		}
	}
	return nil
}

func (m *ClusterManager) Action(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	if ctx.Action.Name == types.ClusterCancel {
		return m.cancelBuild(ctx)
	}
	return nil, resttypes.NewAPIError(resttypes.InvalidAction, fmt.Sprintf("action %s is unknown", ctx.Action.Name))
}

func (m *ClusterManager) cancelBuild(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	clusterName := ctx.Object.(*types.Cluster).GetID()

	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resttypes.NewAPIError(resttypes.PermissionDenied, "only admin can cancel")
	}

	m.lock.Lock()
	defer m.lock.Unlock()
	var cluster *Cluster
	for _, c := range m.clusters {
		if c.Name == clusterName {
			cluster = c
		}
		break
	}

	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("cluster %s desn't exist", clusterName))
	}

	switch cluster.Status {
	case zke.ClusterCreateing:
		cluster.CancelFunction()
	case zke.ClusterUpateing:
		cluster.CancelFunction()
	default:
		return nil, resttypes.NewAPIError(resttypes.InvalidAction, "only cluster createing and updateing state can cancel")
	}

	return nil, nil
}
