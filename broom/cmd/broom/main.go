// Command broom is the custodian's command-line implement for authoring
// playground content. It is a pure client of custodian's admin API.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mihirs16/playground/broom/internal/cli"
	"github.com/mihirs16/playground/broom/internal/config"
)

func main() {
	configPath, err := config.DefaultPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, "broom: cannot locate config directory:", err)
		os.Exit(1)
	}

	root := cli.NewRootCmd(cli.Env{
		Stdin:      os.Stdin,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		Getenv:     os.Getenv,
		ConfigPath: configPath,
		Edit:       cli.RealEditor(os.Getenv),
	})

	if err := root.ExecuteContext(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "broom:", err)
		os.Exit(1)
	}
}
