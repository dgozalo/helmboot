package main

import (
	"os"

	"github.com/jenkins-x-labs/helmboot/cmd/app"
)

// Entrypoint for the command
func main() {
	if err := app.Run(nil); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
