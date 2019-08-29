package monitor

import (
	"context"
	b64 "encoding/base64"

	"github.com/zdnscloud/zke/core"
	"github.com/zdnscloud/zke/core/pki"
	"github.com/zdnscloud/zke/monitor/prometheus"
	"github.com/zdnscloud/zke/pkg/k8s"
	"github.com/zdnscloud/zke/pkg/log"
	"github.com/zdnscloud/zke/pkg/util"

	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/helper"
)

const DeployNamespace = "zcloud"

func DeployMonitoring(ctx context.Context, c *core.Cluster) error {
	select {
	case <-ctx.Done():
		return util.CancelErr
	default:
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

		err = depolyPrometheusCrd(k8sClient, c)
		if err != nil {
			log.Infof(ctx, "[Monitor] deploy prometheus crd failed")
			return err
		}

		log.Infof(ctx, "[Monitor] Successfully deployed Monitor Plugin")
		return nil
	}
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

func depolyPrometheusCrd(cli client.Client, c *core.Cluster) error {
	if err := deployCrdFromBase64(cli, prometheus.AlertManagerCrdB64); err != nil {
		return err
	}

	if err := deployCrdFromBase64(cli, prometheus.PodMonitorCrdB64); err != nil {
		return err
	}

	if err := deployCrdFromBase64(cli, prometheus.PrometheusCrdB64); err != nil {
		return err
	}

	if err := deployCrdFromBase64(cli, prometheus.PrometheusRuleCrdB64); err != nil {
		return err
	}

	return deployCrdFromBase64(cli, prometheus.ServiceMonitorCrdB64)
}

func deployCrdFromBase64(cli client.Client, b64String string) error {
	yamlBytes, err := b64.StdEncoding.DecodeString(b64String)
	if err != nil {
		return err
	}
	return helper.CreateResourceFromYaml(cli, string(yamlBytes))
}
