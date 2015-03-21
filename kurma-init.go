// Copyright 2015 Apcera Inc. All rights reserved.

package main

import (
	"net/url"
	"runtime"

	kinit "github.com/apcera/kurma/init"
	"github.com/apcera/logray"

	_ "github.com/apcera/kurma/stage2"
)

const (
	formatString = "%color:class%[%classfixed%]%color:default% %message%"
)

func main() {
	u := url.URL{
		Scheme: "stdout",
		RawQuery: url.Values(map[string][]string{
			"format": []string{formatString},
		}).Encode(),
	}
	logray.AddDefaultOutput(u.String(), logray.ALL)

	if err := kinit.Run(); err != nil {
		panic(err)
	}
	runtime.Goexit()
}
