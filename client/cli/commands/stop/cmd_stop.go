// Copyright 2015 Apcera Inc. All rights reserved.

package stop

import (
	"fmt"

	"github.com/apcera/kurma/client/cli"

	pb "github.com/apcera/kurma/stage1/client"
	"golang.org/x/net/context"
)

func init() {
	cli.DefineCommand("stop", parseFlags, stop, cliStop, "FIXME")
}

func parseFlags(cmd *cli.Cmd) {
}

func cliStop(cmd *cli.Cmd) error {
	if len(cmd.Args) == 0 || len(cmd.Args) > 1 {
		return fmt.Errorf("Invalid command options specified.")
	}
	return cmd.Run()
}

func stop(cmd *cli.Cmd) error {
	req := &pb.ContainerRequest{Uuid: cmd.Args[0]}

	if _, err := cmd.Client.Destroy(context.Background(), req); err != nil {
		return err
	}

	fmt.Printf("Destroyed container %s\n", cmd.Args[0])
	return nil
}
