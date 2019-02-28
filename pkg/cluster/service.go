package cluster


type Cluster struct {
	Id string 
	Type string
}


func (c *Cluster) ID() string {
    return c.id
}

func (c *Cluster) Type() string {
    return c.typ
}

func (c *Cluster) SetID(id string) {
    c.id = id
}

func (c *Cluster) SetType(typ string) {
    c.typ = typ
}
