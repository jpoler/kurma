// Copyright 2015 Apcera Inc. All rights reserved.

package create

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/apcera/continuum/apc/utils"
	"github.com/apcera/kurma/client/cli"
	"github.com/apcera/util/tarhelper"

	pb "github.com/apcera/kurma/stage1/client"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func init() {
	cli.DefineCommand("create", parseFlags, create, cliCreate, "FIXME")
}

func parseFlags(cmd *cli.Cmd) {
}

func cliCreate(cmd *cli.Cmd) error {
	if len(cmd.Args) == 0 || len(cmd.Args) > 1 {
		return utils.NewUsageError("Invalid command options specified.")
	}
	return cmd.Run()
}

func create(cmd *cli.Cmd) error {
	// open the file
	f, err := os.Open(cmd.Args[0])
	if err != nil {
		return err
	}
	defer f.Close()

	// find the manifest file, then rewind
	manifest, err := findManifest(f)
	if err != nil {
		return err
	}
	if _, err := f.Seek(0, 0); err != nil {
		return err
	}

	// dial the kurma server
	conn, err := grpc.Dial("localhost:12311")
	if err != nil {
		return err
	}
	defer conn.Close()

	// Upload the manifest and start the creation
	c := pb.NewKurmaClient(conn)
	req := &pb.CreateRequest{
		Manifest: manifest,
	}

	// trigger container creation then upload the ACI image
	resp, err := c.Create(context.Background(), req)
	if err != nil {
		return err
	}
	stream, err := c.UploadImage(context.Background())
	if err != nil {
		return err
	}
	w := &UploadStreamWriter{stream: stream, imageId: resp.ImageUploadId}
	if _, err := io.Copy(w, f); err != nil {
		return fmt.Errorf("write error: %v", err)
	}
	if _, err := stream.CloseAndRecv(); err != nil {
		return err
	}

	// fmt.Printf("Launched container %s\n", resp.Uuid)
	return nil
}

type UploadStreamWriter struct {
	imageId string
	stream  pb.Kurma_UploadImageClient
}

func (w *UploadStreamWriter) Write(p []byte) (int, error) {
	packet := &pb.ImageUpload{
		ImageUploadId: w.imageId,
		Bytes:         p,
	}
	return len(p), w.stream.Send(packet)
}

func findManifest(r io.Reader) ([]byte, error) {
	arch, err := tarhelper.DetectArchiveCompression(r)
	if err != nil {
		return nil, err
	}

	for {
		header, err := arch.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("failed to locate manifest file")
		}
		if err != nil {
			return nil, err
		}

		if filepath.Clean(header.Name) != "manifest" {
			continue
		}

		b, err := ioutil.ReadAll(arch)
		if err != nil {
			return nil, err
		}
		return b, nil
	}
}
