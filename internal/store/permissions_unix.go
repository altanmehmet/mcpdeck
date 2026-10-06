//go:build !windows

package store

import "os"

func ensurePrivateFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Mode().Perm()&0077 != 0 {
		return os.Chmod(path, 0600)
	}
	return nil
}
