package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mihirs16/playground/broom/internal/apiclient"
	"github.com/mihirs16/playground/broom/internal/custodian"
	"github.com/spf13/cobra"
)

func newProfileCmd(env Env) *cobra.Command {
	cmd := &cobra.Command{Use: "profile", Short: "Inspect and edit profile records"}
	cmd.AddCommand(
		&cobra.Command{Use: "get <key>", Short: "Show a profile record's JSON", Args: cobra.ExactArgs(1), RunE: runProfileGet(env)},
		&cobra.Command{Use: "edit <key>", Short: "Edit a profile record's JSON", Args: cobra.ExactArgs(1), RunE: runProfileEdit(env)},
	)
	return cmd
}

// runProfileGet fetches a profile record and prints its body as indented JSON,
// so the current value can be inspected. The body is opaque — broom imposes no
// schema — so it is shown verbatim as custodian stored it.
func runProfileGet(env Env) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		client, _, err := env.requireClient()
		if err != nil {
			return err
		}
		key := args[0]

		record, err := client.GetProfile(cmd.Context(), key)
		if err != nil {
			return err
		}
		rendered, err := renderProfileBody(record.Body)
		if err != nil {
			return err
		}
		fmt.Fprintln(env.Stdout, rendered)
		return nil
	}
}

// runProfileEdit round-trips a profile record's raw JSON through $EDITOR and
// PUT-upserts it on save. A record that does not exist yet opens as an empty
// object, so edit is the way to author a new key as well as revise one. An
// unchanged body is left alone — there is nothing to send.
func runProfileEdit(env Env) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		client, _, err := env.requireClient()
		if err != nil {
			return err
		}
		key := args[0]

		initial, err := currentProfileJSON(cmd.Context(), client, key)
		if err != nil {
			return err
		}

		edited, err := editViaTemp(env, "broom-profile-*.json", initial)
		if err != nil {
			return err
		}
		if edited == initial {
			fmt.Fprintf(env.Stdout, "No changes to %s\n", key)
			return nil
		}

		body, err := parseProfileBody(edited)
		if err != nil {
			return err
		}
		if _, err := client.PutProfile(cmd.Context(), key, body); err != nil {
			return err
		}
		fmt.Fprintf(env.Stdout, "Saved profile %s\n", key)
		return nil
	}
}

// currentProfileJSON pulls a record's body as indented JSON to seed the editor,
// treating a missing record as an empty object so a new key can be authored.
func currentProfileJSON(ctx context.Context, client *custodian.Client, key string) (string, error) {
	record, err := client.GetProfile(ctx, key)
	if err != nil {
		if custodian.IsNotFound(err) {
			return "{}\n", nil
		}
		return "", err
	}
	return renderProfileBody(record.Body)
}

// renderProfileBody marshals an opaque profile body as indented JSON with a
// trailing newline, the form both `get` prints and `edit` opens on.
func renderProfileBody(body apiclient.ProfileBody) (string, error) {
	raw, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return "", err
	}
	return string(raw) + "\n", nil
}

// parseProfileBody decodes the author's edited JSON back into an opaque body,
// failing with a legible message when the text is not valid JSON so a typo is
// caught before it reaches the wire.
func parseProfileBody(text string) (apiclient.ProfileBody, error) {
	var body apiclient.ProfileBody
	if err := json.Unmarshal([]byte(text), &body); err != nil {
		return nil, fmt.Errorf("edited profile is not valid JSON: %w", err)
	}
	return body, nil
}
