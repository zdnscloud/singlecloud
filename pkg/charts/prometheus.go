package charts

type Prometheus struct {
	IngressDomain       []string `json:"ingressDomain"`
	PrometheusRetention string   `json:"prometheusRetention"`
	ScrapeInterval      string   `json:"scrapeInterval"`
	AdminPassword       string   `json:"adminPassword"`
}
