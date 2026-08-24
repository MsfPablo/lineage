package main

import (
	"context"
	"os"

	"github.com/agentic-lineage/lineage/internal/cli"
)

func main() {
	if err := cli.Execute(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
}
