package errors

import (
	"fmt"
	"os"
)

type ExitCode int

const (
	ExitSuccess ExitCode = 0
	// general error
	ExitGeneral ExitCode = 1
	// configuration error
	ExitConfig ExitCode = 2
	// git-related error
	ExitGit ExitCode = 3
)

func FatalError(code ExitCode, format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(int(code))
}
