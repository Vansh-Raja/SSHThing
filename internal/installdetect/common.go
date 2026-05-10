package installdetect

import (
	"context"
	"os/exec"
	"strings"
)

// runCmdOutput runs a command with the given context and returns trimmed
// combined output. Empty string + nil err means "command exited cleanly
// but produced no output" — callers should check the error first.
func runCmdOutput(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func hasTool(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
