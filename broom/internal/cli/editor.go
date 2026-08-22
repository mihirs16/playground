package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// EditorFunc opens path in the author's editor and returns when it closes. It is
// the seam the authoring commands write through: production launches $EDITOR as a
// subprocess, tests substitute a scripted fake that writes known content or exits
// without touching the file.
type EditorFunc func(path string) error

// RealEditor launches the author's editor on a file, inheriting the process's
// terminal so the author edits in place. It honours VISUAL then EDITOR, falling
// back to a per-platform default, and passes any arguments baked into the setting
// (e.g. `code --wait`) through to the command.
func RealEditor(getenv func(string) string) EditorFunc {
	return func(path string) error {
		setting := firstNonEmpty(getenv("VISUAL"), getenv("EDITOR"), defaultEditor())
		parts := strings.Fields(setting)
		if len(parts) == 0 {
			return fmt.Errorf("no editor configured — set $EDITOR")
		}

		args := append(parts[1:], path)
		cmd := exec.Command(parts[0], args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
}

func defaultEditor() string {
	if runtime.GOOS == "windows" {
		return "notepad"
	}
	return "vi"
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
