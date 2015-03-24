// Copyright 2015 Apcera Inc. All rights reserved.

package client

import (
	"io"
)

// NewByteStreamWriter generates a new io.Writer to use with a stream.
func NewByteStreamWriter(stream ByteStreamSender, streamId string) io.Writer {
	return &byteStreamWriter{
		streamId: streamId,
		stream:   stream,
	}
}

// NewByteSreamReader generates a new io.ReadClose to use with a stream.
func NewByteStreamReader(stream ByteStreamReceiver, chunk *ByteChunk) io.ReadCloser {
	r := &byteStreamReader{
		stream: stream,
	}

	if chunk != nil {
		r.buf = chunk.Bytes
	} else {
		r.buf = make([]byte, 0)
	}

	return r
}

// A generic interface that is used to describe the sending end of a stream.
type ByteStreamSender interface {
	Send(*ByteChunk) error
}

// A generic interface that is used to describe the receiving end of a stream.
// a ByteChunk.
type ByteStreamReceiver interface {
	Recv() (*ByteChunk, error)
}

type streamSendAndClose interface {
	SendAndClose(*None) error
}
type streamCloseSend interface {
	CloseSend() error
}

type byteStreamWriter struct {
	streamId string
	stream   ByteStreamSender
}

func (w *byteStreamWriter) Write(p []byte) (int, error) {
	chunk := &ByteChunk{
		StreamId: w.streamId,
		Bytes:    p,
	}
	return len(p), w.stream.Send(chunk)
}

type byteStreamReader struct {
	stream      ByteStreamReceiver
	buf         []byte
	eofReceived bool
}

func (r *byteStreamReader) Read(p []byte) (int, error) {
	pn := len(p)
	bn := len(r.buf)

	// if we have no data, and have received an eof, return eof
	if bn == 0 && r.eofReceived {
		return 0, io.EOF
	}

	// if we don't have anything in the buffer, then request a chunk
	if bn == 0 {
		chunk, err := r.stream.Recv()
		if err != nil {
			if err == io.EOF {
				r.eofReceived = true
			} else {
				return 0, err
			}
		} else {
			r.buf = append(r.buf, chunk.Bytes...)
			bn = len(r.buf)
		}
	}

	// if we have more data in the buffer than p can fit, then scope it to p
	if bn >= pn {
		copy(p, r.buf[:pn])
		r.buf = r.buf[pn:]
		return pn, nil
	}

	// otherwise, just write what we have
	pn = len(r.buf)
	copy(p, r.buf)
	r.buf = nil
	return pn, nil
}

func (r *byteStreamReader) Close() error {
	if s, ok := r.stream.(streamSendAndClose); ok {
		return s.SendAndClose(&None{})
	}
	if s, ok := r.stream.(streamCloseSend); ok {
		return s.CloseSend()
	}
	return nil
}
