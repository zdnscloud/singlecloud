package services

import (
	"context"

	"github.com/zdnscloud/zke/pkg/docker"
	"github.com/zdnscloud/zke/pkg/hosts"
	"github.com/zdnscloud/zke/types"
)

const (
	NginxProxyEnvName = "CP_HOSTS"
)

func runNginxProxy(ctx context.Context, host *hosts.Host, prsMap map[string]types.PrivateRegistry, proxyProcess types.Process, alpineImage string) error {
	imageCfg, hostCfg, _ := GetProcessConfig(proxyProcess)
	if err := docker.DoRunContainer(ctx, host.DClient, imageCfg, hostCfg, NginxProxyContainerName, host.Address, WorkerRole, prsMap); err != nil {
		return err
	}
	return createLogLink(ctx, host, NginxProxyContainerName, WorkerRole, alpineImage, prsMap)
}

func removeNginxProxy(ctx context.Context, host *hosts.Host) error {
	return docker.DoRemoveContainer(ctx, host.DClient, NginxProxyContainerName, host.Address)
}

func RestartNginxProxy(ctx context.Context, host *hosts.Host) error {
	return docker.DoRestartContainer(ctx, host.DClient, NginxProxyContainerName, host.Address)
}
