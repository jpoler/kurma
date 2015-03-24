// Copyright 2015 Apcera Inc. All rights reserved.

package remote

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
// an image based on the App Container Image Discovery specification.
func RetrieveImage(imageurl string) (ReaderCloserSeeker, error) {
	u, err := url.Parse(imageurl)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "file":
		// for file:// urls, just load the file and return it
		return os.Open(u.Path)

	case "http", "https":
		// handle http retrievals, wrapped with a tempfile that cleans up
		resp, err := http.Get(imageurl)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		// check the status code to ensure success
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("received HTTP status %d retrieving image %q", resp.StatusCode, imageurl)
		}

		return newTempReader(resp.Body)

	case "":
		app, err := discovery.NewAppFromString(imageurl)
		if err != nil {
			return nil, err
		}

		// FIXME make insecure=true optional via config
		endpoints, _, err := discovery.DiscoverEndpoints(*app, true)
		if err != nil {
			return nil, err
		}

		for _, ep := range endpoints.ACIEndpoints {
			r, err := RetrieveImage(ep.ACI)
			if err != nil {
				continue
			}
			return r, nil
		}
		return nil, fmt.Errorf("failed to find a valid image for %q", imageurl)

	default:
		return nil, fmt.Errorf("%q scheme not implemented", u.Scheme)
	}
}

// ReaderCloserSeeker is a generic interface for the functions common for images
// that are reteived. Generally, Kurma will leverage Read, Close, and Seek. Seek
// is important as it will run through the image to locate the manifest before
// actually extracting it.
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
	f, err := ioutil.TempFile(os.TempDir(), "kurma-retriever")
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
