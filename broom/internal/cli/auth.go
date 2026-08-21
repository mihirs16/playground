package cli

import (
	"errors"
	"fmt"

	"github.com/mihirs16/playground/broom/internal/config"
	"github.com/mihirs16/playground/broom/internal/custodian"
	"github.com/spf13/cobra"
)

func newLoginCmd(env Env) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Verify a token against custodian and store it",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := env.resolveConfig()
			if err != nil {
				return err
			}

			token, err := promptSecret(env, fmt.Sprintf("Enter token for %s: ", cfg.URL))
			if err != nil {
				return err
			}
			if token == "" {
				return errors.New("no token entered")
			}
			cfg.Token = token

			client, err := custodian.New(cfg)
			if err != nil {
				return err
			}
			if err := client.VerifyToken(cmd.Context()); err != nil {
				if custodian.IsUnauthorized(err) {
					return errors.New("token rejected — custodian did not accept that credential")
				}
				return err
			}

			if err := config.Save(env.ConfigPath, cfg); err != nil {
				return err
			}
			fmt.Fprintf(env.Stdout, "Logged in to %s\n", cfg.URL)
			return nil
		},
	}
}

func newLogoutCmd(env Env) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove the stored token from this machine",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := config.ClearToken(env.ConfigPath); err != nil {
				return err
			}
			fmt.Fprintln(env.Stdout, "Logged out.")
			return nil
		},
	}
}
