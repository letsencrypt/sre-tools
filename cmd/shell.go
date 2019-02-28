package cmd

import (
	"fmt"
	"os"

	blog "github.com/letsencrypt/boulder/log"
)

// Fail exits and prints an error message to stderr and the logger audit log.
func Fail(msg string) {
	logger := blog.Get()
	logger.AuditErr(msg)
	fmt.Fprintf(os.Stderr, msg)
	os.Exit(1)
}

// FailOnError exits and prints an error message, but only if we encountered
// a problem and err != nil
func FailOnError(err error, msg string) {
	if err != nil {
		msg := fmt.Sprintf("%s: %s", msg, err)
		Fail(msg)
	}
}
