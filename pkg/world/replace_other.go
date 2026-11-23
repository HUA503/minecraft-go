//go:build !windows && !unix

package world

import "os"

func replaceFile(src, dst string) error {
	return os.Rename(src, dst)
}

func syncDirectory(string) error { return nil }

func syncDirectoryTree(string, string) error { return nil }

