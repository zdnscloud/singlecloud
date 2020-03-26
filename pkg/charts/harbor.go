package charts

type Harbor struct {
	Ingress       HarborIngress     `json:"ingress"`
	Persistence   HarborPersistence `json:"persistence,omitempty"`
	AdminPassword string            `json:"harborAdminPassword,omitempty"`
	ExternalURL   string            `json:"externalURL"`
}

type HarborPersistence struct {
	StorageClass string `json:"storageClass,omitempty"`
	StorageSize  int    `json:"registryStorageSize,omitempty"`
}

type HarborIngress struct {
	Core   string `json:"core"`
	CaCrt  string `json:"caCrt"`
	TlsCrt string `json:"tlsCrt"`
	TlsKey string `json:"tlsKey"`
}
