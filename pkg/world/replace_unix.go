//go:build unix

package world

import (
	"errors"
	"os"
	"path/filepath"
)

func syncDirectoryTree(path string) error {
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
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
	}
}

func replaceFile(src, dst string) error {
	if err := os.Rename(src, dst); err != nil {
		return err
	}
	return syncDirectoryTree(filepath.Dir(dst))
}

func syncDirectory(path string) error {
	return syncDirectoryTree(path)
}

