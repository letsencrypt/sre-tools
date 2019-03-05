package cmd

import "log"

// FailOnError exits and prints an error message, but only if we encountered
// a problem and err != nil
func FailOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)

	}
}
