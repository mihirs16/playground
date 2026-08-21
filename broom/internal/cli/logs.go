package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mihirs16/playground/broom/internal/apiclient"
	"github.com/mihirs16/playground/broom/internal/custodian"
	"github.com/spf13/cobra"
)

func newLogsCmd(env Env) *cobra.Command {
	cmd := &cobra.Command{Use: "logs", Short: "Author and manage logs"}
	cmd.AddCommand(
		&cobra.Command{Use: "new", Short: "Create a log and edit its body", Args: cobra.NoArgs, RunE: runLogsNew(env)},
		&cobra.Command{Use: "edit <slug>", Short: "Edit a log's body", Args: cobra.ExactArgs(1), RunE: runLogsEdit(env)},
		&cobra.Command{Use: "meta <slug>", Short: "Edit a log's metadata", Args: cobra.ExactArgs(1), RunE: authedStub(env, "logs meta")},
		&cobra.Command{Use: "rename <slug> <new-slug>", Short: "Rename an unlisted log's slug", Args: cobra.ExactArgs(2), RunE: authedStub(env, "logs rename")},
		newLogsListCmd(env),
		&cobra.Command{Use: "rm <slug>", Short: "Delete a log", Args: cobra.ExactArgs(1), RunE: runLogsRm(env)},
		&cobra.Command{Use: "publish <slug>", Short: "List a log in the public index", Args: cobra.ExactArgs(1), RunE: runLogsSetState(env, apiclient.Listed, "Published")},
		&cobra.Command{Use: "unpublish <slug>", Short: "Remove a log from the public index", Args: cobra.ExactArgs(1), RunE: runLogsSetState(env, apiclient.Unlisted, "Unpublished")},
	)
	return cmd
}

// newLogsListCmd builds `logs list`, whose --listed/--unlisted flags narrow the
// listing to a single state. The flags are mutually exclusive; with neither, all
// states are shown — the point of the admin listing is that drafts the public
// index hides are still visible here.
func newLogsListCmd(env Env) *cobra.Command {
	cmd := &cobra.Command{Use: "list", Short: "List logs in any state", Args: cobra.NoArgs, RunE: runLogsList(env)}
	cmd.Flags().Bool("listed", false, "show only listed posts")
	cmd.Flags().Bool("unlisted", false, "show only unlisted drafts")
	cmd.MarkFlagsMutuallyExclusive("listed", "unlisted")
	return cmd
}

// runLogsNew is the authoring loop: prompt for metadata, create the post
// immediately as an unlisted draft (the slug-is-the-post invariant — there is
// never a started post custodian does not know about), then open the empty body
// in $EDITOR and PATCH it on save. An empty body, including aborting the editor
// without writing, is valid — it leaves the freshly-created empty draft as is.
func runLogsNew(env Env) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		client, _, err := env.requireClient()
		if err != nil {
			return err
		}

		p := newPrompter(env)
		create, err := promptMetadata(p)
		if err != nil {
			return err
		}

		created, err := createWithSlugRetry(cmd.Context(), client, p, create)
		if err != nil {
			return err
		}
		fmt.Fprintf(env.Stdout, "Created unlisted draft %s\n", created.Slug)

		body, err := editViaTemp(env, "broom-body-*.md", "")
		if err != nil {
			return err
		}
		if body == "" {
			fmt.Fprintf(env.Stdout, "Empty draft — nothing to save yet.\n")
			return nil
		}
		if _, err := client.PatchLog(cmd.Context(), created.Slug, apiclient.LogPatch{Body: &body}); err != nil {
			return err
		}
		fmt.Fprintf(env.Stdout, "Saved body of %s\n", created.Slug)
		return nil
	}
}

// runLogsEdit round-trips an existing post's body: pull the current body into a
// temp file, open $EDITOR, and PATCH on save. An unchanged body is left alone —
// there is nothing to send.
func runLogsEdit(env Env) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		client, _, err := env.requireClient()
		if err != nil {
			return err
		}
		slug := args[0]

		existing, err := client.GetLog(cmd.Context(), slug)
		if err != nil {
			return err
		}

		edited, err := editViaTemp(env, "broom-body-*.md", existing.Body)
		if err != nil {
			return err
		}
		if edited == existing.Body {
			fmt.Fprintf(env.Stdout, "No changes to %s\n", slug)
			return nil
		}
		if _, err := client.PatchLog(cmd.Context(), slug, apiclient.LogPatch{Body: &edited}); err != nil {
			return err
		}
		fmt.Fprintf(env.Stdout, "Saved body of %s\n", slug)
		return nil
	}
}

// runLogsList shows the author's posts of any state, so work in progress —
// including the unlisted drafts the public index hides — can be found and
// managed. The --listed/--unlisted flags narrow to a single state.
func runLogsList(env Env) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		client, _, err := env.requireClient()
		if err != nil {
			return err
		}
		state, err := listStateFilter(cmd)
		if err != nil {
			return err
		}
		index, err := client.ListLogs(cmd.Context(), state)
		if err != nil {
			return err
		}
		printLogIndex(env, index)
		return nil
	}
}

// runLogsSetState toggles a post's state. Publishing and unpublishing are the
// same PATCH {state} on the wire — broom presents them as distinct verbs rather
// than bespoke endpoints.
func runLogsSetState(env Env, state apiclient.LogState, done string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		client, _, err := env.requireClient()
		if err != nil {
			return err
		}
		slug := args[0]
		updated, err := client.PatchLog(cmd.Context(), slug, apiclient.LogPatch{State: &state})
		if err != nil {
			return err
		}
		fmt.Fprintf(env.Stdout, "%s %s\n", done, updated.Slug)
		return nil
	}
}

