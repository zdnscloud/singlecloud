package hosts

import (
	"context"

	"github.com/zdnscloud/zke/pkg/docker"
	"github.com/zdnscloud/zke/types"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
)

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

func CleanHeritageStorge(ctx context.Context, h *Host, removeImage string, prsMap map[string]types.PrivateRegistry) error {
	imageCfg := &container.Config{
		Image: removeImage,
		Tty:   true,
		Cmd: []string{
			"/bin/sh",
			"-x",
			"/remove.sh",
		},
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

	if err := docker.DoRunContainer(ctx, h.DClient, imageCfg, hostCfg, "zke-storge-remover", h.Address, "cleanup", prsMap); err != nil {
		return err
	}

	_, err := docker.WaitForContainer(ctx, h.DClient, h.Address, "zke-storge-remover")
	if err != nil {
		return err
	}

	return docker.DoRemoveContainer(ctx, h.DClient, "zke-storge-remover", h.Address)
}
