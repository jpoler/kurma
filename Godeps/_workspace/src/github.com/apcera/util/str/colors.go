// Copyright 2012 Apcera Inc. All rights reserved.

package str

import (
	"os"
)

var disableColors bool

func DisableColors() {
	disableColors = true
}

func ColorForFile(file *os.File, text string, colorIndex string,
	bold bool) string {
	if disableColors || IsTerminal(file) == false {
		return text
	}
	return Color(text, colorIndex, bold)
}

func Color(text string, colorIndex string, bold bool) string {
	if disableColors {
		return text
	}
	// Two explicit constructions to make it a bit faster
	if bold == false {
		return "\033[" + "38;5;" + colorIndex + "m" + text + "\033[0m"
	}
	return "\033[0;1m\033[" + "38;5;" + colorIndex + "m" + text + "\033[0m"
}
