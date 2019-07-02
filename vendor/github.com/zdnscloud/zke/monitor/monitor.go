package monitor

import (
	"context"

	"github.com/zdnscloud/zke/core"
	"github.com/zdnscloud/zke/core/pki"
	"github.com/zdnscloud/zke/pkg/k8s"
	"github.com/zdnscloud/zke/pkg/log"

	"github.com/zdnscloud/gok8s/client"
)

const DeployNamespace = "zcloud"

func DeployMonitoring(ctx context.Context, c *core.Cluster) error {
	log.Infof(ctx, "[Monitor] Setting up Monitor Plugin")
	templateConfig, k8sClient, err := prepare(c)
	if err != nil {
		return err
	}

	err = k8s.DoCreateFromTemplate(k8sClient, metricsServerTemplate, templateConfig)
	if err != nil {
		log.Infof(ctx, "[Monitor] deploy metrics server failed")
		return err
	}

	log.Infof(ctx, "[Monitor] Successfully deployed Monitor Plugin")
	return nil
}

func prepare(c *core.Cluster) (map[string]interface{}, client.Client, error) {
	templateConfig := map[string]interface{}{
		"MetricsServerImage":        c.Image.MetricsServer,
		"RBACConfig":                c.Authorization.Mode,
		"MetricsServerOptions":      c.Monitor.MetricsOptions,
		"MetricsServerMajorVersion": "v0.3",
		"DeployNamespace":           DeployNamespace,
	}
	k8sClient, err := k8s.GetK8sClientFromYaml(c.Certificates[pki.KubeAdminCertName].Config)
	return templateConfig, k8sClient, err
}
