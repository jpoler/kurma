package grpc

import (
	"net"

	"github.com/apcera/kurma/metadatasvc/backend"
	"github.com/apcera/kurma/metadatasvc/protocol"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// GrpcServer ...
type Server interface {
	Listen() error
}

type server struct {
	server *grpc.Server
	store  backend.Backend
}

// NewGrpcServer ...
func NewGrpcServer(store backend.Backend) Server {
	kms := &server{store: store, server: grpc.NewServer()}
	protocol.RegisterKurmaMetadataServer(kms.server, kms)
	return kms
}

func (gs *server) Listen() error {
	lis, err := net.Listen("unix", "socket")
	if err != nil {
		return err
	}

	return gs.server.Serve(lis)
}

func (gs *server) RegisterPod(c context.Context, pd *protocol.PodDefinition) (*protocol.RegisterResponse, error) {
	token, err := gs.store.RegisterPod(pd.GetID().UUID, pd.PodManifest, pd.HMACKey)
	if err != nil {
		return nil, err
	}
	return &protocol.RegisterResponse{
		URL: token, // TODO: Fix this.  Backend needs to give me a token
	}, nil
}

func (gs *server) UnregisterPod(c context.Context, podID *protocol.PodID) (*protocol.UnregisterResponse, error) {
	return nil, nil
}
