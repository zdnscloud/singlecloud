package k8s

import (
	"github.com/zdnscloud/zke/pkg/templates"

	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/client/config"
	"github.com/zdnscloud/gok8s/helper"
)

func DoCreateFromTemplate(cli client.Client, template string, templateConfig interface{}) error {
	yaml, err := templates.CompileTemplateFromMap(template, templateConfig)
	if err != nil {
		return err
	}
	return doCreateFromYaml(cli, yaml)
}

func doCreateFromYaml(cli client.Client, yaml string) error {
	return helper.CreateResourceFromYaml(cli, yaml)
}

func DoUpdateFromTemplate(cli client.Client, template string, templateConfig interface{}) error {
	yaml, err := templates.CompileTemplateFromMap(template, templateConfig)
	if err != nil {
		return err
	}
	return doUpdateFromYaml(cli, yaml)
}

func doUpdateFromYaml(cli client.Client, yaml string) error {
	return helper.UpdateResourceFromYaml(cli, yaml)
}

func GetK8sClientFromConfig(kubeConfigPath string) (client.Client, error) {
	cfg, err := config.GetConfigFromFile(kubeConfigPath)
	if err != nil {
		return nil, err
	}
	return client.New(cfg, client.Options{})
}

func GetK8sClientFromYaml(kubeConfig string) (client.Client, error) {
	cfg, err := config.BuildConfig([]byte(kubeConfig))
	if err != nil {
		return nil, err
	}
	return client.New(cfg, client.Options{})
}
