package core

type Context struct {
	Client Client
}

func NewContext() *Context {
	return &Context{}
}

func (c *Context) Reset() {
	c.Client.reset()
}

func (c *Context) Clone(client *Client) *Context {
	c.Client = *c.Client.clone(client)
	return c
}
