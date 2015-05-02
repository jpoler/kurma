// Copyright 2015 Apcera Inc. All rights reserved.

package aciremote

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/appc/spec/discovery"
)

// RetrieveImage can be used to retrieve a remote image, and optionally discover
// an image based on the App Container Image Discovery specification. Supports
// handling local images as well as
func RetrieveImage(imageUri string, insecure bool) (ReaderCloserSeeker, error) {
	u, err := url.Parse(imageUri)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "file":
		// for file:// urls, just load the file and return it
		return os.Open(u.Path)

	case "http", "https":
		// Handle HTTP retrievals, wrapped with a tempfile that cleans up.
		resp, err := http.Get(imageUri)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK:
		default:
			return nil, fmt.Errorf("HTTP %d on retrieving %q", imageUri)
		}

		return newTempReader(resp.Body)

	case "":
		app, err := discovery.NewAppFromString(imageUri)
		if err != nil {
			return nil, err
		}

		endpoints, _, err := discovery.DiscoverEndpoints(*app, insecure)
		if err != nil {
			return nil, err
		}

		for _, ep := range endpoints.ACIEndpoints {
			r, err := RetrieveImage(ep.ACI, insecure)
			if err != nil {
				continue
			}
			return r, nil
		}
		return nil, fmt.Errorf("failed to find a valid image for %q", imageUri)

	default:
		return nil, fmt.Errorf("%q scheme not supported", u.Scheme)
	}
}

// ReaderCloserSeeker is a generic interface for the functions common for images
// that are reteived. Seek is important as it will run through the image to
// locate the ACI manifest before actually extracting it.
type ReaderCloserSeeker interface {
	io.ReadCloser

	Seek(offset int64, whence int) (ret int64, err error)
}

// tempFileReader is an implementation of the ReaderCloserSeeker interface which
// is used with images that are remotely retrieved. It will download the remote
// file to a local temp file and ensures the file is removed when Close is
// called.
type tempFileReader struct {
	file *os.File
}

func newTempReader(r io.Reader) (*tempFileReader, error) {
	f, err := ioutil.TempFile(os.TempDir(), "remote-aci-tarfile")
	if err != nil {
		return nil, err
	}

	// construct the tempFileReader and setup its cleanup if the download fails
	tr := &tempFileReader{
		file: f,
	}
	success := false
	defer func() {
		if !success {
			tr.Close()
		}
	}()

	if _, err := io.Copy(tr.file, r); err != nil {
		return nil, err
	}

	if err := tr.file.Sync(); err != nil {
		return nil, err
	}

	if _, err := tr.file.Seek(0, 0); err != nil {
		return nil, err
	}
	success = true

	return tr, nil
}

func (r *tempFileReader) Read(p []byte) (int, error) {
	return r.file.Read(p)
}

func (r *tempFileReader) Seek(offset int64, whence int) (int64, error) {
	return r.file.Seek(offset, whence)
}

func (r *tempFileReader) Close() error {
	if err := r.file.Close(); err != nil {
		return err
	}
	if err := os.Remove(r.file.Name()); err != nil {
		return err
	}
	return nil
}
