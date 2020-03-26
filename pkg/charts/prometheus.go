package charts

type Prometheus struct {
	Grafana      PrometheusGrafana      `json:"grafana,omitempty"`
	Prometheus   PrometheusPrometheus   `json:"prometheus,omitempty"`
	AlertManager PrometheusAlertManager `json:"alertmanager,omitempty"`
	KubeEtcd     PrometheusEtcd         `json:"kubeEtcd,omitempty"`
}

type PrometheusGrafana struct {
	Ingress       PrometheusGrafanaIngress `json:"ingress"`
	AdminPassword string                   `json:"adminPassword,omitempty"`
}

type PrometheusGrafanaIngress struct {
	Hosts string `json:"hosts"`
}

type PrometheusPrometheus struct {
	PrometheusSpec PrometheusSpec `json:"prometheusSpec,omitempty"`
}

type PrometheusSpec struct {
	Retention      int    `json:"retention,omitempty"`
	ScrapeInterval int    `json:"scrapeInterval,omitempty"`
	StorageClass   string `json:"storageClass,omitempty"`
	StorageSize    int    `json:"storageSize,omitempty"`
}

type PrometheusAlertManager struct {
	AlertManagerSpec AlertManagerSpec `json:"alertmanagerSpec"`
}

type AlertManagerSpec struct {
	StorageClass string `json:"storageClass,omitempty"`
}

type PrometheusEtcd struct {
	Enabled   bool     `json:"enabled"`
	EndPoints []string `json:"endpoints"`
}
