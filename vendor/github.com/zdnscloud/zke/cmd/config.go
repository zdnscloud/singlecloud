package cmd

import (
	"context"
	"io/ioutil"

	"github.com/zdnscloud/zke/core"
	"github.com/zdnscloud/zke/core/pki"
	"github.com/zdnscloud/zke/pkg/log"
	"github.com/zdnscloud/zke/types"

	"github.com/urfave/cli"
	cementlog "github.com/zdnscloud/cement/log"
	"gopkg.in/yaml.v2"
)

func ConfigCommand() cli.Command {
	return cli.Command{
		Name:   "generateconfig",
		Usage:  "Generate an empty configuration file",
		Action: generateConfig,
	}
}

func writeConfig(ctx context.Context, cluster *types.ZKEConfig, configFile string) error {
	yamlConfig, err := yaml.Marshal(*cluster)
	if err != nil {
		return err
	}
	log.Debugf(ctx, "Deploying cluster configuration file: %s", configFile)
	return ioutil.WriteFile(configFile, yamlConfig, 0640)
}

func generateConfig(cliCtx *cli.Context) error {
	parentCtx := context.Background()
	logger := cementlog.NewLog4jConsoleLogger(log.LogLevel)
	defer logger.Close()
	ctx, err := log.SetLogger(parentCtx, logger)
	if err != nil {
		return err
	}

	cluster := types.ZKEConfig{}
	cluster.ConfigVersion = core.DefaultConfigVersion
	cluster.Nodes = make([]types.ZKEConfigNode, 1)
	return writeConfig(ctx, &cluster, pki.ClusterConfig)
}
