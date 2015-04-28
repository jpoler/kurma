// Copyright 2012 Apcera Inc. All rights reserved.

// +build ignore

package main

import (
	"fmt"
	target "github.com/apcera/util/str"
	"os"
)

func main() {
	for _, label := range []string{
		"stdin", "stdout", "stderr",
	} {
		var fh *os.File

		switch label {
		case "stdin":
			fh = os.Stdin
		case "stdout":
			fh = os.Stdout
		case "stderr":
			fh = os.Stderr
		default:
			panic("unknown label")
		}

		if target.IsTerminal(fh) {
			fmt.Printf("%s is a terminal :)\n", label)
		} else {
			fmt.Printf("%s is not a terminal :(\n", label)
		}
	}
}
