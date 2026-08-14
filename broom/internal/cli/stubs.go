package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// The noun groups below stand up the whole command surface with real routing,
// help text, and argument parsing; each leaf is stubbed until its ticket lands.
// A stub is a visible "not built yet", never a silent no-op — running one
// fails, so a half-wired command can't be mistaken for a working one.

func newLogsCmd(streams IO) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Author and manage logs",
	}
	cmd.AddCommand(
		stub(streams, "list", "List logs"),
		stub(streams, "show", "Show a single log"),
		stub(streams, "create", "Create a log"),
		stub(streams, "edit", "Edit a log"),
		stub(streams, "publish", "List a log in the blog index"),
		stub(streams, "unpublish", "Remove a log from the blog index"),
		stub(streams, "delete", "Delete a log"),
	)
	return cmd
}

func newMediaCmd(streams IO) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "media",
		Short: "Upload and manage media",
	}
	cmd.AddCommand(
		stub(streams, "list", "List media"),
		stub(streams, "show", "Show a single media item"),
		stub(streams, "upload", "Upload a media file"),
		stub(streams, "delete", "Delete a media item"),
	)
	return cmd
}

func newProfileCmd(streams IO) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Read and write profile records",
	}
	cmd.AddCommand(
		stub(streams, "get", "Show a profile record"),
		stub(streams, "set", "Write a profile record"),
	)
	return cmd
}

func newIntegrationCmd(streams IO) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "integration",
		Short: "Inspect and refresh derived integrations",
	}
	cmd.AddCommand(
		stub(streams, "show", "Show an integration's latest state"),
		stub(streams, "refresh", "Force custodian to re-poll an integration"),
	)
	return cmd
}

func stub(streams IO, use, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("%q is not implemented yet", use)
		},
	}
}
