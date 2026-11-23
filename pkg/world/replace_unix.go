//go:build unix

package world

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func syncDirectoryTree(path, stopAt string) error {
	stopAt = filepath.Clean(stopAt)
	for current := filepath.Clean(path); ; current = filepath.Dir(current) {
		dir, err := os.Open(current)
		if err != nil {
			return err
		}
		syncErr := dir.Sync()
		closeErr := dir.Close()
		if err := errors.Join(syncErr, closeErr); err != nil {
			return err
		}
		if current == stopAt {
			return nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return fmt.Errorf("directory sync stop path %q is not an ancestor of %q", stopAt, path)
		}
	}
}

func replaceFile(src, dst string) error {
	if err := os.Rename(src, dst); err != nil {
		return err
	}
	return syncDirectoryTree(filepath.Dir(dst), filepath.Dir(dst))
}

func syncDirectory(path string) error {
	return syncDirectoryTree(path, path)
}

