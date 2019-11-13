package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/zdnscloud/zke/core"
	"github.com/zdnscloud/zke/core/pki"
	"github.com/zdnscloud/zke/monitor"
	"github.com/zdnscloud/zke/network"
	"github.com/zdnscloud/zke/pkg/hosts"
	"github.com/zdnscloud/zke/pkg/log"
	"github.com/zdnscloud/zke/types"
	"github.com/zdnscloud/zke/zcloud"

	"github.com/urfave/cli"
	"github.com/zdnscloud/cement/errgroup"
	cementlog "github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client/config"
	"k8s.io/client-go/kubernetes"
)

func UpCommand() cli.Command {
	return cli.Command{
		Name:   "up",
		Usage:  "Bring the cluster up",
		Action: clusterUpFromCli,
	}
}

func ClusterUp(ctx context.Context, dialersOptions hosts.DialersOptions) error {
	clusterState, err := core.ReadStateFile(ctx, pki.StateFileName)
	if err != nil {
		return err
	}

	kubeCluster, err := core.InitClusterObject(ctx, clusterState.DesiredState.ZKEConfig.DeepCopy())
	if err != nil {
		return err
	}

	log.Infof(ctx, "Building Kubernetes cluster")

	err = kubeCluster.SetupDialers(ctx, dialersOptions)
	if err != nil {
		return err
	}

	err = kubeCluster.TunnelHosts(ctx)
	if err != nil {
		return err
	}

	currentCluster, err := kubeCluster.GetClusterState(ctx, clusterState)
	if err != nil {
		return err
	}

	isNewCluster := true
	if currentCluster != nil {
		isNewCluster = false
	}

	if !kubeCluster.Option.DisablePortCheck {
		if err = kubeCluster.CheckClusterPorts(ctx, currentCluster); err != nil {
			return err
		}
	}
	core.SetUpAuthentication(ctx, kubeCluster, currentCluster, clusterState)

	kubeConfig, err := config.BuildConfig([]byte(kubeCluster.Certificates[pki.KubeAdminCertName].Config))
	if err != nil {
		return err
	}

	kubeClientSet, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}
	kubeCluster.KubeClient = kubeClientSet

	err = kubeCluster.SetUpHosts(ctx)
	if err != nil {
		return err
	}

	err = core.ReconcileCluster(ctx, kubeCluster, currentCluster)
	if err != nil {
		return err
	}

	if err := kubeCluster.PrePullK8sImages(ctx); err != nil {
		return err
	}

	err = kubeCluster.DeployControlPlane(ctx)
	if err != nil {
		return err
	}

	err = core.ApplyAuthzResources(ctx, kubeCluster.ZKEConfig, kubeCluster.KubeClient, dialersOptions)
	if err != nil {
		return err
	}

	err = kubeCluster.UpdateClusterCurrentState(ctx, clusterState)
	if err != nil {
		return err
	}

	err = core.SaveZKEConfigToKubernetes(ctx, kubeCluster, clusterState)
	if err != nil {
		return err
	}

	err = kubeCluster.DeployWorkerPlane(ctx)
	if err != nil {
		return err
	}

	err = kubeCluster.CleanDeadLogs(ctx)
	if err != nil {
		return err
	}

	err = kubeCluster.SyncLabelsAndTaints(ctx, currentCluster)
	if err != nil {
		return err
	}

	err = ConfigureCluster(ctx, kubeCluster.ZKEConfig, kubeCluster.Certificates, dialersOptions, isNewCluster)
	if err != nil {
		return err
	}

	err = checkAllIncluded(kubeCluster)
	if err != nil {
		return err
	}

	if err = pki.DeployAdminConfig(ctx, kubeCluster.Certificates[pki.KubeAdminCertName].Config, pki.KubeAdminConfigName); err != nil {
		return err
	}

	log.Infof(ctx, "Finished building Kubernetes cluster successfully")
	return nil
}

