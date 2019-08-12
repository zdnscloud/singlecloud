package globaldns

type ZKEConfig struct {
	Option ZKEConfigOption `json:"option,omitempty"`
}

type ZKEConfigOption struct {
	ClusterDomain string `json:"clusterDomain,omitempty"`
}
