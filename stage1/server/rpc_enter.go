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
	chunk, err := stream.Recv()
	if err != nil {
		return err
	}

	cuuid, err := types.NewUUID(chunk.StreamId)
	if err != nil {
		return err
	}

	container := s.manager.Container(*cuuid)
	if container == nil {
		return fmt.Errorf("specified container not found")
	}

	w := pb.NewByteStreamWriter(stream, chunk.StreamId)
	r := pb.NewByteStreamReader(stream, nil)

	ppp, tty, err := pty.Open()
	if err != nil {
		return err
	}
	defer tty.Close()

	go func() {
		_, err := io.Copy(w, ppp)
		fmt.Printf("stderr done: %v\n", err)
	}()
	go func() {
		_, err := io.Copy(ppp, r)
		fmt.Printf("stdin done: %v\n", err)
	}()
	defer fmt.Printf("all done\n")

	//if err := cmd.Start(); err != nil {
	if err := container.Enter(tty); err != nil {
		ppp.Close()
		return err
	}

	return nil
}
