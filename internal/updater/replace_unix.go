//go:build unix

package updater

import (
	"fmt"
	"os"
)

// atomicReplace moves source to target by first renaming target to backupPath,
// then renaming source to target. If the final rename fails it attempts to
// restore the backup.
func atomicReplace(target, source, backupPath string) error {
	if err := os.Rename(target, backupPath); err != nil {
		return fmt.Errorf("failed to backup current binary: %w", err)
	}
	if err := os.Rename(source, target); err != nil {
		// Attempt rollback.
		_ = os.Rename(backupPath, target)
		return fmt.Errorf("failed to install new binary: %w", err)
	}
	return nil
}
