// Copyright 2015 Apcera Inc. All rights reserved.

package server

import (
	"io"

	pb "github.com/apcera/kurma/stage1/client"
)

// imageUploadReader is used to give an io.Reader interface to the gRPC streamed
// input of the container image. This is done to allow the ACI image to be
// streamed in, rather than sent in bulk, and gives a common interface to read
// the binary data.
type imageUploadReader struct {
	stream pb.Kurma_UploadImageServer
	packet *pb.ImageUpload
	buf    []byte
}

func newImageUploadReader(stream pb.Kurma_UploadImageServer, packet *pb.ImageUpload) *imageUploadReader {
	return &imageUploadReader{
		stream: stream,
		packet: packet,
		buf:    make([]byte, 0),
	}
}

func (r *imageUploadReader) Read(p []byte) (int, error) {
	n := len(p)

	if len(r.buf) >= n {
		copy(p, r.buf[:n])
		r.buf = r.buf[n:]
		return n, nil
	}

	if r.packet != nil {
		r.buf = append(r.buf, r.packet.Bytes...)
		p, err := r.stream.Recv()
		if err != nil && err != io.EOF {
			return 0, err
		}
		r.packet = p
	}

	if len(r.buf) >= n {
		copy(p, r.buf[:n])
		r.buf = r.buf[n:]
		return n, nil
	}

	if r.packet == nil && len(r.buf) > 0 {
		n = len(r.buf)
		copy(p, r.buf)
		r.buf = nil
		return n, nil
	}

	return 0, io.EOF
}

func (r *imageUploadReader) Close() error {
	return r.stream.SendAndClose(&pb.None{})
}
