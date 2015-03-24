// Copyright 2015 Apcera Inc. All rights reserved.

package show

import (
	"encoding/json"
	"fmt"

	"github.com/apcera/kurma/client/cli"
	"github.com/appc/spec/schema"

	pb "github.com/apcera/kurma/stage1/client"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func init() {
	cli.DefineCommand("show", parseFlags, show, cliShow, "FIXME")
}

func parseFlags(cmd *cli.Cmd) {
}

func cliShow(cmd *cli.Cmd) error {
	if len(cmd.Args) == 0 || len(cmd.Args) > 1 {
		return fmt.Errorf("Invalid command options specified.")
	}
	return cmd.Run()
}

func show(cmd *cli.Cmd) error {
	conn, err := grpc.Dial("127.0.0.1:12311")
	if err != nil {
		return err
	}
	defer conn.Close()

	req := &pb.ContainerRequest{Uuid: cmd.Args[0]}

	c := pb.NewKurmaClient(conn)
	resp, err := c.Get(context.Background(), req)
	if err != nil {
		return err
	}

	fmt.Printf("Container %s:\n\n", resp.Uuid)

	// convert the manifest to the object
	var pod *schema.PodManifest
	if err := json.Unmarshal(resp.Manifest, &pod); err != nil {
		return err
	}

	// convert back with pretty mode
	b, err := json.MarshalIndent(pod, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", string(b))

	return nil
}
