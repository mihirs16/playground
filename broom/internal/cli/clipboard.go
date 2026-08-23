package cli

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// ClipboardFunc copies text onto the system clipboard. It is the seam the media
// workflow writes its markdown reference through: production shells out to the
// platform's clipboard tool, tests substitute a fake that captures the string.
type ClipboardFunc func(text string) error

// RealClipboard copies text using the platform's clipboard command — `clip` on
// Windows, `pbcopy` on macOS, and a Wayland/X11 tool on Linux. The text is fed on
// stdin so no shell quoting is involved.
func RealClipboard() ClipboardFunc {
	return func(text string) error {
		name, args := clipboardCommand()
		if name == "" {
			return fmt.Errorf("no clipboard tool available on %s", runtime.GOOS)
		}
		cmd := exec.Command(name, args...)
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}
}

// clipboardCommand picks the clipboard writer for the current platform, or an
// empty name when none is known.
func clipboardCommand() (string, []string) {
	switch runtime.GOOS {
	case "windows":
		return "clip", nil
	case "darwin":
		return "pbcopy", nil
	default:
		if path, err := exec.LookPath("wl-copy"); err == nil {
			return path, nil
		}
		if path, err := exec.LookPath("xclip"); err == nil {
			return path, []string{"-selection", "clipboard"}
		}
		return "", nil
	}
}
