//go:build !windows

package testutil

import (
	"os"
	"testing"
)

func AssertPrivateFile(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("unsafe file permissions", err)
	}
}
