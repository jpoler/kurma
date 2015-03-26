// Copyright 2015 Apcera Inc. All rights reserved.

package enter

import (
	"fmt"
	"io"
	"os"

	"github.com/apcera/kurma/client/cli"
	"github.com/nsf/termbox-go"

	pb "github.com/apcera/kurma/stage1/client"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func init() {
	cli.DefineCommand("enter", parseFlags, enter, cliEnter, "FIXME")
}

func parseFlags(cmd *cli.Cmd) {
}

func cliEnter(cmd *cli.Cmd) error {
	if len(cmd.Args) == 0 || len(cmd.Args) > 1 {
		return fmt.Errorf("Invalid command options specified.")
	}
	return cmd.Run()
}

func enter(cmd *cli.Cmd) error {
	err := termbox.Init()
	if err != nil {
		return err
	}
	defer termbox.Close()

	conn, err := grpc.Dial("127.0.0.1:12311")
	if err != nil {
		return err
	}
	defer conn.Close()

	c := pb.NewKurmaClient(conn)
	stream, err := c.Enter(context.Background())
	if err != nil {
		return err
	}
	w := pb.NewByteStreamWriter(stream, cmd.Args[0])
	r := pb.NewByteStreamReader(stream, nil)
	w.Write(nil)

	go func() {
		_, err := io.Copy(w, os.Stdin)
		fmt.Printf("writer done: %v\n", err)
	}()
	_, err = io.Copy(os.Stdout, r)
	fmt.Printf("reader done: %v\n", err)
	stream.CloseSend()
	fmt.Printf("done!\n")
	return nil
}
