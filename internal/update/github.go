package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	githubOwner = "Vansh-Raja"
	githubRepo  = "SSHThing"
)

type githubReleaseAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

type githubRelease struct {
	TagName    string               `json:"tag_name"`
	HTMLURL    string               `json:"html_url"`
	Prerelease bool                 `json:"prerelease"`
	Draft      bool                 `json:"draft"`
	Assets     []githubReleaseAsset `json:"assets"`
}

func fetchLatestRelease(ctx context.Context, etag string) (*githubRelease, string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", githubOwner, githubRepo), nil)
	if err != nil {
		return nil, "", false, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "sshthing-updater")
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return nil, resp.Header.Get("ETag"), true, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, "", false, fmt.Errorf("github latest release request failed: %s (%s)", resp.Status, string(body))
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, "", false, err
	}
	if rel.Draft || rel.Prerelease {
		return nil, "", false, fmt.Errorf("latest release is not a stable release")
	}
	return &rel, resp.Header.Get("ETag"), false, nil
}

func findAsset(assets []githubReleaseAsset, name string) AssetInfo {
	for _, a := range assets {
		if a.Name == name {
			return AssetInfo{Name: a.Name, URL: a.URL}
		}
	}
	return AssetInfo{}
}
