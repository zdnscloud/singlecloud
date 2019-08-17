package charts

type Prometheus struct {
	Grafana      PrometheusGrafana      `json:"grafana"`
	Prometheus   PrometheusPrometheus   `json:"prometheus"`
	AlertManager PrometheusAlertManager `json:"alertmanager"`
}

type PrometheusGrafana struct {
	Ingress       PrometheusGrafanaIngress `json:"ingress"`
	AdminPassword string                   `json:"adminPassword"`
}

type PrometheusGrafanaIngress struct {
	Hosts []string `json:"hosts"`
}

type PrometheusPrometheus struct {
	PrometheusSpec PrometheusSpec `json:"prometheusSpec"`
}

type PrometheusSpec struct {
	Retention      string `json:"retention"`
	ScrapeInterval string `json:"scrapeInterval"`
	StorageClass   string `json:"storageClass"`
	StorageSize    string `json:"storageSize"`
}

type PrometheusAlertManager struct {
	AlertManagerSpec AlertManagerSpec `json:"alertmanagerSpec"`
}

type AlertManagerSpec struct {
	StorageClass string `json:"storageClass"`
	StorageSize  string `json:"storageSize"`
}
