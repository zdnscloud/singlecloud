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
	Resources PvcResources `json:"resources"`
}

type PvcResources struct {
	Requests PvcRequests `json:"requests"`
}

type PvcRequests struct {
	Storage string `json:"storage"`
}

type KA struct {
	Service Svc           `json:"service"`
	Ingress KibanaIngress `json:"ingress"`
}

type Svc struct {
	Type string `json:"type"`
}

type KibanaIngress struct {
	Ingress string `json:"hosts"`
}
