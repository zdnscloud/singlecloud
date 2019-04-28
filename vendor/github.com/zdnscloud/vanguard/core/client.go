package core

import (
	"net"
	"time"

	"github.com/zdnscloud/g53"
)

type Client struct {
	Addr        net.Addr
	DestAddr    net.Addr
	UsingTCP    bool
	Request     *g53.Message
	Response    *g53.Message
	View        string
	ViewId      uint16
	CacheHit    bool
	CacheAnswer bool
	CreateTime  time.Time
}

func (c *Client) QueryKey() uint64 {
	question := c.Request.Question
	return (uint64(question.Name.Hash(false)) << 32) |
		(uint64(c.ViewId) << 16) |
		uint64(question.Type)
}

func (c *Client) reset() {
	c.Addr = nil
	c.DestAddr = nil
	c.Request = nil
	c.Response = nil
	c.View = "default"
	c.ViewId = 0
	c.CacheHit = false
	c.CacheAnswer = true
	c.CreateTime = time.Now()
}

func (c *Client) clone(other *Client) *Client {
	c.Addr = other.Addr
	c.DestAddr = other.DestAddr
	c.Request = other.Request
	c.Response = other.Response
	c.View = other.View
	c.ViewId = other.ViewId
	c.CacheHit = other.CacheHit
	c.CacheAnswer = other.CacheAnswer
	c.CreateTime = other.CreateTime
	return c
}

func (c *Client) IP() net.IP {
	if c.UsingTCP {
		return c.Addr.(*net.TCPAddr).IP
	} else {
		return c.Addr.(*net.UDPAddr).IP
	}
}

func (c *Client) DestIP() net.IP {
	if c.UsingTCP {
		return c.DestAddr.(*net.TCPAddr).IP
	} else {
		return c.DestAddr.(*net.UDPAddr).IP
	}
}

func (c *Client) Port() int {
	if c.UsingTCP {
		return c.Addr.(*net.TCPAddr).Port
	} else {
		return c.Addr.(*net.UDPAddr).Port
	}
}