func ClusterUpForSingleCloud(ctx context.Context, clusterState *core.FullState, dialersOptions hosts.DialersOptions) (*core.FullState, error) {
	kubeCluster, err := core.InitClusterObject(ctx, clusterState.DesiredState.ZKEConfig.DeepCopy())
	if err != nil {
		return clusterState, err
	}

	log.Infof(ctx, "Building Kubernetes cluster")

	err = kubeCluster.SetupDialers(ctx, dialersOptions)
	if err != nil {
		return clusterState, err
	}

	err = kubeCluster.TunnelHosts(ctx)
	if err != nil {
		return clusterState, err
	}

	currentCluster, err := kubeCluster.GetClusterState(ctx, clusterState)
	if err != nil {
		return clusterState, err
	}

	isNewCluster := true
	if currentCluster != nil {
		isNewCluster = false
	}

	if !kubeCluster.Option.DisablePortCheck {
		if err = kubeCluster.CheckClusterPorts(ctx, currentCluster); err != nil {
			return clusterState, err
		}
	}

	core.SetUpAuthentication(ctx, kubeCluster, currentCluster, clusterState)

	kubeConfig, err := config.BuildConfig([]byte(kubeCluster.Certificates[pki.KubeAdminCertName].Config))
	if err != nil {
		return clusterState, err
	}

	kubeClientSet, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return clusterState, err
	}
	kubeCluster.KubeClient = kubeClientSet

	err = kubeCluster.SetUpHosts(ctx)
	if err != nil {
		return clusterState, err
	}

	err = core.ReconcileCluster(ctx, kubeCluster, currentCluster)
	if err != nil {
		return clusterState, err
	}

	if err := kubeCluster.PrePullK8sImages(ctx); err != nil {
		return clusterState, err
	}

	err = kubeCluster.DeployControlPlane(ctx)
	if err != nil {
		return clusterState, err
	}

	err = core.ApplyAuthzResources(ctx, kubeCluster.ZKEConfig, kubeCluster.KubeClient, dialersOptions)
	if err != nil {
		return clusterState, err
	}

	err = kubeCluster.DeployWorkerPlane(ctx)
	if err != nil {
		return clusterState, err
	}

	err = kubeCluster.CleanDeadLogs(ctx)
	if err != nil {
		return clusterState, err
	}

	err = kubeCluster.SyncLabelsAndTaints(ctx, currentCluster)
	if err != nil {
		return clusterState, err
	}

	err = ConfigureCluster(ctx, kubeCluster.ZKEConfig, kubeCluster.Certificates, dialersOptions, isNewCluster)
	if err != nil {
		return clusterState, err
	}

	clusterState, err = kubeCluster.UpdateClusterCurrentStateForSingleCloud(ctx, clusterState)
	if err != nil {
		return clusterState, err
	}

	err = core.SaveZKEConfigToKubernetes(ctx, kubeCluster, clusterState)
	if err != nil {
		return clusterState, err
	}

	err = checkAllIncluded(kubeCluster)
	if err != nil {
		return clusterState, err
	}

	if !isNewCluster {
		log.Infof(ctx, "Begin clean to delete hosts")
		if err := postCleanToDeleteHosts(ctx, kubeCluster, currentCluster); err != nil {
			return clusterState, err
		}
	}
	log.Infof(ctx, "Finished building Kubernetes cluster successfully")
	return clusterState, nil
}

func checkAllIncluded(cluster *core.Cluster) error {
	if len(cluster.InactiveHosts) == 0 {
		return nil
	}
	var names []string
	for _, host := range cluster.InactiveHosts {
		names = append(names, host.Address)
	}
	return fmt.Errorf("Provisioning incomplete, host(s) [%s] skipped because they could not be contacted", strings.Join(names, ","))
}

