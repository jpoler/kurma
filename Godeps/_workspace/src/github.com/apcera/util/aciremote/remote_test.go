// Copyright 2015 Apcera Inc. All rights reserved.

package aciremote

import (
	"io/ioutil"
	"os"

	"testing"
)

func TestRetrieveLocalFile(t *testing.T) {
	f, err := ioutil.TempFile(os.TempDir(), "localACi")
	if err != nil {
		t.Fatalf("Error creating temp file: %s", err)
	}
	defer f.Close()

	uri := "file://" + f.Name()

	reader, err := RetrieveImage(uri, false)
	if err != nil {
		t.Fatalf("Expected no error retrieving %s; got %s", uri, err)
	}
	reader.Close()
}

func TestRetrieveUnsupportedScheme(t *testing.T) {
	uri := "fakescheme://google.com"

	_, err := RetrieveImage(uri, false)
	if err == nil {
		t.Fatalf("Expected error with URI %q, got none", uri)
	}
}
