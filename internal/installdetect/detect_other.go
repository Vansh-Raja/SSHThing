//go:build !darwin && !linux && !windows

package installdetect

import "context"

// detectPlatform stub for unsupported targets (FreeBSD, etc.). The CLI
// binary may still be runnable from source there, but `sshthing update`
// won't try to manage it; users build from source.
func detectPlatform(_ context.Context) []Install {
	return nil
}
