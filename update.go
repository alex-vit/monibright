package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const githubReleaseURL = "https://api.github.com/repos/alex-vit/monibright/releases/latest"

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Assets  []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func autoUpdate() {
	latestVer, url, err := checkForUpdate()
	if err != nil {
		log.Printf("update check failed: %v", err)
		return
	}
	if url == "" {
		log.Printf("no update available (current=%s)", displayVersion())
		return
	}
	log.Printf("update available: v%s", latestVer)
	tmpPath, err := downloadUpdate(url)
	if err != nil {
		log.Printf("update download failed: %v", err)
		return
	}
	if err := applyUpdate(tmpPath); err != nil {
		log.Printf("update apply failed: %v", err)
	}
}

// cleanOldBinary removes a leftover .old file from a previous update.
func cleanOldBinary() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	old := exe + ".old"
	if err := os.Remove(old); err == nil {
		log.Printf("removed old binary: %s", old)
	}
}

// checkForUpdate queries the GitHub releases API and returns the latest
// version and asset download URL if it's newer than the current version.
func checkForUpdate() (latestVer, downloadURL string, err error) {
	req, err := http.NewRequest(http.MethodGet, githubReleaseURL, nil) //nolint:noctx
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", "", err
	}

	latestVer = strings.TrimPrefix(rel.TagName, "v")
	if !isNewer(latestVer, version) {
		return "", "", nil // up to date
	}

	for _, a := range rel.Assets {
		if strings.EqualFold(a.Name, "monibright.exe") {
			return latestVer, a.BrowserDownloadURL, nil
		}
	}
	return "", "", fmt.Errorf("no monibright.exe asset in release %s", rel.TagName)
}

// downloadUpdate downloads the new binary to a .tmp file next to the running exe.
func downloadUpdate(url string) (tmpPath string, err error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodGet, url, nil) //nolint:noctx
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned %d", resp.StatusCode)
	}

	tmpPath = exe + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}

	log.Printf("downloaded update to %s", tmpPath)
	return tmpPath, nil
}

// applyUpdate replaces the running exe with the downloaded update.
// The new version takes effect on next launch (reboot, autostart, or manual).
func applyUpdate(tmpPath string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	old := exe + ".old"

	// Windows allows renaming a running exe but not overwriting it.
	if err := os.Rename(exe, old); err != nil {
		return fmt.Errorf("rename current to .old: %w", err)
	}
	if err := os.Rename(tmpPath, exe); err != nil {
		_ = os.Rename(old, exe)
		return fmt.Errorf("rename .tmp to exe: %w", err)
	}

	log.Printf("applied update, new version ready on next launch")
	return nil
}

// isNewer reports whether latest is a higher semver than current.
// Versions are expected as "X.Y.Z" (no "v" prefix).
func isNewer(latest, current string) bool {
	if current == "" || current == "dev" {
		return false // dev builds don't auto-update
	}
	lp := parseSemver(latest)
	cp := parseSemver(current)
	if lp == nil || cp == nil {
		return false
	}
	for i := range 3 {
		if lp[i] > cp[i] {
			return true
		}
		if lp[i] < cp[i] {
			return false
		}
	}
	return false
}

func parseSemver(s string) []int {
	parts := strings.SplitN(s, ".", 3)
	if len(parts) != 3 {
		return nil
	}
	nums := make([]int, 3)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil
		}
		nums[i] = n
	}
	return nums
}
