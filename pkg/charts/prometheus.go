package charts

type Prometheus struct {
	Grafana      PrometheusGrafana      `json:"grafana"`
	Prometheus   PrometheusPrometheus   `json:"prometheus"`
	AlertManager PrometheusAlertManager `json:"alertmanager"`
	KubeEtcd     PrometheusEtcd         `json:"kubeEtcd"`
}

type PrometheusGrafana struct {
	Ingress       PrometheusGrafanaIngress `json:"ingress"`
	AdminPassword string                   `json:"adminPassword"`
}

type PrometheusGrafanaIngress struct {
	Hosts string `json:"hosts"`
}

type PrometheusPrometheus struct {
	PrometheusSpec PrometheusSpec `json:"prometheusSpec"`
}

type PrometheusSpec struct {
	Retention      int    `json:"retention"`
	ScrapeInterval int    `json:"scrapeInterval"`
	StorageClass   string `json:"storageClass"`
	StorageSize    int    `json:"storageSize"`
}

type PrometheusAlertManager struct {
	AlertManagerSpec AlertManagerSpec `json:"alertmanagerSpec"`
}

type AlertManagerSpec struct {
	StorageClass string `json:"storageClass"`
}

type PrometheusEtcd struct {
	Enabled   bool     `json:"enabled"`
	EndPoints []string `json:"endpoints"`
}
