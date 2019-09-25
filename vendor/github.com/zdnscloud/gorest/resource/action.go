package resource

type Action struct {
	Name  string      `json:"name"`
	Input interface{} `json:"input,omitempty"`
}
