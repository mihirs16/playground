package cli

import "github.com/spf13/cobra"

// authedStub is the body of a command whose wiring exists but whose behaviour
// lands in a later ticket. It still enforces the credential gate so a
// not-logged-in run fails with the auth message rather than a bare stub notice.
func authedStub(env Env, what string) func(*cobra.Command, []string) error {
	return func(_ *cobra.Command, _ []string) error {
		if _, _, err := env.requireClient(); err != nil {
			return err
		}
		return notImplemented(what)
	}
}

func newLogsCmd(env Env) *cobra.Command {
	cmd := &cobra.Command{Use: "logs", Short: "Author and manage logs"}
	cmd.AddCommand(
		&cobra.Command{Use: "new", Short: "Create a log and edit its body", RunE: authedStub(env, "logs new")},
		&cobra.Command{Use: "edit <slug>", Short: "Edit a log's body", Args: cobra.ExactArgs(1), RunE: authedStub(env, "logs edit")},
		&cobra.Command{Use: "meta <slug>", Short: "Edit a log's metadata", Args: cobra.ExactArgs(1), RunE: authedStub(env, "logs meta")},
		&cobra.Command{Use: "rename <slug> <new-slug>", Short: "Rename an unlisted log's slug", Args: cobra.ExactArgs(2), RunE: authedStub(env, "logs rename")},
		&cobra.Command{Use: "list", Short: "List logs in any state", RunE: authedStub(env, "logs list")},
		&cobra.Command{Use: "rm <slug>", Short: "Delete a log", Args: cobra.ExactArgs(1), RunE: authedStub(env, "logs rm")},
		&cobra.Command{Use: "publish <slug>", Short: "List a log in the public index", Args: cobra.ExactArgs(1), RunE: authedStub(env, "logs publish")},
		&cobra.Command{Use: "unpublish <slug>", Short: "Remove a log from the public index", Args: cobra.ExactArgs(1), RunE: authedStub(env, "logs unpublish")},
	)
	return cmd
}

func newMediaCmd(env Env) *cobra.Command {
	cmd := &cobra.Command{Use: "media", Short: "Upload and manage media"}
	cmd.AddCommand(
		&cobra.Command{Use: "add <file>", Short: "Upload a file and print its markdown reference", Args: cobra.ExactArgs(1), RunE: authedStub(env, "media add")},
		&cobra.Command{Use: "ls", Short: "List and search media", RunE: authedStub(env, "media ls")},
		&cobra.Command{Use: "rm <key>", Short: "Delete a media record", Args: cobra.ExactArgs(1), RunE: authedStub(env, "media rm")},
	)
	return cmd
}

func newProfileCmd(env Env) *cobra.Command {
	cmd := &cobra.Command{Use: "profile", Short: "Inspect and edit profile records"}
	cmd.AddCommand(
		&cobra.Command{Use: "get <key>", Short: "Show a profile record's JSON", Args: cobra.ExactArgs(1), RunE: authedStub(env, "profile get")},
		&cobra.Command{Use: "edit <key>", Short: "Edit a profile record's JSON", Args: cobra.ExactArgs(1), RunE: authedStub(env, "profile edit")},
	)
	return cmd
}

func newIntegrationCmd(env Env) *cobra.Command {
	cmd := &cobra.Command{Use: "integration", Short: "Manage derived third-party content"}
	cmd.AddCommand(
		&cobra.Command{Use: "refresh [name]", Short: "Force custodian to poll a source now", Args: cobra.MaximumNArgs(1), RunE: authedStub(env, "integration refresh")},
	)
	return cmd
}
