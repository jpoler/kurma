// Copyright 2015 Apcera Inc. All rights reserved.

package server

import (
	"net"

	pb "github.com/apcera/kurma/stage1/client"
	"github.com/apcera/kurma/stage1/container"
	"github.com/apcera/logray"
	"google.golang.org/grpc"
)

// Options devices the configuration fields that can be passed to New() when
// instantiating a new Server.
type Options struct {
	ParentCgroupName   string
	ContainerDirectory string
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
	s := &Server{
		log:     logray.New(),
		options: options,
	}
	return s
}

// Start begins the server. It will return an error if starting the Server
// fails, or return nil on success.
func (s *Server) Start() error {
	l, err := net.Listen("tcp", ":12311")
	if err != nil {
		return err
	}
	defer l.Close()

	// create the RPC handler
	rpc := &rpcServer{
		log:            s.log.Clone(),
		pendingUploads: make(map[string]*pendingContainer),
	}

	// initialize the container manager
	rpc.manager, err = s.initializeManager()
	if err != nil {
		return err
	}

	// create the gRPC server and run
	gs := grpc.NewServer()
	pb.RegisterKurmaServer(gs, rpc)
	s.log.Info("Server is ready")
	gs.Serve(l)
	return nil
}

// initializeManager creates the stage0 manager object which will handle
// container launching.
func (s *Server) initializeManager() (*container.Manager, error) {
	mopts := &container.Options{
		ParentCgroupName:   s.options.ParentCgroupName,
		ContainerDirectory: s.options.ContainerDirectory,
	}

	m, err := container.NewManager(mopts)
	if err != nil {
		return nil, err
	}
	m.Log = s.log.Clone()
	return m, nil
}
