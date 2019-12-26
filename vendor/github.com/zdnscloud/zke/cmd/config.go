package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"

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
		Name:   "genconfig",
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

func LoadImageConfig(path string) error {
	if path == "" {
		return fmt.Errorf("must specify image config file")
	}
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open image config file %s failed %s", path, err.Error())
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return fmt.Errorf("read image config file failed %s", err.Error())
	}

	if err := yaml.Unmarshal(b, &types.AllK8sVersions); err != nil {
		return fmt.Errorf("unmarshal image config failed %s", err.Error())
	}
	return validateImageConfig(types.AllK8sVersions)
}

func validateImageConfig(in map[string]types.ZKEConfigImages) error {
	if _, ok := in[types.DefaultK8s]; !ok {
		return fmt.Errorf("validate image config failed: defaultK8s %s not in image configs", types.DefaultK8s)
	}

	for version, images := range in {
		t := reflect.TypeOf(images)
		v := reflect.ValueOf(images)
		for i := 0; i < t.NumField(); i++ {
			if v.Field(i).String() == "" {
				return fmt.Errorf("validate image config failed: k8s version %s field %s nil", version, t.Field(i).Name)
			}
		}
	}
	return nil
}
