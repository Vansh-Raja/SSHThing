//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/Vansh-Raja/SSHThing/internal/installdetect"
	"github.com/Vansh-Raja/SSHThing/internal/update"
)

func applyGUI(ctx context.Context, p planEntry) error {
	if p.install.Channel != installdetect.ChannelAppImage {
		return fmt.Errorf("unsupported Linux GUI channel %q", p.install.Channel)
	}
	return update.ApplyGUILinux(ctx, p.asset, p.checksums, p.install.Path)
}
