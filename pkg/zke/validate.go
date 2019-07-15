package zke

import (
	"fmt"

	"github.com/zdnscloud/zke/core/services"
	"github.com/zdnscloud/zke/types"
)

func validateConfig(c *types.ZKEConfig) error {
	if err := validateDuplicateNodes(c); err != nil {
		return err
	}

	if err := validateHostsOptions(c); err != nil {
		return err
	}

	if err := validateControlplaneCount(c); err != nil {
		return err
	}

	if err := validateEtcdCount(c); err != nil {
		return err
	}

	return nil
}

func validateDuplicateNodes(c *types.ZKEConfig) error {
	for i := range c.Nodes {
		for j := range c.Nodes {
			if i == j {
				continue
			}
			if c.Nodes[i].Address == c.Nodes[j].Address {
				return fmt.Errorf("Cluster can't have duplicate node: %s", c.Nodes[i].Address)
			}
			if c.Nodes[i].NodeName == c.Nodes[j].NodeName {
				return fmt.Errorf("Cluster can't have duplicate node: %s", c.Nodes[i].NodeName)
			}
		}
	}
	return nil
}

func validateHostsOptions(c *types.ZKEConfig) error {
	for i, host := range c.Nodes {
		if len(host.Address) == 0 {
			return fmt.Errorf("Address for host (%d) is not provided", i+1)
		}

		if len(host.Role) == 0 {
			return fmt.Errorf("Role for host (%d) is not provided", i+1)
		}

		for _, role := range host.Role {
			if role != services.ETCDRole && role != services.ControlRole && role != services.WorkerRole && role != services.StorageRole && role != services.EdgeRole {
				return fmt.Errorf("Role [%s] for host (%d) is not recognized", role, i+1)
			}
		}
	}
	return nil
}

func validateControlplaneCount(c *types.ZKEConfig) error {
	etcds := []types.ZKEConfigNode{}
	for _, host := range c.Nodes {
		for _, role := range host.Role {
			if role == "etcd" {
				etcds = append(etcds, host)
			}
		}
	}
	if len(etcds) == 0 {
		return fmt.Errorf("Cluster must have at least one etcd plane host: please specify one or more etcd in cluster config")
	}
	return nil
}

func validateEtcdCount(c *types.ZKEConfig) error {
	controlplanes := []types.ZKEConfigNode{}
	for _, host := range c.Nodes {
		for _, role := range host.Role {
			if role == "controlplane" {
				controlplanes = append(controlplanes, host)
			}
		}
	}
	if len(controlplanes) == 0 {
		return fmt.Errorf("Cluster must have at least one controlplane host: please specify one or more etcd in cluster config")
	}
	return nil
}
