package handler

import (
	"path"
	"strings"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gorest/resource"
	ca "github.com/zdnscloud/singlecloud/pkg/clusteragent"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type SvcMeshPodManager struct {
	clusters *ClusterManager
}

func newSvcMeshPodManager(clusters *ClusterManager) *SvcMeshPodManager {
	return &SvcMeshPodManager{clusters: clusters}
}

func (m *SvcMeshPodManager) Get(ctx *resource.Context) resource.Resource {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	pod := &types.SvcMeshPod{}
	if err := ca.GetAgent().GetResource(cluster.Name, genClusterAgentURL(ctx.Request.URL.Path, cluster.Name), pod); err != nil {
		log.Warnf("get svcmeshpod failed:%s", err.Error())
		return nil
	}

	setPodRelativeResourceLink(ctx.Request.URL.Path, ctx.Resource.GetParent().GetParent().GetID(), pod)
	return pod
}

func setPodRelativeResourceLink(reqPath, namespace string, pod *types.SvcMeshPod) {
	workloadLinkPrefix := strings.SplitAfterN(reqPath, "/namespaces/"+namespace+"/svcmeshworkloads/", 2)[0]
	for i, in := range pod.Inbound {
		pod.Inbound[i].Link = path.Join(workloadLinkPrefix, in.WorkloadID, "svcmeshpods", in.ID)
	}

	for i, out := range pod.Outbound {
		pod.Outbound[i].Link = path.Join(workloadLinkPrefix, out.WorkloadID, "svcmeshpods", out.ID)
	}
}
