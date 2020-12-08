package terminal

import (
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-tty"
)

const defaultWidth uint = 100

var terminalWidth uint
var clearer string

func init() {
	var err error

	t, err := tty.Open()
	if err == nil {
		var width int
		if width, _, err = t.Size(); err == nil {
			terminalWidth = uint(width)
		}
		t.Close()
	}
	if err != nil || terminalWidth <= 0 {
		fmt.Fprintf(os.Stderr, "couldn't determine the width of the terminal, defaulting to %d", defaultWidth)
		terminalWidth = defaultWidth
	}

	clearer = strings.Repeat(" ", int(terminalWidth)-1)
}

// ClearLine clears the current line in the terminal
func ClearLine() {
	fmt.Printf("\r%s\r", clearer)
}

// Overwritef clears the current line in the terminal and then performs
// a fmt.Printf() with the provided args
func Overwritef(format string, s ...interface{}) {
	fmt.Printf("\r%s\r%s", clearer, fmt.Sprintf(format, s...))
}
