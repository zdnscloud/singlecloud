package hosts

import (
	"context"
	"fmt"

	"github.com/zdnscloud/zke/pkg/docker"
	"github.com/zdnscloud/zke/types"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
)

const ZKERemoverEnvName = "PodCIDR"

func CleanHeritageContainers(ctx context.Context, h *Host) error {
	var op dockertypes.ContainerListOptions
	op.All = true
	containers, err := h.DClient.ContainerList(ctx, op)
	if err != nil {
		return err
	}
	for _, i := range containers {
		ops := dockertypes.ContainerRemoveOptions{
			RemoveVolumes: true,
			RemoveLinks:   false,
			Force:         true,
		}
		err := h.DClient.ContainerRemove(ctx, i.ID, ops)
		if err != nil {
			return err
		}
	}
	return nil
}

func CleanHeritageStorge(ctx context.Context, h *Host, removeImage, clusterCIDR string, prsMap map[string]types.PrivateRegistry) error {
	imageCfg := &container.Config{
		Image: removeImage,
		Tty:   true,
		Env:   []string{fmt.Sprintf("%s=%s", ZKERemoverEnvName, clusterCIDR)},
	}

	hostcfgMounts := []mount.Mount{
		{
			Type:        "bind",
			Source:      "/var/lib",
			Target:      "/var/lib",
			BindOptions: &mount.BindOptions{Propagation: "rshared"},
		},
		{
			Type:        "bind",
			Source:      "/dev",
			Target:      "/dev",
			BindOptions: &mount.BindOptions{Propagation: "rprivate"},
		},
	}
	hostCfg := &container.HostConfig{
		Mounts:      hostcfgMounts,
		Privileged:  true,
		NetworkMode: "host",
	}

	if err := docker.DoRunContainer(ctx, h.DClient, imageCfg, hostCfg, "zke-remover", h.Address, "cleanup", prsMap); err != nil {
		return err
	}

	_, err := docker.WaitForContainer(ctx, h.DClient, h.Address, "zke-remover")
	if err != nil {
		return err
	}

	return docker.DoRemoveContainer(ctx, h.DClient, "zke-remover", h.Address)
}
