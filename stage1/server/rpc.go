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
	"github.com/appc/spec/schema/types"
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

	pc := s.pendingUploads[packet.ImageUploadId]
	delete(s.pendingUploads, packet.ImageUploadId)

	r := newImageUploadReader(stream, packet)
	s.log.Debug("Initializing container")
	_, err = s.manager.Create(pc.name, pc.imageManifest, r)
	if err != nil {
		return err
	}
	return nil
}

func (s *rpcServer) Destroy(ctx context.Context, in *pb.ContainerRequest) (*pb.None, error) {
	cuuid, err := types.NewUUID(in.Uuid)
	if err != nil {
		return nil, err
	}

	container := s.manager.Container(*cuuid)
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
	cuuid, err := types.NewUUID(in.Uuid)
	if err != nil {
		return nil, err
	}

	container := s.manager.Container(*cuuid)
	if container == nil {
		return nil, fmt.Errorf("specified container not found")
	}
	return pbContainer(container)
}

func pbContainer(c *container.Container) (*pb.Container, error) {
	manifest := c.Manifest()
	pbc := &pb.Container{
		Uuid: manifest.UUID.String(),
	}

	// marshal the pod manifest
	b, err := manifest.MarshalJSON()
	if err != nil {
		return nil, err
	}
	pbc.Manifest = b

	// map the container state
	switch c.State() {
	case container.NEW:
		pbc.State = pb.Container_NEW
	case container.STARTING:
		pbc.State = pb.Container_STARTING
	case container.RUNNING:
		pbc.State = pb.Container_RUNNING
	case container.STOPPING:
		pbc.State = pb.Container_STOPPING
	case container.STOPPED:
		pbc.State = pb.Container_STOPPED
	case container.EXITED:
		pbc.State = pb.Container_EXITED
	}

	return pbc, nil
}
