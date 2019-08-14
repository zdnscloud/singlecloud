package charts

type Prometheus struct {
	IngressDomain string `json:"ingressDomain"`
	StorageClass  string `json:"storageClass"`
	StorageSize   string `json:"storageSize"`
	AdminPassword string `json:"adminPassword"`
}
