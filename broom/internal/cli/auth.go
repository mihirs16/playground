package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mihirs16/playground/broom/internal/client"
	"github.com/mihirs16/playground/broom/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newLoginCmd(streams IO) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Store and verify a custodian admin token",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runLogin(cmd.Context(), streams)
		},
	}
}

func newLogoutCmd(streams IO) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove the stored custodian admin token",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := config.ClearToken(); err != nil {
				return err
			}
			fmt.Fprintln(streams.Out, "Logged out.")
			return nil
		},
	}
}

// runLogin resolves the target custodian, prompts for a token, verifies it with
// a single authenticated call, and only on success persists url + token. A
// rejected token is never written, so a failed login leaves any prior
// credential untouched.
func runLogin(ctx context.Context, streams IO) error {
	cfg, err := config.Resolve()
	if err != nil {
		return err
	}

	fmt.Fprintf(streams.Out, "Custodian: %s\n", cfg.URL)
	token, err := readToken(streams)
	if err != nil {
		return err
	}
	if token == "" {
		return fmt.Errorf("no token entered")
	}
	cfg.Token = token

	api, err := client.New(cfg, nil)
	if err != nil {
		return err
	}
	if err := api.Verify(ctx); err != nil {
		return fmt.Errorf("token rejected: %w", err)
	}

	if err := config.Save(cfg); err != nil {
		return err
	}
	fmt.Fprintln(streams.Out, "Logged in.")
	return nil
}

// readToken reads the token without echoing it when stdin is a real terminal,
// and falls back to a plain line read when it is piped — the shape the test
// harness feeds it.
func readToken(streams IO) (string, error) {
	fmt.Fprint(streams.Out, "Admin token: ")

	if file, ok := streams.In.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		raw, err := term.ReadPassword(int(file.Fd()))
		fmt.Fprintln(streams.Out)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(raw)), nil
	}

	line, err := bufio.NewReader(streams.In).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
