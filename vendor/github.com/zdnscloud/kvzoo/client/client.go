package client

import (
	"time"

	pb "github.com/zdnscloud/kvzoo/proto"
	"google.golang.org/grpc"
)

type Client struct {
	pb.KVSClient
	conn *grpc.ClientConn
}

func NewClient(addr string, timeout time.Duration) (*Client, error) {
	dialOptions := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithTimeout(timeout),
	}

	conn, err := grpc.Dial(addr, dialOptions...)
	if err != nil {
		return nil, err
	}

	return &Client{
		KVSClient: pb.NewKVSClient(conn),
		conn:      conn,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Target() string {
	return c.conn.Target()
}
