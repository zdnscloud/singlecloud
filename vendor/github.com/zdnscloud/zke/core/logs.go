package core

import (
	"context"
	"fmt"

	"github.com/zdnscloud/zke/pkg/hosts"

	"github.com/zdnscloud/cement/errgroup"
)

func (c *Cluster) CleanDeadLogs(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("cluster build has beed canceled")
	default:
		hostList := hosts.GetUniqueHostList(c.EtcdHosts, c.ControlPlaneHosts, c.WorkerHosts, c.EdgeHosts)

		_, err := errgroup.Batch(hostList, func(h interface{}) (interface{}, error) {
			return nil, hosts.DoRunLogCleaner(ctx, h.(*hosts.Host), c.Image.Alpine, c.PrivateRegistriesMap)
		})
		return err
	}
}
