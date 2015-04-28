// Copyright 2015 Apcera Inc. All rights reserved.

package server

import (
	"encoding/json"
	"fmt"

	pb "github.com/apcera/kurma/stage1/client"
	"github.com/apcera/kurma/stage1/container"
	"github.com/apcera/logray"
	"github.com/apcera/util/uuid"
	"github.com/appc/spec/schema"
	"golang.org/x/net/context"
)

type rpcServer struct {
	log     *logray.Logger
	manager *container.Manager

	pendingUploads map[string]*pendingContainer
}

type pendingContainer struct {
	name          string
	imageManifest *schema.ImageManifest
}

func (s *rpcServer) Create(ctx context.Context, in *pb.CreateRequest) (*pb.CreateResponse, error) {
	s.log.Debug("Received Create request.")

	// unmarshal the image manifest, ensure its valid
	var imageManifest *schema.ImageManifest
	if err := json.Unmarshal(in.Manifest, &imageManifest); err != nil {
		return nil, fmt.Errorf("invalid image manifest: %v", err)
	}

	// validate the manifest with the manager
	if err := s.manager.Validate(imageManifest); err != nil {
		return nil, fmt.Errorf("image manifest is not valid: %v", err)
	}

	// put together the pending container handler
	pc := &pendingContainer{
		name:          in.Name,
		imageManifest: imageManifest,
	}
	resp := &pb.CreateResponse{
		ImageUploadId: uuid.Variant4().String(),
	}
	s.pendingUploads[resp.ImageUploadId] = pc

	s.log.Debug("Finished Create request.")
	return resp, nil
}

func (s *rpcServer) UploadImage(stream pb.Kurma_UploadImageServer) error {
	s.log.Debug("Received upload request")
	packet, err := stream.Recv()
	if err != nil {
		return err
	}

	pc := s.pendingUploads[packet.StreamId]
	delete(s.pendingUploads, packet.StreamId)

	r := pb.NewByteStreamReader(stream, packet)
	s.log.Debug("Initializing container")
	return s.manager.Create(pc.name, pc.imageManifest, r)
}

func (s *rpcServer) Destroy(ctx context.Context, in *pb.ContainerRequest) (*pb.None, error) {
	container := s.manager.Container(in.Uuid)
	if container == nil {
		return nil, fmt.Errorf("specified container not found")
	}
	if err := container.Stop(); err != nil {
		return nil, err
	}

	return &pb.None{}, nil
}

func (s *rpcServer) List(ctx context.Context, in *pb.None) (*pb.ListResponse, error) {
	resp := &pb.ListResponse{
		Containers: make([]*pb.Container, 0),
	}

	for _, container := range s.manager.Containers() {
		c, err := pbContainer(container)
		if err != nil {
			return nil, err
		}
		resp.Containers = append(resp.Containers, c)
	}

	return resp, nil
}

func (s *rpcServer) Get(ctx context.Context, in *pb.ContainerRequest) (*pb.Container, error) {
	container := s.manager.Container(in.Uuid)
	if container == nil {
		return nil, fmt.Errorf("specified container not found")
	}
	return pbContainer(container)
}
