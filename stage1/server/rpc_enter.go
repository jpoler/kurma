// Copyright 2015 Apcera Inc. All rights reserved.

package server

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"

	pb "github.com/apcera/kurma/stage1/client"
	"github.com/kr/pty"
)

func (s *rpcServer) Enter(stream pb.Kurma_EnterServer) error {
	s.log.Debug("Received enter request")
	chunk, err := stream.Recv()
	if err != nil {
		return err
	}

	w := pb.NewByteStreamWriter(stream, chunk.StreamId)
	r := pb.NewByteStreamReader(stream, chunk)

	ppp, tty, err := pty.Open()
	if err != nil {
		return err
	}
	defer tty.Close()

	cmd := exec.Command("/bin/bash")
	cmd.Env = os.Environ()

	cmd.Stdin = tty
	cmd.Stdout = tty
	cmd.Stderr = tty
	cmd.SysProcAttr = &syscall.SysProcAttr{Setctty: true, Setsid: true}

	go func() {
		_, err := io.Copy(w, ppp)
		fmt.Printf("stderr done: %v\n", err)
	}()
	go func() {
		_, err := io.Copy(ppp, r)
		fmt.Printf("stdin done: %v\n", err)
	}()
	defer fmt.Printf("all done\n")

	// wc, err := cmd.StdinPipe()
	// if err != nil {
	// 	return err
	// }
	// ro, err := cmd.StdoutPipe()
	// if err != nil {
	// 	return err
	// }
	// re, err := cmd.StderrPipe()
	// if err != nil {
	// 	return err
	// }
	// defer func() {
	// 	wc.Close()
	// 	ro.Close()
	// 	re.Close()
	// }()

	// go func() {
	// 	_, err := io.Copy(w, ro)
	// 	fmt.Printf("stdout done: %v\n", err)
	// }()
	// go func() {
	// 	_, err := io.Copy(w, re)
	// 	fmt.Printf("stderr done: %v\n", err)
	// }()
	// go func() {
	// 	_, err := io.Copy(wc, r)
	// 	fmt.Printf("stdin done: %v\n", err)
	// }()

	if err := cmd.Start(); err != nil {
		ppp.Close()
		return err
	}

	return cmd.Wait()
}
