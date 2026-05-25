# Implementation Plan: `howe update` Self-Update Command

## 1. The Core Problem: Replacing a Running Binary

On Unix systems (Linux, macOS, *BSD), a running executable is held open by the kernel via an **inode reference**, not a path reference. This means:

- You **can** `unlink` (remove) the file path of a running binary — the process keeps running because the inode is still referenced by the open file descriptor.
- You **can** create a new file at the same path — this creates a brand-new inode.
- The running process continues to use the old inode until it exits.

Therefore, the safe replacement sequence on Unix is:

```
1. Download new binary → /tmp/howe-<version>-<os>-<arch>
2. chmod +x /tmp/howe-<version>-<os>-<arch>
3. mv /usr/local/bin/howe /usr/local/bin/howe.old   # renames inode, keeps it open
4. mv /tmp/howe-<version>-<os>-<arch> /usr/local/bin/howe
5. /usr/local/bin/howe version                      # sanity check
6. rm /usr/local/bin/howe.old                       # frees old inode
```

On **Windows**, a running `.exe` holds a mandatory file lock, so steps 3–4 fail. The Windows workaround is:
1. Rename running binary: `howe.exe → howe.exe.old`
2. Write new binary to `howe.exe`
3. Use `MoveFileEx` with `MOVEFILE_DELAY_UNTIL_REBOOT` to schedule `.old` deletion.

Since Howe’s primary target is Debian/Linux, this plan focuses on the Unix flow with Windows noted as a deferred concern.

---

## 2. Permission Handling

The binary typically lives in a system directory (`/usr/local/bin`, `/usr/bin`, `/bin`). The user running `howe update` may not have write access.

### Options

| Approach | Pros | Cons |
|----------|------|------|
| **A. Detect permissions, bail with `sudo` hint** | Simple, explicit, no privilege escalation surprises | User must re-run manually |
| **B. Auto-re-exec via `sudo` if available** | One-shot UX | May prompt for password unexpectedly; fragile in scripts |
| **C. Write to `~/.local/bin` fallback** | No sudo needed | PATH may not include it; breaks system-wide installs |

**Recommendation: Approach A** — check writability of the current binary’s directory. If not writable, print a helpful message:

```
$ howe update
Update available: v0.3.1 → v0.3.2
Error: cannot write to /usr/local/bin. Re-run with:
    sudo howe update
```

If the binary is in a user-writable directory (e.g. `~/bin`), proceed automatically.

---

## 3. Release Detection & Download

### 3.1 Version Source of Truth

Use the GitHub API:
```
GET https://api.github.com/repos/gophertribe/howe/releases/latest
```

Response JSON contains:
- `tag_name` — e.g. `"v0.3.2"`
- `assets[]` — list of release binaries

### 3.2 Asset Naming Convention

Existing releases use: `howe-<GOOS>-<GOARCH>`

Map `runtime.GOOS` + `runtime.GOARCH` to asset names:

| GOOS | GOARCH | Asset Name |
|------|--------|------------|
| linux | amd64 | `howe-linux-amd64` |
| linux | arm64 | `howe-linux-arm64` |
| darwin | arm64 | `howe-darwin-arm64` |

### 3.3 Checksum Verification

The release already publishes `shasums`. Download `shasums` alongside the binary, verify the SHA256 hash before replacing anything.

### 3.4 HTTP Client Requirements

- Set a reasonable timeout (e.g. 30s)
- Follow redirects
- Respect `HTTP_PROXY` / `HTTPS_PROXY` environment variables (Go’s `http.DefaultClient` does this automatically)

---

## 4. Proposed CLI UX

```
$ howe update
Current version: v0.3.1
Latest version:  v0.3.2
Downloading howe-linux-amd64 ... done
Verifying checksum ... ok
Backing up /usr/local/bin/howe → /usr/local/bin/howe.old
Installing new version ... done
Cleaning up ... done
Updated to v0.3.2
```

Flags:
- `--dry-run` — check for update but do not download/replace
- `--force` / `-f` — reinstall even if versions match
- `--tag <tag>` — install a specific release instead of `latest`

---

## 5. Error Handling & Rollback

| Failure Point | Behaviour |
|---------------|-----------|
| Network failure / API unreachable | Print error, exit non-zero, leave system untouched |
| Asset not found for this OS/arch | Print error with detected platform, suggest opening an issue |
| Checksum mismatch | Delete downloaded file, print error, exit non-zero |
| `mv` fails (permissions) | Print `sudo` hint, exit non-zero |
| New binary fails `version` sanity check | Restore `.old` → original name, print rollback notice |

**Rollback mechanism:**
- Keep `.old` until the new binary successfully reports its version.
- If the sanity check fails, atomically move `.old` back to the original name.

---

## 6. Implementation Steps

### Step 1: Add `update` Cobra command
- File: `cmd/howe/update.go`
- Uses `version` variable already defined in `cmd/howe/version.go`

### Step 2: Create `internal/updater` package (or `pkg/updater`)
Responsibilities:
- `CheckLatest(ctx, owner, repo, currentVersion) (*ReleaseInfo, error)`
- `Download(ctx, url) (path string, cleanup func(), error)`
- `VerifyChecksum(binaryPath, checksumsURL, assetName) error`
- `Replace(binaryPath, newBinaryPath) (rollback func(), error)`

### Step 3: Platform-specific replacement logic
- `internal/updater/replace_unix.go` (`//go:build unix`)
  - `atomicReplace(target, source) error`
- `internal/updater/replace_windows.go` (`//go:build windows`)
  - Stub returning `ErrNotImplemented` or Windows-specific rename-then-replace logic

### Step 4: Wire into `cmd/howe`
- Register `update` subcommand in `rootCmd`
- Add `--dry-run` and `--tag` flags

### Step 5: Testing
- Unit tests for checksum verification using testdata
- Unit tests for version comparison (`v0.3.1` vs `v0.3.2`)
- Integration test: dry-run against real GitHub API (tagged, not run in CI unless mocked)

---

## 7. Open Questions

1. **Should we keep `.old` indefinitely or delete immediately after sanity check?**
   - *Recommendation:* Delete immediately on success; it’s the Unix norm and avoids clutter.

2. **Should the update command support downgrades via `--tag`?**
   - *Recommendation:* Yes, `--tag v0.3.1` should work. The checksum and replacement logic is identical.

3. **What if `howe` is running inside a container or as a snap/flatpak?**
   - *Recommendation:* Detect if the binary is read-only or in an immutable location, print a targeted error message.

4. **Rate limiting / authenticated API requests?**
   - *Recommendation:* Use unauthenticated API by default. GitHub rate-limits unauthenticated requests to 60/hr/IP, which is fine for a CLI tool. If users hit limits, we can add `GITHUB_TOKEN` support later.

---

## 8. Files to Create / Modify

| File | Action |
|------|--------|
| `cmd/howe/update.go` | New: Cobra command implementation |
| `internal/updater/updater.go` | New: Core orchestration logic |
| `internal/updater/github.go` | New: GitHub API client |
| `internal/updater/replace_unix.go` | New: Unix atomic replacement |
| `internal/updater/replace_windows.go` | New: Windows stub/deferred-delete |
| `cmd/howe/howe.go` | Modify: Register `update` subcommand |
| `README.md` | Modify: Document `howe update` |
| `spec/ROADMAP.md` | Modify: Mark update command as done |
