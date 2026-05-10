//go:build !darwin && !linux && !windows

package main

import (
	"context"
	"fmt"
)

func applyGUI(_ context.Context, _ planEntry) error {
	return fmt.Errorf("GUI updates are not supported on this platform; build from source")
}
