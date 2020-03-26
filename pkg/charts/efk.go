package charts

type EFK struct {
	Elasticsearch ES `json:"elasticsearch,omitempty"`
	Kibana        KA `json:"kibana,omitempty"`
}

type ES struct {
	Replicas            int `json:"replicas,omitempty"`
	VolumeClaimTemplate Pvc `json:"volumeClaimTemplate,omitempty"`
}

type Pvc struct {
	StorageClass string       `json:"storageClassName,omitempty"`
	Resources    PvcResources `json:"resources,omitempty"`
}

type PvcResources struct {
	Requests PvcRequests `json:"requests,omitempty"`
}

type PvcRequests struct {
	Storage int `json:"storage,omitempty"`
}

type KA struct {
	Ingress KibanaIngress `json:"ingress,omitempty"`
}

type KibanaIngress struct {
	Hosts string `json:"hosts,omitempty"`
}
