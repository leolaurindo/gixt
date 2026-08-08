package main

import (
	"context"
	"errors"
	"os"
	"os/exec"

	"github.com/leolaurindo/gixt/internal/cli"
)

func main() {
	if err := cli.Execute(context.Background(), os.Args[1:]); err != nil {
		cli.PrintError(err)
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
}
