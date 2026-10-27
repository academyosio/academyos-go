// Command academyos checks, uploads, and publishes course-spec courses to
// an AcademyOS academy.
package main

import (
	"fmt"
	"os"
)

func main() {
	cmd := newRootCommand()
	err := cmd.Execute()
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(exitCode(err))
}
