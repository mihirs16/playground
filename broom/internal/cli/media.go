package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/mihirs16/playground/broom/internal/apiclient"
	"github.com/mihirs16/playground/broom/internal/custodian"
	"github.com/spf13/cobra"
)

func newMediaCmd(env Env) *cobra.Command {
	cmd := &cobra.Command{Use: "media", Short: "Upload and manage media"}
	cmd.AddCommand(
		newMediaAddCmd(env),
		newMediaLsCmd(env),
		newMediaRmCmd(env),
	)
	return cmd
}

// newMediaAddCmd builds `media add <file> [--key k]`, the reserve → upload →
// confirm gesture. --key names the asset for a legible reference; omitted, the
// author accepts a custodian-minted random kebab key.
func newMediaAddCmd(env Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <file>",
		Short: "Upload a file and print its markdown reference",
		Args:  cobra.ExactArgs(1),
		RunE:  runMediaAdd(env),
	}
	cmd.Flags().String("key", "", "kebab-case key for a meaningful reference (default: a random one)")
	return cmd
}

func newMediaLsCmd(env Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ls [query]",
		Short: "List and search media",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runMediaLs(env),
	}
	return cmd
}

func newMediaRmCmd(env Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rm <key>",
		Short: "Delete a media record",
		Args:  cobra.ExactArgs(1),
		RunE:  runMediaRm(env),
	}
	cmd.Flags().Bool("force", false, "delete without the reference-scan confirmation")
	return cmd
}

// runMediaAdd runs custodian's reserve → upload → confirm flow. broom reserves a
// pending record (getting a presigned URL back), PUTs the file's bytes straight
// to that URL, then confirms so custodian HEADs S3 and flips the record to
// available. broom holds no AWS credential — it talks only to custodian and to
// the presigned URL. On success it prints and clipboard-copies the markdown
// reference custodian's public url gives, so the author pastes a readable link.
func runMediaAdd(env Env) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		client, _, err := env.requireClient()
		if err != nil {
			return err
		}
		path := args[0]
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		contentType := detectContentType(path, data)

		reserve := apiclient.MediaReserve{ContentType: contentType}
		if key, _ := cmd.Flags().GetString("key"); key != "" {
			reserve.Key = &key
		}

		reservation, err := client.ReserveMedia(cmd.Context(), reserve)
		if err != nil {
			if custodian.IsMediaKeyTaken(err) {
				return fmt.Errorf("key %q is already taken — choose another --key or reuse it with `broom media ls`", *reserve.Key)
			}
			return err
		}

		if err := putBytes(cmd.Context(), reservation.UploadUrl, contentType, data); err != nil {
			return err
		}

		confirmed, err := client.ConfirmMedia(cmd.Context(), reservation.Key)
		if err != nil {
			return err
		}

		reference := markdownReference(confirmed.Url)
		fmt.Fprintln(env.Stdout, reference)
		copyToClipboard(env, reference)
		return nil
	}
}

// runMediaLs lists and searches existing media so an asset can be reused rather
// than re-uploaded. An optional query narrows the listing to matches on the key.
func runMediaLs(env Env) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		client, _, err := env.requireClient()
		if err != nil {
			return err
		}
		var query string
		if len(args) == 1 {
			query = args[0]
		}
		list, err := client.ListMedia(cmd.Context(), query)
		if err != nil {
			return err
		}
		printMediaList(env, list)
		return nil
	}
}

// runMediaRm deletes a media record, but first scans the author's post bodies for
// references to its public url and warns before deleting, so a live post is not
// left pointing at an orphaned image. custodian does not parse bodies for urls;
// this scan is broom's courtesy. --force skips the scan and its confirmation.
func runMediaRm(env Env) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		client, _, err := env.requireClient()
		if err != nil {
			return err
		}
		key := args[0]
		force, _ := cmd.Flags().GetBool("force")

		if !force {
			record, err := client.GetMedia(cmd.Context(), key)
			if err != nil {
				return err
			}
			referencing, err := scanReferences(cmd.Context(), client, record.Url)
			if err != nil {
				return err
			}
			if len(referencing) > 0 {
				if !confirmDeleteReferenced(env, key, referencing) {
					fmt.Fprintf(env.Stdout, "Left %s in place.\n", key)
					return nil
				}
			}
		}

		if err := client.DeleteMedia(cmd.Context(), key); err != nil {
			return err
		}
		fmt.Fprintf(env.Stdout, "Deleted %s\n", key)
		return nil
	}
}

// scanReferences returns the slugs of the author's posts whose body contains the
// media url. It reads every post's body — the admin listing carries only
// summaries — since that is where a live reference to the asset would live.
func scanReferences(ctx context.Context, client *custodian.Client, url string) ([]string, error) {
	index, err := client.ListLogs(ctx, nil)
	if err != nil {
		return nil, err
	}
	var referencing []string
	for _, item := range index.Items {
		log, err := client.GetLog(ctx, item.Slug)
		if err != nil {
			return nil, err
		}
		if strings.Contains(log.Body, url) {
			referencing = append(referencing, item.Slug)
		}
	}
	return referencing, nil
}

// confirmDeleteReferenced warns that live posts reference the asset and asks
// whether to delete anyway. A missing or non-affirmative answer aborts, so the
// default is to keep the asset the posts still point at.
func confirmDeleteReferenced(env Env, key string, slugs []string) bool {
	fmt.Fprintf(env.Stderr, "%s is still referenced by %d post(s):\n", key, len(slugs))
	for _, slug := range slugs {
		fmt.Fprintf(env.Stderr, "  - %s\n", slug)
	}
	p := newPrompter(env)
	answer, err := p.line("Delete anyway? [y/N]: ")
	if err != nil {
		return false
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes"
}

// putBytes uploads data to the presigned URL with a straight HTTP PUT, sending
// the same content type the reservation was presigned for. This is the only leg
// of the flow that leaves broom without touching custodian, and it carries no
// AWS credential — the presigned URL is the whole authorisation.
func putBytes(ctx context.Context, url, contentType string, data []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	req.ContentLength = int64(len(data))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("uploading to the presigned URL: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("presigned upload failed: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

// detectContentType names the file's media type, preferring the extension's
// registered type and falling back to sniffing the bytes, so custodian presigns
// — and S3 stores — the upload under a real content type.
func detectContentType(path string, data []byte) string {
	if ext := filepath.Ext(path); ext != "" {
		if ct := mime.TypeByExtension(ext); ct != "" {
			return ct
		}
	}
	return http.DetectContentType(data)
}

// markdownReference wraps a media url as the legible markdown image the author
// pastes into a post body.
func markdownReference(url string) string {
	return fmt.Sprintf("![](%s)", url)
}

// copyToClipboard copies the reference and notes the outcome. A failing or
// unconfigured clipboard is not fatal — the reference is already on stdout — so
// the note goes to stderr and the command still succeeds.
func copyToClipboard(env Env, reference string) {
	if env.Copy == nil {
		return
	}
	if err := env.Copy(reference); err != nil {
		fmt.Fprintf(env.Stderr, "(could not copy to clipboard: %v)\n", err)
		return
	}
	fmt.Fprintln(env.Stderr, "Copied to clipboard.")
}

// printMediaList renders the media index one line per asset — state, key, url —
// so an asset can be recognised and its reference reused at a glance.
func printMediaList(env Env, list *apiclient.MediaList) {
	if len(list.Items) == 0 {
		fmt.Fprintln(env.Stdout, "No media.")
		return
	}
	for _, item := range list.Items {
		fmt.Fprintf(env.Stdout, "%-10s  %-24s  %s\n", item.State, item.Key, item.Url)
	}
}
