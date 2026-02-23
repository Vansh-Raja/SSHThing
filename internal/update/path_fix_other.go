//go:build !windows

package update

import (
	"context"
	"fmt"
)

func fixPathConflicts(_ context.Context, _ string) (PathHealth, error) {
	return PathHealth{Healthy: true, Message: "PATH fix is currently available on Windows only"}, fmt.Errorf("path fix unsupported on this platform")
}
