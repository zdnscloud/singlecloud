package server

import (
	"net"

	"github.com/zdnscloud/kvzoo"
	"github.com/zdnscloud/kvzoo/backend/bolt"
	pb "github.com/zdnscloud/kvzoo/proto"
	"google.golang.org/grpc"
)

type KVGRPCServer struct {
	service  *KVService
	server   *grpc.Server
	listener net.Listener
}

func NewWithBoltDB(addr string, dbFilePath string) (*KVGRPCServer, error) {
	db, err := bolt.New(dbFilePath)
	if err != nil {
		return nil, err
	}

	if s, err := New(addr, db); err == nil {
		return s, err
	} else {
		db.Destroy()
		return nil, err
	}
}

func New(addr string, db kvzoo.DB) (*KVGRPCServer, error) {
	server := grpc.NewServer()

	service := newKVService(db)
	pb.RegisterKVSServer(server, service)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &KVGRPCServer{
		service:  service,
		server:   server,
		listener: listener,
	}, nil
}

func (s *KVGRPCServer) Start() error {
	return s.server.Serve(s.listener)
}

func (s *KVGRPCServer) Stop() error {
	s.server.GracefulStop()
	s.service.Close()
	return nil
}
