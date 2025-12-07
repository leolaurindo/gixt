package main

import (
	"context"
	"os"

	"github.com/leolaurindo/gixt/internal/cli"
)

func main() {
	ctx := context.Background()
	if err := cli.Execute(ctx, os.Args[1:]); err != nil {
		cli.PrintError(err)
		os.Exit(1)
	}
}
