// Copyright 2014-2015 Apcera Inc. All rights reserved.

// +build !windows

package terminal

import "fmt"

var ColorError int = 167
var ColorWarn int = 93
var ColorSuccess int = 82
var BackgroundColorBlack = "\033[30;49m"
var BackgroundColorWhite = "\033[30;47m"
var ResetCode string = "\033[0m"

//---------------------------------------------------------------------------
// Display helpers
//---------------------------------------------------------------------------

// Colorize returns a string which is wrapped with the appropriate
// escape sequences to print a colorized verison of the string `msg`
func Colorize(color int, msg string) string {
	//FIXME(Sha): This logic assumes that because tty is set that the terminal supports colors.
	//	This needs to be fixed via bringing in PDCurses or some other library.
	if !stdoutIsTTY {
		return msg
	}
	return fmt.Sprintf("\033[0;1m\033[38;5;%dm%s%s", color, msg, ResetCode)
}

// BoldText returns the bold-ified version of the passed-in string.
func BoldText(msg string) string {
	if !stdoutIsTTY {
		return msg
	}
	return fmt.Sprintf("\033[1m%s\033[0m", msg)
}
