//go:build windows

package updater

import (
	"errors"
	"fmt"
	"os"
)

var ErrWindowsUpdate = errors.New("self-update is not yet supported on Windows")

func atomicReplace(target, source, backupPath string) error {
	// Windows locks running executables. A full implementation would:
	// 1. Move running exe to backupPath (os.Rename succeeds because it's the same dir).
	// 2. Move source to target.
	// 3. Schedule backupPath deletion on next reboot via MoveFileEx.
	// For now, bail out cleanly.
	_ = target
	_ = source
	_ = backupPath
	return fmt.Errorf("%w", ErrWindowsUpdate)
}
