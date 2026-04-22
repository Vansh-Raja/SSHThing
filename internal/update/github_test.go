package update

import "testing"

func releaseWithAssets(tag string, prerelease bool) githubRelease {
	return githubRelease{
		TagName:    tag,
		Prerelease: prerelease,
		Draft:      false,
		Assets: []githubReleaseAsset{
			{Name: "sshthing-windows-amd64.zip", URL: "https://example.com/sshthing-windows-amd64.zip"},
			{Name: "sshthing-setup-windows-amd64.exe", URL: "https://example.com/sshthing-setup-windows-amd64.exe"},
			{Name: "sshthing-macos-amd64.zip", URL: "https://example.com/sshthing-macos-amd64.zip"},
			{Name: "sshthing-macos-arm64.zip", URL: "https://example.com/sshthing-macos-arm64.zip"},
			{Name: "sshthing-linux-amd64.tar.gz", URL: "https://example.com/sshthing-linux-amd64.tar.gz"},
			{Name: "sshthing-linux-arm64.tar.gz", URL: "https://example.com/sshthing-linux-arm64.tar.gz"},
			{Name: "SHA256SUMS", URL: "https://example.com/SHA256SUMS"},
		},
	}
}

func TestSelectNewestReleaseForChannelStableIgnoresPrereleases(t *testing.T) {
	selected, err := selectNewestReleaseForChannel([]githubRelease{
		releaseWithAssets("v0.10.0-beta.1", true),
		releaseWithAssets("v0.9.9", false),
	}, ReleaseChannelStable)
	if err != nil {
		t.Fatalf("selectNewestReleaseForChannel stable: %v", err)
	}
	if selected.TagName != "v0.9.9" {
		t.Fatalf("expected stable release v0.9.9, got %q", selected.TagName)
	}
}

func TestSelectNewestReleaseForChannelBetaIncludesStableAndPrerelease(t *testing.T) {
	selected, err := selectNewestReleaseForChannel([]githubRelease{
		releaseWithAssets("v0.10.0-beta.2", true),
		releaseWithAssets("v0.10.0", false),
		releaseWithAssets("v0.9.9", false),
	}, ReleaseChannelBeta)
	if err != nil {
		t.Fatalf("selectNewestReleaseForChannel beta: %v", err)
	}
	if selected.TagName != "v0.10.0" {
		t.Fatalf("expected newest release v0.10.0, got %q", selected.TagName)
	}
}
