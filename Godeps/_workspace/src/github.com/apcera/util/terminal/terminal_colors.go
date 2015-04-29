// Copyright 2014 Apcera Inc. All rights reserved.

package terminal

import "strconv"

//---------------------------------------------------------------------------
// Display helpers
//---------------------------------------------------------------------------

// ColorizeEscapeNamespace gives us a color namespace, escaped much like %q
// would, but without the quotes (as they don't get colored).
func ColorizeEscapeNamespace(ns string) string {
	ourNs := strconv.Quote(ns)
	if len(ourNs) < 2 {
		// should panic, this is an API violation from strconv.Quote
		return ns
	}
	ourNs = ourNs[1 : len(ourNs)-1]
	ourColor := ColorSuccess
	if ourNs != ns {
		ourColor = ColorWarn
	}
	return Colorize(ourColor, ourNs)
}
