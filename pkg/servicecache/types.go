package resourcerepo

type Service struct {
	Name string `json:"name"`
	Self string `json:"self"`

	Ingress   *Ingress    `json:"ingress,omitempty"`
	Workloads []*Workload `json:"workloads"`
}

type Pod struct {
	Name    string `json:"name"`
	Self    string `json:"self"`
	IsReady bool   `json:"isReady"`
}

type Ingress struct {
	Name string `json:"name"`
	Self string `json:"self"`
}

type Workload struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Self string `json:"self"`
	Pods []*Pod `json:"pods"`
}
