// Package cli assembles broom's command tree and wires each command to the
// config, the authenticated custodian client, and the process's IO. Everything
// the commands touch that is not custodian's HTTP boundary — stdin, stdout,
// stderr, the environment, the config file location — arrives through Env so the
// whole tree is drivable in a test.
package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mihirs16/playground/broom/internal/config"
	"github.com/mihirs16/playground/broom/internal/custodian"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// Env is everything a command depends on besides custodian's HTTP API. Tests
// substitute buffers, a scripted getenv, and a temp config path; main wires the
// real process.
type Env struct {
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
	Getenv     func(string) string
	ConfigPath string
	Edit       EditorFunc
	Copy       ClipboardFunc
}

// errNotLoggedIn is the single message any command shows when no usable
// credential is available, so an auth failure never reads as a content bug.
var errNotLoggedIn = errors.New("not logged in — run `broom login` (or set BROOM_TOKEN)")

// NewRootCmd builds the full command tree bound to env.
func NewRootCmd(env Env) *cobra.Command {
	root := &cobra.Command{
		Use:   "broom",
		Short: "broom — the custodian's implement for authoring playground content",
		Long: "broom is a client of custodian's admin API. It moves content from " +
			"your editor into custodian and holds no storage or authority of its own.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetIn(env.Stdin)
	root.SetOut(env.Stdout)
	root.SetErr(env.Stderr)

	root.AddCommand(
		newLoginCmd(env),
		newLogoutCmd(env),
		newLogsCmd(env),
		newMediaCmd(env),
		newProfileCmd(env),
		newIntegrationCmd(env),
	)
	return root
}

// resolveConfig produces the effective config for a command run.
func (env Env) resolveConfig() (config.Config, error) {
	return config.Resolve(env.ConfigPath, env.Getenv)
}

// requireClient resolves the config and builds an authenticated client, failing
// with the not-logged-in message when no token is available. Every command that
// talks to custodian's admin surface goes through here.
func (env Env) requireClient() (*custodian.Client, config.Config, error) {
	cfg, err := env.resolveConfig()
	if err != nil {
		return nil, config.Config{}, err
	}
	if cfg.Token == "" {
		return nil, cfg, errNotLoggedIn
	}
	client, err := custodian.New(cfg)
	if err != nil {
		return nil, cfg, err
	}
	return client, cfg, nil
}

// promptSecret reads a secret from stdin, echoing nothing when stdin is a real
// terminal and falling back to a plain line read when it is not (a pipe or a
// test buffer).
func promptSecret(env Env, prompt string) (string, error) {
	fmt.Fprint(env.Stderr, prompt)

	if f, ok := env.Stdin.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		b, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(env.Stderr)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(b)), nil
	}

	line, err := bufio.NewReader(env.Stdin).ReadString('\n')
	line = strings.TrimSpace(line)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return line, nil
}

// prompter reads a sequence of interactive answers from one buffered view of
// stdin, so consecutive prompts don't lose input to per-read buffering. Prompts
// are written to stderr, leaving stdout free for a command's real output.
type prompter struct {
	in  *bufio.Reader
	out io.Writer
	eof bool
}

func newPrompter(env Env) *prompter {
	return &prompter{in: bufio.NewReader(env.Stdin), out: env.Stderr}
}

// line reads one answer, showing label first. A trailing newline and surrounding
// whitespace are trimmed; EOF yields whatever was read (possibly empty), so an
// unterminated final answer is not lost. Once EOF is seen it is remembered, so a
// caller that must have an answer can stop rather than re-prompting into a
// closed stdin forever.
func (p *prompter) line(label string) (string, error) {
	fmt.Fprint(p.out, label)
	s, err := p.in.ReadString('\n')
	s = strings.TrimSpace(s)
	if errors.Is(err, io.EOF) {
		p.eof = true
	} else if err != nil {
		return "", err
	}
	return s, nil
}

// notImplemented reports a subcommand whose wiring is live but whose behaviour
// is not built.
func notImplemented(what string) error {
	return fmt.Errorf("%s is not implemented yet", what)
}
