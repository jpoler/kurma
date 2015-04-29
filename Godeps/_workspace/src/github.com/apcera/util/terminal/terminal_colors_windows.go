// Copyright 2014-2015 Apcera Inc. All rights reserved.

// +build windows

package terminal

//FIXME(Sha): These color codes aren't correct for windows.  Leaving here for compilation reasons.
//  Should be fixed in the future when windows term colors are implemented.
var ColorError int = 167
var ColorWarn int = 93
var ColorSuccess int = 82
var BackgroundColorBlack = "\033[30;49m"
var BackgroundColorWhite = "\033[30;47m"
var ResetCode string = "\033[0m"

//---------------------------------------------------------------------------
// Display helpers
//---------------------------------------------------------------------------

func Colorize(color int, msg string) string {
	return msg
}

// BoldText returns the bold-ified version of the passed-in string.
func BoldText(msg string) string {
	return msg
}
