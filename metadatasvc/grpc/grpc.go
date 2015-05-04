package grpc

import (
	"net"

	backend "github.com/apcera/kurma/metadatasvc/backend"
	"github.com/apcera/kurma/metadatasvc/protocol"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// GrpcServer ...
type GrpcServer interface {
	Listen() error
}

type grpcServer struct {
	server *grpc.Server
	store  backend.Backend
}

// NewGrpcServer ...
func NewGrpcServer(store backend.Backend) GrpcServer {
	kms := &grpcServer{store: store, server: grpc.NewServer()}
	protocol.RegisterKurmaMetadataServer(kms.server, kms)
	return kms
}

func (gs *grpcServer) Listen() error {
	lis, err := net.Listen("unix", "socket")
	if err != nil {
		return err
	}

	return gs.server.Serve(lis)
}

func (gs *grpcServer) RegisterPod(context.Context, *protocol.PodDefinition) (*protocol.Response, error) {
	return nil, nil
}

func (gs *grpcServer) UnregisterPod(context.Context, *protocol.PodID) (*protocol.Response, error) {
	return nil, nil
}
