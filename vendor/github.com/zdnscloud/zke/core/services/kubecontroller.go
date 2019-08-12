package services

import (
	"context"

	"github.com/zdnscloud/zke/pkg/docker"
	"github.com/zdnscloud/zke/pkg/hosts"
	"github.com/zdnscloud/zke/types"
)

func runKubeController(ctx context.Context, host *hosts.Host, prsMap map[string]types.PrivateRegistry, controllerProcess types.Process, alpineImage string) error {
	imageCfg, hostCfg, healthCheckURL := GetProcessConfig(controllerProcess)
	if err := docker.DoRunContainer(ctx, host.DClient, imageCfg, hostCfg, KubeControllerContainerName, host.Address, ControlRole, prsMap); err != nil {
		return err
	}
	if err := runHealthcheck(ctx, host, KubeControllerContainerName, healthCheckURL, nil); err != nil {
		return err
	}
	return createLogLink(ctx, host, KubeControllerContainerName, ControlRole, alpineImage, prsMap)
}

func removeKubeController(ctx context.Context, host *hosts.Host) error {
	return docker.DoRemoveContainer(ctx, host.DClient, KubeControllerContainerName, host.Address)
}

func RestartKubeController(ctx context.Context, host *hosts.Host) error {
	return docker.DoRestartContainer(ctx, host.DClient, KubeControllerContainerName, host.Address)
}
