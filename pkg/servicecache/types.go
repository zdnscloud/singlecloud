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
	Name  string        `json:"name"`
	Self  string        `json:"self"`
	Rules []IngressRule `json:"rules"`
}

type IngressRule struct {
	Domain string        `json:"domain"`
	Paths  []IngressPath `json:"path"`
}

type IngressPath struct {
	Service string `json:"service"`
	Path    string `json:"path"`
}

type Workload struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Self string `json:"self"`
	Pods []*Pod `json:"pods"`
}
