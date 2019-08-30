package core

import (
	"context"

	"github.com/zdnscloud/zke/pkg/hosts"
	"github.com/zdnscloud/zke/pkg/util"

	"github.com/zdnscloud/cement/errgroup"
)

func (c *Cluster) CleanDeadLogs(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return util.CancelErr
	default:
		hostList := hosts.GetUniqueHostList(c.EtcdHosts, c.ControlPlaneHosts, c.WorkerHosts, c.EdgeHosts)

		_, err := errgroup.Batch(hostList, func(h interface{}) (interface{}, error) {
			return nil, hosts.DoRunLogCleaner(ctx, h.(*hosts.Host), c.Image.Alpine, c.PrivateRegistriesMap)
		})
		return err
	}
}
