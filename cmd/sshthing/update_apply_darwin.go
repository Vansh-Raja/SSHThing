//go:build darwin

package main

import (
	"context"
	"fmt"

	"github.com/Vansh-Raja/SSHThing/internal/installdetect"
	"github.com/Vansh-Raja/SSHThing/internal/update"
)

func applyGUI(ctx context.Context, p planEntry) error {
	if p.install.Channel != installdetect.ChannelDMG {
		return fmt.Errorf("unsupported macOS GUI channel %q", p.install.Channel)
	}
	return update.ApplyGUIDarwin(ctx, p.asset, p.checksums, p.install.Path)
}
