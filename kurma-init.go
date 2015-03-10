// Copyright 2015 Apcera Inc. All rights reserved.

package main

import (
	"os"

	"github.com/apcera/kurma/stage1/server"

	_ "github.com/apcera/kurma/stage2"
)

func main() {
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
