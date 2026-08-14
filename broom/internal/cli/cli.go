// Package cli assembles broom's command tree. The tree is noun-grouped — logs,
// media, profile, integration — with login and logout at the top level. Every
// command shares one IO so an in-process test can drive the real wiring against
// a fake custodian without touching the process's stdin/stdout.
package cli

import (
	"io"

	"github.com/spf13/cobra"
)

// IO is the set of streams a command reads from and writes to. Threading it
// through the tree (rather than reaching for os.Stdin/os.Stdout) is what lets
// the test harness capture output and feed a token on stdin.
type IO struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
}

// NewRoot builds the full command tree bound to the given IO.
func NewRoot(streams IO) *cobra.Command {
	root := &cobra.Command{
		Use:           "broom",
		Short:         "The custodian's implement — configure, edit and write playground content",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetIn(streams.In)
	root.SetOut(streams.Out)
	root.SetErr(streams.Err)

	root.AddCommand(
		newLoginCmd(streams),
		newLogoutCmd(streams),
		newLogsCmd(streams),
		newMediaCmd(streams),
		newProfileCmd(streams),
		newIntegrationCmd(streams),
	)
	return root
}
