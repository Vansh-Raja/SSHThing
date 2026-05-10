//go:build windows

package main

import (
	"context"
	"fmt"

	"github.com/Vansh-Raja/SSHThing/internal/installdetect"
	"github.com/Vansh-Raja/SSHThing/internal/update"
)

func applyGUI(ctx context.Context, p planEntry) error {
	if p.install.Channel != installdetect.ChannelNSIS {
		return fmt.Errorf("unsupported Windows GUI channel %q", p.install.Channel)
	}
	return update.ApplyGUIWindows(ctx, p.asset, p.checksums, p.install.Path)
}
