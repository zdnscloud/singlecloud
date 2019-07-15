package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/zdnscloud/zke/core"
	"github.com/zdnscloud/zke/core/pki"
	"github.com/zdnscloud/zke/pkg/hosts"
	"github.com/zdnscloud/zke/pkg/log"
	"github.com/zdnscloud/zke/types"

	"github.com/urfave/cli"
)

func RemoveCommand() cli.Command {
	return cli.Command{
		Name:   "remove",
		Usage:  "Teardown the cluster and clean cluster nodes",
		Action: clusterRemoveFromCli,
	}
}

func ClusterRemove(
	ctx context.Context,
	zkeConfig *types.ZKEConfig,
	dialersOptions hosts.DialersOptions) error {
	log.Infof(ctx, "Tearing down Kubernetes cluster")
	kubeCluster, err := core.InitClusterObject(ctx, zkeConfig)
	if err != nil {
		return err
	}
	if err := kubeCluster.SetupDialers(ctx, dialersOptions); err != nil {
		return err
	}
	err = kubeCluster.TunnelHosts(ctx)
	if err != nil {
		return err
	}
	log.Debugf("Starting Cluster removal")
	err = kubeCluster.ClusterRemove(ctx)
	if err != nil {
		return err
	}
	log.Infof(ctx, "Cluster removed successfully")
	return nil
}

func ClusterRemoveWithoutCleanFiles(
	ctx context.Context,
	zkeConfig *types.ZKEConfig,
	dialersOptions hosts.DialersOptions) error {
	log.Infof(ctx, "Tearing down Kubernetes cluster")
	kubeCluster, err := core.InitClusterObject(ctx, zkeConfig)
	if err != nil {
		return err
	}
	if err := kubeCluster.SetupDialers(ctx, dialersOptions); err != nil {
		return err
	}
	err = kubeCluster.TunnelHosts(ctx)
	if err != nil {
		return err
	}
	log.Debugf("Starting Cluster removal")
	err = kubeCluster.CleanupNodes(ctx)
	if err != nil {
		return err
	}
	log.Infof(ctx, "Cluster removed successfully")
	return nil
}

func clusterRemoveFromCli(ctx *cli.Context) error {
	clusterFile, err := resolveClusterFile(pki.ClusterConfig)
	if err != nil {
		return fmt.Errorf("Failed to resolve cluster file: %v", err)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Are you sure you want to remove Kubernetes cluster [y/n]: ")
	input, err := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if err != nil {
		return err
	}
	if input != "y" && input != "Y" {
		return nil
	}

	zkeConfig, err := core.ParseConfig(clusterFile)
	if err != nil {
		return fmt.Errorf("Failed to parse cluster file: %v", err)
	}
	err = validateConfigVersion(zkeConfig)
	if err != nil {
		return err
	}

	return ClusterRemove(context.Background(), zkeConfig, hosts.DialersOptions{})
}

func ClusterRemoveFromRest(ctx context.Context, zkeConfig *types.ZKEConfig, logCh chan string) error {
	SetRestLogCh(logCh)
	return ClusterRemoveWithoutCleanFiles(ctx, zkeConfig, hosts.DialersOptions{})
}
