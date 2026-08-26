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
