// Copyright 2015 Apcera Inc. All rights reserved.

package init

import (
	"github.com/apcera/kurma/stage1/container"
	"github.com/apcera/logray"
)

// runner is an object that is used to handle the startup of the full KurmaOS
// system. It will take of the running of the process once init.Run() is
// invoked.
type runner struct {
	config  *kurmaConfig
	log     *logray.Logger
	manager *container.Manager
}

// Run takes over the process and launches KurmaOS.
func Run() error {
	r := &runner{
		config: defaultConfiguration(),
		log:    logray.New(),
	}
	return r.Run()
}

// Run handles executing the bootstrap setup. This prepares the current host
// environment to run and manage containers. It will return an error if any part
// of the setup fails.
func (r *runner) Run() error {
	r.log.Info("Launching KurmaOS\n\n")

	for _, f := range setupFunctions {
		if err := f(r); err != nil {
			r.log.Errorf("ERROR: %v", err)
			return err
		}
	}
	return nil
}
