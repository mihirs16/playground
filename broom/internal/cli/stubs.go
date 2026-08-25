package cli

import "github.com/spf13/cobra"

// authedStub is the body of a command whose wiring exists but whose behaviour is
// not built. It still enforces the credential gate so a not-logged-in run fails
// with the auth message rather than a bare stub notice.
func authedStub(env Env, what string) func(*cobra.Command, []string) error {
	return func(_ *cobra.Command, _ []string) error {
		if _, _, err := env.requireClient(); err != nil {
			return err
		}
		return notImplemented(what)
	}
}

func newIntegrationCmd(env Env) *cobra.Command {
	cmd := &cobra.Command{Use: "integration", Short: "Manage derived third-party content"}
	cmd.AddCommand(
		&cobra.Command{Use: "refresh [name]", Short: "Force custodian to poll a source now", Args: cobra.MaximumNArgs(1), RunE: authedStub(env, "integration refresh")},
	)
	return cmd
}
