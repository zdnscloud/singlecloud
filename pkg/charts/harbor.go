package charts

type Harbor struct {
	IngressDomain       string `json:"ingressDomain"`
	StorageClass        string `json:"storageClass"`
	RegistryStorageSize string `json:"registryStorageSize"`
	AdminPassword       string `json:"adminPassword"`
	CaCert              string `json:"caCert"`
	TlsCert             string `json:"tlsCert"`
	TlsKey              string `json:"tlsKey"`
	ExternalURL         string `json:"externalURL"`
}
