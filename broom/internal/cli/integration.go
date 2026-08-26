package cli

import (
	"encoding/json"
	"fmt"

	"github.com/mihirs16/playground/broom/internal/apiclient"
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
			Use:   "refresh [name]",
			Short: "Force custodian to poll a source now and print the fresh record",
			Args:  cobra.MaximumNArgs(1),
			RunE:  runIntegrationRefresh(env),
		},
	)
	return cmd
}

// runIntegrationRefresh forces custodian's manual poll and prints the record it
// polled, so a newly rotated Steam key or a fixed GitHub PAT can be verified
// without waiting for the next automatic tick. With a name it refreshes that one
// source; with none it refreshes every known source in turn. A failed poll on
// any source is surfaced as an error — the whole point is to learn the poll did
// not land.
func runIntegrationRefresh(env Env) func(*cobra.Command, []string) error {
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
			record, err := client.RefreshIntegration(cmd.Context(), source)
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

// renderIntegration prints a fresh record as a one-line header (the source and
// when custodian last fetched it) followed by its polled data as indented JSON.
// A source that has never observed data prints an explicit note rather than a
// bare null, so an empty-but-present record is unambiguous.
func renderIntegration(record *apiclient.Integration) (string, error) {
	header := fmt.Sprintf("%s — fetched %s", record.Source, record.FetchedAt.UTC().Format("2006-01-02T15:04:05Z"))
	if record.Data == nil {
		return header + "\n(no data observed yet)", nil
	}
	raw, err := json.MarshalIndent(record.Data, "", "  ")
	if err != nil {
		return "", err
	}
	return header + "\n" + string(raw), nil
}
