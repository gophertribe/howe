package updater

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Updater handles checking for and applying updates.
type Updater struct {
	Owner          string
	Repo           string
	CurrentVersion string
}

// UpdateInfo describes an available update.
type UpdateInfo struct {
	Current string
	Latest  string
	Asset   struct {
		Name string
		URL  string
	}
	ChecksumAsset struct {
		Name string
		URL  string
	}
}

// HasUpdate reports whether Latest is newer than Current.
// It always returns true when the user explicitly requested a tag (downgrades supported).
func (u *UpdateInfo) HasUpdate(requestedTag string) bool {
	if requestedTag != "" && requestedTag != "latest" {
		return u.Current != requestedTag
	}
	return compareVersions(u.Current, u.Latest) < 0
}

// Check queries GitHub for the requested release and determines whether an update is available.
func (u *Updater) Check(ctx context.Context, requestedTag string) (*UpdateInfo, error) {
	tag := requestedTag
	if tag == "" {
		tag = "latest"
	}

	release, err := FetchRelease(ctx, u.Owner, u.Repo, tag)
	if err != nil {
		return nil, err
	}

	info := &UpdateInfo{
		Current: stripV(u.CurrentVersion),
		Latest:  stripV(release.TagName),
	}
	assetName := assetNameForRelease(info.Latest)

	for _, a := range release.Assets {
		if a.Name == assetName {
			info.Asset.Name = a.Name
			info.Asset.URL = a.BrowserDownloadURL
		}
		if a.Name == "checksums.txt" {
			info.ChecksumAsset.Name = a.Name
			info.ChecksumAsset.URL = a.BrowserDownloadURL
		}
	}

	if info.Asset.URL == "" {
		return nil, fmt.Errorf("no release asset found for %s/%s (looking for %q)", runtime.GOOS, runtime.GOARCH, assetName)
	}

	return info, nil
}

// Update performs the full update flow.
func (u *Updater) Update(ctx context.Context, requestedTag string, dryRun bool) error {
	info, err := u.Check(ctx, requestedTag)
	if err != nil {
		return err
	}

	if !info.HasUpdate(requestedTag) {
		fmt.Printf("Already up to date: %s\n", info.Current)
		return nil
	}

	if dryRun {
		fmt.Printf("Update available: %s → %s\n", info.Current, info.Latest)
		fmt.Printf("Asset: %s\n", info.Asset.Name)
		return nil
	}

	fmt.Printf("Update available: %s → %s\n", info.Current, info.Latest)

	// Determine current binary path.
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot locate current binary: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("cannot resolve current binary path: %w", err)
	}

	// Ensure we can write to the directory.
	installDir := filepath.Dir(execPath)
	if err := checkWritable(installDir); err != nil {
		return fmt.Errorf("cannot write to %s: %w\n\nRe-run with:\n    sudo howe update", installDir, err)
	}

	// Download new binary.
	fmt.Printf("Downloading %s ... ", info.Asset.Name)
	tmpPath, err := DownloadAsset(ctx, info.Asset.URL)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmpPath) }()
	fmt.Println("done")

	// Verify checksum if available.
	if info.ChecksumAsset.URL != "" {
		fmt.Print("Verifying checksum ... ")
		if err := verifyChecksum(ctx, tmpPath, info.ChecksumAsset.URL, info.Asset.Name); err != nil {
			return err
		}
		fmt.Println("ok")
	}

	// Make new binary executable.
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return fmt.Errorf("failed to make new binary executable: %w", err)
	}

	// Sanity-check the new binary.
	if err := sanityCheck(ctx, tmpPath); err != nil {
		return fmt.Errorf("new binary failed sanity check: %w", err)
	}

	// Atomic replacement.
	oldPath := execPath + ".old"
	fmt.Printf("Installing %s → %s ... ", info.Latest, execPath)
	if err := atomicReplace(execPath, tmpPath, oldPath); err != nil {
		return fmt.Errorf("failed to install new binary: %w", err)
	}
	fmt.Println("done")

	// Clean up backup on success.
	_ = os.Remove(oldPath)
	fmt.Printf("Updated to %s\n", info.Latest)
	return nil
}

func verifyChecksum(ctx context.Context, binaryPath, checksumsURL, assetName string) error {
	checksumsPath, err := DownloadAsset(ctx, checksumsURL)
	if err != nil {
		return fmt.Errorf("failed to download checksums: %w", err)
	}
	defer func() { _ = os.Remove(checksumsPath) }()

	expectedHash, err := findChecksum(checksumsPath, assetName)
	if err != nil {
		return err
	}

	f, err := os.Open(binaryPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	actualHash := hex.EncodeToString(h.Sum(nil))

	if !strings.EqualFold(expectedHash, actualHash) {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actualHash)
	}
	return nil
}

func findChecksum(checksumsPath, assetName string) (string, error) {
	f, err := os.Open(checksumsPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			// Handle both "HASH  filename" and "HASH *filename" formats.
			name := strings.TrimPrefix(fields[1], "*")
			if name == assetName {
				return fields[0], nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("checksum for %q not found in checksums file", assetName)
}

func assetNameForRelease(version string) string {
	osName := runtime.GOOS
	if len(osName) > 0 {
		osName = strings.ToUpper(osName[:1]) + osName[1:]
	}
	archName := runtime.GOARCH
	if archName == "amd64" {
		archName = "x86_64"
	}
	return fmt.Sprintf("howe_%s_%s_%s", version, osName, archName)
}

func checkWritable(dir string) error {
	// Try creating a temporary file in the target directory.
	f, err := os.CreateTemp(dir, ".howe-write-test-*")
	if err != nil {
		return err
	}
	_ = f.Close()
	_ = os.Remove(f.Name())
	return nil
}

func sanityCheck(ctx context.Context, path string) error {
	cmd := exec.CommandContext(ctx, path, "version")
	out, err := cmd.Output()
	if err != nil {
		return err
	}
	if !strings.Contains(string(out), "howe version") {
		return fmt.Errorf("unexpected version output: %s", out)
	}
	return nil
}

// stripV removes a leading 'v' or 'V' from a version string.
func stripV(v string) string {
	return strings.TrimPrefix(strings.TrimPrefix(v, "v"), "V")
}

// compareVersions compares two "x.y.z" semver strings (without leading v).
// Returns -1, 0, or 1.
func compareVersions(a, b string) int {
	pa := parseVersion(a)
	pb := parseVersion(b)
	for i := 0; i < len(pa) && i < len(pb); i++ {
		if pa[i] < pb[i] {
			return -1
		}
		if pa[i] > pb[i] {
			return 1
		}
	}
	if len(pa) < len(pb) {
		return -1
	}
	if len(pa) > len(pb) {
		return 1
	}
	return 0
}

func parseVersion(v string) []int {
	parts := strings.Split(v, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		// Handle prerelease suffixes like "1-beta" by stripping non-numeric tail.
		idx := 0
		for idx < len(p) && p[idx] >= '0' && p[idx] <= '9' {
			idx++
		}
		n, _ := strconv.Atoi(p[:idx])
		out = append(out, n)
	}
	return out
}
