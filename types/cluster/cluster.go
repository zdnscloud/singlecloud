package cluster

type Cluster struct {
	ID                string `json:"id,omitempty"`
	Type              string `json:"type,omitempty"`
	Name              string `json:"name,omitempty"`
	NodesCount        uint32 `json:"nodes_count,omitempty"`
	Version           string `json:"version,omitempty"`
	CreationTimestamp string `json:"create_timestamp,omitempty"`
}

func (c *Cluster) GetID() string {
	return c.ID
}

func (c *Cluster) SetID(id string) {
	c.ID = id
}

func (c *Cluster) GetType() string {
	return c.Type
}

func (c *Cluster) SetType(typ string) {
	c.Type = typ
}
