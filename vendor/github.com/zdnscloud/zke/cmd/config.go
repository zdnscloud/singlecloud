package cmd

import (
	"io/ioutil"

	"github.com/zdnscloud/zke/core/pki"
	"github.com/zdnscloud/zke/types"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
)

func ConfigCommand() cli.Command {
	return cli.Command{
		Name:   "generateconfig",
		Usage:  "Generate an empty configuration file",
		Action: generateConfig,
	}
}

func writeConfig(cluster *types.ZKEConfig, configFile string) error {
	yamlConfig, err := yaml.Marshal(*cluster)
	if err != nil {
		return err
	}
	logrus.Debugf("Deploying cluster configuration file: %s", configFile)
	return ioutil.WriteFile(configFile, yamlConfig, 0640)
}

func generateConfig(ctx *cli.Context) error {
	cluster := types.ZKEConfig{}
	cluster.ConfigVersion = defaultConfigVersion
	cluster.Nodes = make([]types.ZKEConfigNode, 1)
	return writeConfig(&cluster, pki.ClusterConfig)
}
