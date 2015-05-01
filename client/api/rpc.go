// Copyright 2015 Apcera Inc. All rights reserved.

package api

import (
	"encoding/json"
	"fmt"
	"io"

	pb "github.com/apcera/kurma/stage1/client"
	"github.com/apcera/logray"
	"github.com/appc/spec/schema"
	"golang.org/x/net/context"
)

type rpcServer struct {
	log    *logray.Logger
	client pb.KurmaClient
}

func (s *rpcServer) Create(ctx context.Context, in *pb.CreateRequest) (*pb.CreateResponse, error) {
	s.log.Debug("Received Create request.")

	// unmarshal the image manifest, ensure its valid
	var imageManifest *schema.ImageManifest
	if err := json.Unmarshal(in.Manifest, &imageManifest); err != nil {
		return nil, fmt.Errorf("invalid image manifest: %v", err)
	}

	// locally validate the manifest to gate remote vs local container functionality
	if err := validateImageManifest(imageManifest); err != nil {
		return nil, fmt.Errorf("image manifest is not valid: %v", err)
	}

	// send the request to the backend
	return s.client.Create(ctx, in)
}

func (s *rpcServer) UploadImage(inStream pb.Kurma_UploadImageServer) error {
	s.log.Debug("Received upload request")

	// NOTE: Unlike in the local daemon, we don't need to revalidate the image
	// manifest between Create and Upload. The backend validates in the Manager
	// because within the backend, there are multiple inputs to Manager.Create
	// (ie, bootstrapping vs the local API).
	//
	// Over the local API, it will cache the image manifest from the Create call
	// and re-use it on the UploadImage call, not pulling it from the binary
	// image.

	packet, err := inStream.Recv()
	if err != nil {
		return err
	}

	outStream, err := s.client.UploadImage(inStream.Context())
	if err != nil {
		return err
	}

	r := pb.NewByteStreamReader(inStream, packet)
	w := pb.NewByteStreamWriter(outStream, packet.StreamId)

	if _, err := io.Copy(w, r); err != nil {
		return fmt.Errorf("write error: %v", err)
	}
	if _, err := outStream.CloseAndRecv(); err != nil {
		return err
	}
	if err := inStream.SendAndClose(&pb.None{}); err != nil {
		return err
	}
	return nil
}

func (s *rpcServer) Destroy(ctx context.Context, in *pb.ContainerRequest) (*pb.None, error) {
	s.log.Debugf("Received container destroy request for %s", in.Uuid)
	return s.client.Destroy(ctx, in)
}

func (s *rpcServer) List(ctx context.Context, in *pb.None) (*pb.ListResponse, error) {
	s.log.Debug("Received container list request")
	return s.client.List(ctx, in)
}

func (s *rpcServer) Get(ctx context.Context, in *pb.ContainerRequest) (*pb.Container, error) {
	s.log.Debug("Received container get request for %s", in.Uuid)
	return s.client.Get(ctx, in)
}
