package charts

type Harbor struct {
	IngressDomain       string `json:"ingressDomain"`
	StorageClass        string `json:"storageClass"`
	RegistryStorageSize string `json:"registryStorageSize"`
	AdminPassword       string `json:"adminPassword"`
}
