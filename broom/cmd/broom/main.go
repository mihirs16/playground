// Command broom is the custodian's implement: a command-line client for
// configuring, editing and writing playground content. It holds no storage or
// authority of its own — every action is a call to custodian's API.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mihirs16/playground/broom/internal/cli"
)

func main() {
	root := cli.NewRoot(cli.IO{In: os.Stdin, Out: os.Stdout, Err: os.Stderr})
	if err := root.ExecuteContext(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "broom:", err)
		os.Exit(1)
	}
}
