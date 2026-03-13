package main

import (
	"os"

	"github.com/papercomputeco/sweeper/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
