// Copyright 2015 Apcera Inc. All rights reserved.

package server

import (
	"fmt"
	"io"

	pb "github.com/apcera/kurma/stage1/client"
	"github.com/appc/spec/schema/types"
	"github.com/kr/pty"
)

func (s *rpcServer) Enter(stream pb.Kurma_EnterServer) error {
	s.log.Debug("Received enter request")

	// Receive the first chunk so we can get the stream ID, which will be the UUID
	// of the container. The byte portion will be blank, the client always sends a
	// chunk first so the UUID is available immediately.
	chunk, err := stream.Recv()
	if err != nil {
		return err
	}
	cuuid, err := types.NewUUID(chunk.StreamId)
	if err != nil {
		return err
	}

	// get the container
	container := s.manager.Container(*cuuid)
	if container == nil {
		return fmt.Errorf("specified container not found")
	}

	// configure the io.Reader/Writer for the transport
	w := pb.NewByteStreamWriter(stream, chunk.StreamId)
	r := pb.NewByteStreamReader(stream, nil)

	// create a pty, which we'll use for the process entering the container and
	// copy the data back up the transport.
	master, slave, err := pty.Open()
	if err != nil {
		return err
	}
	defer func() {
		slave.Close()
		master.Close()
	}()
	go io.Copy(w, master)
	go io.Copy(master, r)

	// enter into the container
	if err := container.Enter(slave); err != nil {
		return err
	}
	s.log.Debugf("Enter request finished")
	return nil
}
