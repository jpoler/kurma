// Copyright 2015 Apcera Inc. All rights reserved.

package remote

import (
	"fmt"
	"io"
	"net/url"
	"os"
)

type ReaderCloserSeeker interface {
	io.ReadCloser
	Seek(offset int64, whence int) (ret int64, err error)
}

func RetrieveImage(imageurl string) (ReaderCloserSeeker, error) {
	u, err := url.Parse(imageurl)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "file":
		// for file:// urls, just load the file and return it
		return os.Open(u.Path)
	default:
		return nil, fmt.Errorf("not implemented")
	}
}
