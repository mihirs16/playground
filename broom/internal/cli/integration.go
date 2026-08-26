package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mihirs16/playground/broom/internal/apiclient"
	"github.com/mihirs16/playground/broom/internal/custodian"
	"github.com/spf13/cobra"
)

// knownSources is the set of third-party sources custodian polls. `refresh` with
// no name fans out over all of them; a named source is validated against this
// set so a typo reads as a legible error rather than a custodian 404.
var knownSources = []string{string(apiclient.Steam), string(apiclient.Github)}

func newIntegrationCmd(env Env) *cobra.Command {
	cmd := &cobra.Command{Use: "integration", Short: "Manage derived third-party content"}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "get [name]",
			Short: "Print a source's latest stored record without forcing a poll",
			Args:  cobra.MaximumNArgs(1),
			RunE: runIntegrationOver(env, func(c *custodian.Client, ctx context.Context, source string) (*apiclient.Integration, error) {
				return c.GetIntegration(ctx, source)
			}),
		},
		&cobra.Command{
			Use:   "refresh [name]",
			Short: "Force custodian to poll a source now and print the fresh record",
			Args:  cobra.MaximumNArgs(1),
			RunE: runIntegrationOver(env, func(c *custodian.Client, ctx context.Context, source string) (*apiclient.Integration, error) {
				return c.RefreshIntegration(ctx, source)
			}),
		},
	)
	return cmd
}

// runIntegrationOver resolves the target sources — the named one, or every known
// source when none is given — and prints each record fetch returns. `get` reads
// the last stored record; `refresh` forces a fresh poll first; the resolution,
// validation, and rendering are identical either way. A per-source error (a
// failed poll, an unreachable custodian) stops the run and is surfaced, so a
// gesture never reports partial success as whole success.
func runIntegrationOver(env Env, fetch func(*custodian.Client, context.Context, string) (*apiclient.Integration, error)) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		client, _, err := env.requireClient()
		if err != nil {
			return err
		}

		sources := knownSources
		if len(args) == 1 {
			if err := validateSource(args[0]); err != nil {
				return err
			}
			sources = []string{args[0]}
		}

		for _, source := range sources {
			record, err := fetch(client, cmd.Context(), source)
			if err != nil {
				return err
			}
			rendered, err := renderIntegration(record)
			if err != nil {
				return err
			}
			fmt.Fprintln(env.Stdout, rendered)
		}
		return nil
	}
}

// validateSource rejects an unknown source name before it reaches the wire,
// naming the sources custodian actually polls.
func validateSource(name string) error {
	for _, known := range knownSources {
		if name == known {
			return nil
		}
	}
	return fmt.Errorf("unknown source %q — known sources are %v", name, knownSources)
}

// renderIntegration prints a record as a one-line header (the source and when
// custodian last fetched it) followed by its data as indented JSON. A source
// never polled yet carries a zero timestamp and null data — the empty-but-
// present shape `get` can return — so both are spelled out rather than shown as
// a bare zero time or null.
func renderIntegration(record *apiclient.Integration) (string, error) {
	header := fmt.Sprintf("%s — never fetched", record.Source)
	if !record.FetchedAt.IsZero() {
		header = fmt.Sprintf("%s — fetched %s", record.Source, record.FetchedAt.UTC().Format("2006-01-02T15:04:05Z"))
	}
	if record.Data == nil {
		return header + "\n(no data observed yet)", nil
	}
	raw, err := json.MarshalIndent(record.Data, "", "  ")
	if err != nil {
		return "", err
	}
	return header + "\n" + string(raw), nil
}
