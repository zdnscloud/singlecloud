package charts

type Harbor struct {
	Ingress       HarborIngress     `json:"ingress"`
	Persistence   HarborPersistence `json:"persistence"`
	AdminPassword string            `json:"harborAdminPassword"`
	ExternalURL   string            `json:"externalURL"`
}

type HarborPersistence struct {
	StorageClass string `json:"storageClass"`
	StorageSize  int    `json:"registryStorageSize"`
}

type HarborIngress struct {
	Core   string `json:"core"`
	CaCrt  string `json:"caCrt"`
	TlsCrt string `json:"tlsCrt"`
	TlsKey string `json:"tlsKey"`
}
