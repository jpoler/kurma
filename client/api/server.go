// Copyright 2015 Apcera Inc. All rights reserved.

package api

import (
	"net"

	pb "github.com/apcera/kurma/stage1/client"
	"github.com/apcera/logray"
	"google.golang.org/grpc"
)

// Options devices the configuration fields that can be passed to New() when
// instantiating a new api.Server.
type Options struct {
	BindAddress string
}

// Server represents the process that acts as a daemon to receive container
// management requests.
type Server struct {
	log     *logray.Logger
	options *Options
}

// New creates and returns a new Server object with the provided Options as
// configuration.
func New(options *Options) *Server {
	if options.BindAddress == "" {
		options.BindAddress = ":12312"
	}

	s := &Server{
		log:     logray.New(),
		options: options,
	}
	return s
}

// Start begins the server. It will return an error if starting the Server
// fails, or return nil on success.
func (s *Server) Start() error {
	l, err := net.Listen("tcp", s.options.BindAddress)
	if err != nil {
		return err
	}
	defer l.Close()

	// create the client RPC connection to the host
	conn, err := grpc.Dial("127.0.0.1:12311")
	if err != nil {
		return err
	}

	// create the RPC handler
	rpc := &rpcServer{
		log:    s.log.Clone(),
		client: pb.NewKurmaClient(conn),
	}

	// create the gRPC server and run
	gs := grpc.NewServer()
	pb.RegisterKurmaServer(gs, rpc)
	s.log.Debug("Server is ready")
	gs.Serve(l)
	return nil
}
