// Copyright 2015 Apcera Inc. All rights reserved.

package main

import (
	"fmt"
	"os"

	"github.com/apcera/kurma/bootstrap"
	"github.com/apcera/kurma/stage1/server"
	"github.com/apcera/logray"

	_ "github.com/apcera/kurma/stage2"
)

func main() {
	logray.AddDefaultOutput("stdout://", logray.ALL)

	// check if we're pid 1
	if os.Getpid() == 1 {
		if err := bootstrap.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to bootstrap: %v\n", err)
			os.Exit(1)
			return
		}
	}

	directory, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	opts := &server.Options{
		ParentCgroupName:   "kurma",
		ContainerDirectory: directory,
	}

	s := server.New(opts)
	if err := s.Start(); err != nil {
		panic(err)
	}
}
