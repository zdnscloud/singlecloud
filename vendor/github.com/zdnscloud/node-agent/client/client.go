package client

import (
	"time"

	"google.golang.org/grpc"

	pb "github.com/zdnscloud/node-agent/proto"
)

func NewClient(addr string, timeout time.Duration) (pb.NodeAgentClient, error) {
	dialOptions := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithTimeout(timeout),
	}

	conn, err := grpc.Dial(addr, dialOptions...)
	if err != nil {
		return nil, err
	}

	return pb.NewNodeAgentClient(conn), nil
}
