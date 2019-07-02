package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/zdnscloud/zke/core"
	"github.com/zdnscloud/zke/core/pki"
	"github.com/zdnscloud/zke/pkg/hosts"
	"github.com/zdnscloud/zke/pkg/log"
	"github.com/zdnscloud/zke/types"
)

func resolveClusterFile(clusterFile string) (string, error) {
	fp, err := filepath.Abs(clusterFile)
	if err != nil {
		return "", fmt.Errorf("failed to lookup current directory name: %v", err)
	}
	file, err := os.Open(fp)
	if err != nil {
		return "", fmt.Errorf("can not find cluster configuration file: %v", err)
	}
	defer file.Close()
	buf, err := ioutil.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}
	clusterFileBuff := string(buf)
	return clusterFileBuff, nil
}

func ClusterInit(ctx context.Context, zkeConfig *types.ZKEConfig, dialersOptions hosts.DialersOptions) error {
	log.Infof(ctx, "Initiating Kubernetes cluster")
	var fullState *core.FullState
	stateFilePath := pki.StateFileName
	zkeFullState, _ := core.ReadStateFile(ctx, stateFilePath)
	kubeCluster, err := core.InitClusterObject(ctx, zkeConfig)
	if err != nil {
		return err
	}
	if err := kubeCluster.SetupDialers(ctx, dialersOptions); err != nil {
		return err
	}
	err = doUpgradeLegacyCluster(ctx, kubeCluster, zkeFullState)
	if err != nil {
		log.Warnf(ctx, "[state] can't fetch legacy cluster state from Kubernetes")
	}
	fullState, err = core.RebuildState(ctx, &kubeCluster.ZKEConfig, zkeFullState)
	if err != nil {
		return err
	}
	zkeState := core.FullState{
		DesiredState: fullState.DesiredState,
		CurrentState: fullState.CurrentState,
	}
	return zkeState.WriteStateFile(ctx, stateFilePath)
}

func ClusterInitForRest(ctx context.Context, zkeConfig *types.ZKEConfig, zkeFullState *core.FullState, dialersOptions hosts.DialersOptions) (*core.FullState, error) {
	log.Infof(ctx, "Initiating Kubernetes cluster")
	var fullState *core.FullState
	kubeCluster, err := core.InitClusterObject(ctx, zkeConfig)
	if err != nil {
		return zkeFullState, err
	}

	if err := kubeCluster.SetupDialers(ctx, dialersOptions); err != nil {
		return zkeFullState, err
	}

	if err := core.RebuildKubeconfigForRest(ctx, kubeCluster); err != nil {
		return zkeFullState, err
	}

	fullState, err = core.RebuildState(ctx, &kubeCluster.ZKEConfig, zkeFullState)
	if err != nil {
		return fullState, err
	}
	zkeState := core.FullState{
		DesiredState: fullState.DesiredState,
		CurrentState: fullState.CurrentState,
	}
	return &zkeState, nil
}
