package terminal

import (
	"fmt"
	"log"
)

var (
	PrefixColor = "\033[38;5;141m"
	Reset       = "\033[0m"
	Black       = "\033[30m"
	Red         = "\033[31m"
	Green       = "\033[32m"
	Yellow      = "\033[33m"
	Orange      = "\033[38;5;202m"
	Blue        = "\033[34m"
	Magenta     = "\033[35m"
	Cyan        = "\033[36m"
	Gray        = "\033[37m"
	White       = "\033[97m"
	Bold        = "\033[1m"
	Italic      = "\033[3m"
	Underline   = "\033[4m"
	Invert      = "\033[7m"

	BrightRed     = "\033[91m"
	BrightGreen   = "\033[92m"
	BrightYellow  = "\033[93m"
	BrightBlue    = "\033[94m"
	BrightMagenta = "\033[95m"
	BrightCyan    = "\033[96m"
	BrightWhite   = "\033[97m"
)

func color(input interface{}, color ...string) string {
	var s string
	c := ""
	for i := range color {
		c = c + color[i]
	}

	s = c + fmt.Sprint(input) + Reset
	return s
}

func Colorln(input interface{}, c ...string) {
	log.Println(color(input, c...))
}

func Sprintf(f string, input ...any) string {
	return fmt.Sprintf(f, input...)
}
