// Copyright 2015 Apcera Inc. All rights reserved.

package enter

import (
	"fmt"
	"io"
	"os"

	"github.com/apcera/kurma/client/cli"
	"github.com/creack/termios/raw"

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
	// Set the local terminal in raw mode to turn off buffering and local
	// echo. Also defers setting it back to normal for when the call is done.
	termios, err := raw.MakeRaw(os.Stdin.Fd())
	if err != nil {
		return err
	}
	defer raw.TcSetAttr(os.Stdin.Fd(), termios)

	// Call the server
	conn, err := grpc.Dial("127.0.0.1:12311")
	if err != nil {
		return err
	}
	defer conn.Close()

	// Initialize the call and send the first packet so that it knows what
	// container we're connecting to.
	c := pb.NewKurmaClient(conn)
	stream, err := c.Enter(context.Background())
	if err != nil {
		return err
	}
	w := pb.NewByteStreamWriter(stream, cmd.Args[0])
	r := pb.NewByteStreamReader(stream, nil)
	if _, err := w.Write(nil); err != nil {
		return err
	}

	go io.Copy(w, os.Stdin)
	io.Copy(os.Stdout, r)
	stream.CloseSend()
	return nil
}
