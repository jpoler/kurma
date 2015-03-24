// Copyright 2015 Apcera Inc. All rights reserved.

package server

import (
	pb "github.com/apcera/kurma/stage1/client"
	"github.com/apcera/kurma/stage1/container"
)

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