func clusterUpFromCli(cliCtx *cli.Context) error {
	parentCtx := context.Background()
	logger := cementlog.NewLog4jConsoleLogger(log.LogLevel)
	defer logger.Close()
	ctx, err := log.SetLogger(parentCtx, logger)
	if err != nil {
		return err
	}
	startUPtime := time.Now()

	clusterFile, err := resolveClusterFile(pki.ClusterConfig)
	if err != nil {
		return fmt.Errorf("Failed to resolve cluster file: %v", err)
	}
	zkeConfig, err := core.ParseConfig(ctx, clusterFile)
	if err != nil {
		return fmt.Errorf("Failed to parse cluster file: %v", err)
	}
	err = validateConfigVersion(zkeConfig)
	if err != nil {
		return err
	}

	err = ClusterInit(ctx, zkeConfig, hosts.DialersOptions{})
	if err != nil {
		return err
	}

	err = ClusterUp(ctx, hosts.DialersOptions{})
	if err == nil {
		endUPtime := time.Since(startUPtime) / 1e9
		log.Infof(ctx, fmt.Sprintf("This up takes [%s] secends", strconv.FormatInt(int64(endUPtime), 10)))
	}
	return err
}

func ClusterUpFromSingleCloud(scCtx context.Context, zkeConfig *types.ZKEConfig, clusterState *core.FullState, logger cementlog.Logger) (*core.FullState, error) {
	startUPtime := time.Now()
	ctx, err := log.SetLogger(scCtx, logger)
	if err != nil {
		return clusterState, err
	}

	newClusterState, err := ClusterInitForSingleCloud(ctx, zkeConfig, clusterState, hosts.DialersOptions{})
	if err != nil {
		return clusterState, err
	}

	newClusterState, err = ClusterUpForSingleCloud(ctx, newClusterState, hosts.DialersOptions{})

	if err == nil {
		endUPtime := time.Since(startUPtime) / 1e9
		log.Infof(ctx, fmt.Sprintf("This up takes [%s] secends", strconv.FormatInt(int64(endUPtime), 10)))
	}
	return newClusterState, err
}

func ConfigureCluster(
	ctx context.Context,
	zkeConfig types.ZKEConfig,
	crtBundle map[string]pki.CertificatePKI,
	dailersOptions hosts.DialersOptions,
	isNewCluster bool) error {
	// dialer factories are not needed here since we are not uses docker only k8s jobs
	kubeCluster, err := core.InitClusterObject(ctx, &zkeConfig)
	if err != nil {
		return err
	}
	if err := kubeCluster.SetupDialers(ctx, dailersOptions); err != nil {
		return err
	}
	if len(kubeCluster.ControlPlaneHosts) > 0 && isNewCluster {
		kubeCluster.Certificates = crtBundle
		if err := network.DeployNetwork(ctx, kubeCluster); err != nil {
			return err
			log.Warnf(ctx, "Failed to deploy [%s]: %v", network.NetworkPluginResourceName, err)
		}

		if err := monitor.DeployMonitoring(ctx, kubeCluster); err != nil {
			return err
		}

		if err := zcloud.DeployZcloudManager(ctx, kubeCluster); err != nil {
			return err
		}
	}
	return nil
}

func postCleanToDeleteHosts(ctx context.Context, kubeCluster, currentCluster *core.Cluster) error {
	currentAllHosts := hosts.GetUniqueHostList(currentCluster.ControlPlaneHosts, currentCluster.EtcdHosts, currentCluster.WorkerHosts, currentCluster.EdgeHosts)
	configAllHosts := hosts.GetUniqueHostList(kubeCluster.ControlPlaneHosts, kubeCluster.EtcdHosts, kubeCluster.WorkerHosts, kubeCluster.EdgeHosts)
	toDeleteHosts := hosts.GetToDeleteHosts(currentAllHosts, configAllHosts, []*hosts.Host{})

	_, err := errgroup.Batch(toDeleteHosts, func(h interface{}) (interface{}, error) {
		runHost := h.(*hosts.Host)
		return nil, runHost.CleanUpAll(ctx, kubeCluster.Image.Alpine, kubeCluster.PrivateRegistriesMap, false, currentCluster.Option.ClusterCidr)
	})

	if err != nil {
		return err
	}
	return nil
}
