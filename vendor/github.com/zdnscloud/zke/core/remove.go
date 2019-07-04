package core

import (
	"context"

	"github.com/zdnscloud/zke/core/pki"
	"github.com/zdnscloud/zke/core/services"
	"github.com/zdnscloud/zke/pkg/hosts"
	"github.com/zdnscloud/zke/pkg/k8s"
	"github.com/zdnscloud/zke/pkg/log"
	"github.com/zdnscloud/zke/types"

	"github.com/zdnscloud/cement/errgroup"
)

func (c *Cluster) ClusterRemove(ctx context.Context) error {
	if err := c.CleanupNodes(ctx); err != nil {
		return err
	}
	c.CleanupFiles(ctx)
	return nil
}

func cleanUpHosts(ctx context.Context, cpHosts, workerHosts, etcdHosts, edgeHosts []*hosts.Host, cleanerImage string, prsMap map[string]types.PrivateRegistry, externalEtcd bool) error {
	uniqueHosts := hosts.GetUniqueHostList(cpHosts, workerHosts, etcdHosts, edgeHosts)

	_, err := errgroup.Batch(uniqueHosts, func(h interface{}) (interface{}, error) {
		runHost := h.(*hosts.Host)
		return nil, runHost.CleanUpAll(ctx, cleanerImage, prsMap, externalEtcd)
	})
	return err
}

func (c *Cluster) CleanupNodes(ctx context.Context) error {
	externalEtcd := false
	if len(c.Core.Etcd.ExternalURLs) > 0 {
		externalEtcd = true
	}
	// Remove Worker Plane
	if err := services.RemoveWorkerPlane(ctx, c.WorkerHosts, true); err != nil {
		return err
	}
	// Remove Contol Plane
	if err := services.RemoveControlPlane(ctx, c.ControlPlaneHosts, true); err != nil {
		return err
	}

	// Remove Etcd Plane
	if !externalEtcd {
		if err := services.RemoveEtcdPlane(ctx, c.EtcdHosts, true); err != nil {
			return err
		}
	}

	// Clean up all hosts
	return cleanUpHosts(ctx, c.ControlPlaneHosts, c.WorkerHosts, c.EtcdHosts, c.EdgeHosts, c.Image.Alpine, c.PrivateRegistriesMap, externalEtcd)
}

func (c *Cluster) CleanupFiles(ctx context.Context) error {
	pki.RemoveAdminConfig(ctx, pki.KubeAdminConfigName)
	removeStateFile(ctx, pki.StateFileName)
	return nil
}

func (c *Cluster) RemoveOldNodes(ctx context.Context) error {
	nodeList, err := k8s.GetNodeList(c.KubeClient)
	if err != nil {
		return err
	}
	uniqueHosts := hosts.GetUniqueHostList(c.EtcdHosts, c.ControlPlaneHosts, c.WorkerHosts, c.EdgeHosts)
	for _, node := range nodeList.Items {
		if k8s.IsNodeReady(node) {
			continue
		}
		host := &hosts.Host{}
		host.NodeName = node.Name
		if !hosts.IsNodeInList(host, uniqueHosts) {
			if err := k8s.DeleteNode(c.KubeClient, node.Name, ""); err != nil {
				log.Warnf(ctx, "Failed to delete old node [%s] from kubernetes")
			}
		}
	}
	return nil
}
