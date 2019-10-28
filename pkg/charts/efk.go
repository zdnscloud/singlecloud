package charts

type EFK struct {
	Elasticsearch ES `json:"elasticsearch"`
	Kibana        KA `json:"kibana"`
}

type ES struct {
	Replicas            int `json:"replicas"`
	VolumeClaimTemplate Pvc `json:"volumeClaimTemplate"`
}

type Pvc struct {
	StorageClass string       `json:"storageClassName"`
	Resources    PvcResources `json:"resources"`
}

type PvcResources struct {
	Requests PvcRequests `json:"requests"`
}

type PvcRequests struct {
	Storage string `json:"storage"`
}

type KA struct {
	Ingress KibanaIngress `json:"ingress"`
}

type KibanaIngress struct {
	Hosts string `json:"hosts"`
}
