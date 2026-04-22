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

var githubAPIBaseURL = "https://api.github.com"

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

func fetchReleaseForChannel(ctx context.Context, channel ReleaseChannel, etag string) (*githubRelease, string, bool, error) {
	switch channel {
	case ReleaseChannelBeta:
		return fetchBetaRelease(ctx, etag)
	default:
		return fetchStableRelease(ctx, etag)
	}
}

func fetchStableRelease(ctx context.Context, etag string) (*githubRelease, string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/repos/%s/%s/releases/latest", githubAPIBaseURL, githubOwner, githubRepo), nil)
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

func fetchBetaRelease(ctx context.Context, etag string) (*githubRelease, string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/repos/%s/%s/releases?per_page=30", githubAPIBaseURL, githubOwner, githubRepo), nil)
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
		return nil, "", false, fmt.Errorf("github releases request failed: %s (%s)", resp.Status, string(body))
	}

	var releases []githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, "", false, err
	}

	selected, err := selectNewestReleaseForChannel(releases, ReleaseChannelBeta)
	if err != nil {
		return nil, "", false, err
	}
	return selected, resp.Header.Get("ETag"), false, nil
}

func selectNewestReleaseForChannel(releases []githubRelease, channel ReleaseChannel) (*githubRelease, error) {
	var selected *githubRelease
	for i := range releases {
		rel := releases[i]
		if rel.Draft {
			continue
		}
		if channel == ReleaseChannelStable && rel.Prerelease {
			continue
		}
		if resolveReleaseAsset(rel.Assets).URL == "" {
			continue
		}
		if _, err := parseSemverVersion(rel.TagName); err != nil {
			continue
		}
		if selected == nil || compareVersions(selected.TagName, rel.TagName) < 0 {
			selected = &rel
		}
	}
	if selected == nil {
		if channel == ReleaseChannelBeta {
			return nil, fmt.Errorf("no beta or stable release with a matching platform asset found")
		}
		return nil, fmt.Errorf("no stable release with a matching platform asset found")
	}
	return selected, nil
}

func findAsset(assets []githubReleaseAsset, name string) AssetInfo {
	for _, a := range assets {
		if a.Name == name {
			return AssetInfo{Name: a.Name, URL: a.URL}
		}
	}
	return AssetInfo{}
}