// runLogsRm deletes a post entirely, so a draft or a retired post can be removed.
func runLogsRm(env Env) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		client, _, err := env.requireClient()
		if err != nil {
			return err
		}
		slug := args[0]
		if err := client.DeleteLog(cmd.Context(), slug); err != nil {
			return err
		}
		fmt.Fprintf(env.Stdout, "Deleted %s\n", slug)
		return nil
	}
}

// listStateFilter reads the mutually-exclusive --listed/--unlisted flags into a
// state filter, returning nil to list every state.
func listStateFilter(cmd *cobra.Command) (*apiclient.LogState, error) {
	listed, err := cmd.Flags().GetBool("listed")
	if err != nil {
		return nil, err
	}
	unlisted, err := cmd.Flags().GetBool("unlisted")
	if err != nil {
		return nil, err
	}
	switch {
	case listed:
		s := apiclient.Listed
		return &s, nil
	case unlisted:
		s := apiclient.Unlisted
		return &s, nil
	default:
		return nil, nil
	}
}

// printLogIndex renders the listing as one line per post — state, slug, title —
// so a draft is legible beside its state at a glance.
func printLogIndex(env Env, index *apiclient.LogIndex) {
	if len(index.Items) == 0 {
		fmt.Fprintln(env.Stdout, "No logs.")
		return
	}
	for _, item := range index.Items {
		fmt.Fprintf(env.Stdout, "%-8s  %-24s  %s\n", item.State, item.Slug, item.Title)
	}
}

// promptMetadata collects the fields a new post is described by. Only the title
// is required; the slug defaults to a kebab-case rendering of the title that the
// author can accept or override. Absent optional fields are left unset so the
// create body carries only what the author actually entered.
func promptMetadata(p *prompter) (apiclient.LogCreate, error) {
	title, err := requiredLine(p, "Title: ")
	if err != nil {
		return apiclient.LogCreate{}, err
	}

	defaultSlug := slugify(title)
	slug, err := p.line(fmt.Sprintf("Slug [%s]: ", defaultSlug))
	if err != nil {
		return apiclient.LogCreate{}, err
	}
	if slug == "" {
		slug = defaultSlug
	}

	subtitle, err := p.line("Subtitle (optional): ")
	if err != nil {
		return apiclient.LogCreate{}, err
	}
	tagsLine, err := p.line("Tags (comma-separated, optional): ")
	if err != nil {
		return apiclient.LogCreate{}, err
	}
	description, err := p.line("Description (optional): ")
	if err != nil {
		return apiclient.LogCreate{}, err
	}

	create := apiclient.LogCreate{Slug: slug, Title: title}
	if subtitle != "" {
		create.Subtitle = &subtitle
	}
	if description != "" {
		create.Description = &description
	}
	if tags := parseTags(tagsLine); len(tags) > 0 {
		create.Tags = &tags
	}
	return create, nil
}

// createWithSlugRetry creates the draft, re-prompting for a fresh slug when
// custodian reports the slug is taken, so the author recovers in-flow rather
// than reading a raw conflict. An empty answer at the re-prompt gives up and
// surfaces the conflict.
func createWithSlugRetry(ctx context.Context, client *custodian.Client, p *prompter, create apiclient.LogCreate) (*apiclient.Log, error) {
	for {
		created, err := client.CreateLog(ctx, create)
		if err == nil {
			return created, nil
		}
		if !custodian.IsSlugConflict(err) {
			return nil, err
		}

		fmt.Fprintf(p.out, "Slug %q is already taken.\n", create.Slug)
		next, perr := p.line("Choose another slug: ")
		if perr != nil {
			return nil, perr
		}
		if next == "" {
			return nil, err
		}
		create.Slug = next
	}
}

// editViaTemp writes initial into a temp file, opens it in the author's editor,
// and returns the file's contents after the editor closes. The temp file is the
// only thing on disk — custodian stays canonical — and it is removed on return.
func editViaTemp(env Env, pattern, initial string) (string, error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	path := f.Name()
	defer os.Remove(path)

	if initial != "" {
		if _, err := f.WriteString(initial); err != nil {
			f.Close()
			return "", err
		}
	}
	if err := f.Close(); err != nil {
		return "", err
	}

	if err := env.Edit(path); err != nil {
		return "", err
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(contents), nil
}

// requiredLine re-prompts until the author enters a non-empty answer, giving up
// if stdin closes first so a piped or empty stdin fails rather than spinning.
func requiredLine(p *prompter, label string) (string, error) {
	for {
		s, err := p.line(label)
		if err != nil {
			return "", err
		}
		if s != "" {
			return s, nil
		}
		if p.eof {
			return "", fmt.Errorf("no %s entered", strings.TrimRight(strings.ToLower(label), ": "))
		}
	}
}

// parseTags splits a comma-separated answer into trimmed, non-empty tags.
func parseTags(line string) []string {
	var tags []string
	for _, part := range strings.Split(line, ",") {
		if tag := strings.TrimSpace(part); tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}

// slugify renders text as a default slug in custodian's shape: lowercase
// alphanumeric words joined by single hyphens, with no leading or trailing
// hyphen. It is only a starting point — the author confirms or overrides it.
func slugify(text string) string {
	var b strings.Builder
	pendingHyphen := false
	for _, r := range strings.ToLower(text) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			if pendingHyphen && b.Len() > 0 {
				b.WriteByte('-')
			}
			pendingHyphen = false
			b.WriteRune(r)
		} else {
			pendingHyphen = true
		}
	}
	return b.String()
}
