// Copyright 2015 Apcera Inc. All rights reserved.

package main

import (
	"fmt"
	"net/url"
	"os"
	"runtime"

	kinit "github.com/apcera/kurma/init"
	"github.com/apcera/logray"
)

const (
	formatString = "%color:class%[%class%]%color:default% %message%"
)

func main() {
	u := url.URL{
		Scheme: "stdout",
		RawQuery: url.Values(map[string][]string{
			"format": []string{formatString},
		}).Encode(),
	}
	logray.AddDefaultOutput(u.String(), logray.INFOPLUS)

	if err := kinit.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failure running process: %v", err)
	}
	runtime.Goexit()
}
