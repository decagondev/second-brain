// Command sb is the Second Brain CLI. See manual/sb.1.md.
package main

import (
	"os"

	"example.com/second-brain/internal/cli"
)

func main() { os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr)) }
