package store

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"testing"
)

func TestUpdateFilePreservesExternalEdits(t *testing.T) {
	for _, absent := range []bool{false, true} {
		t.Run(fmt.Sprint(absent), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "agent.json")
			if !absent {
				if err := os.WriteFile(path, []byte("before"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			err := UpdateFile(path, ".mcpdeck-backup", func(before []byte, exists bool) ([]byte, error) {
				if err := os.WriteFile(path, []byte("external edit"), 0600); err != nil {
					return nil, err
				}
				return []byte("our edit"), nil
			})
			if err == nil {
				t.Fatal("concurrent edit accepted")
			}
			b, err := os.ReadFile(path)
			if err != nil || string(b) != "external edit" {
				t.Fatal("external edit lost", err)
			}
		})
	}
}

func TestUpdateFileSerializesWriters(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent")
	if err := os.WriteFile(path, []byte("0"), 0600); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- UpdateFile(path, "", func(before []byte, exists bool) ([]byte, error) {
				n, err := strconv.Atoi(string(before))
				return []byte(strconv.Itoa(n + 1)), err
			})
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	b, err := os.ReadFile(path)
	if err != nil || string(b) != "8" {
		t.Fatal("updates lost", err)
	}
}

func TestUpdateFileKeepsSymlinkAndNoopBackup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions")
	}
	dir := t.TempDir()
	path, link := filepath.Join(dir, "agent"), filepath.Join(dir, "link")
	if err := os.WriteFile(path, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := UpdateFile(link, ".mcpdeck-backup", func([]byte, bool) ([]byte, error) { return []byte("updated"), nil }); err != nil {
			t.Fatal(err)
		}
	}
	info, err := os.Lstat(link)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("symlink replaced", err)
	}
	b, err := os.ReadFile(path + ".mcpdeck-backup")
	if err != nil || string(b) != "original" {
		t.Fatal("noop overwrote backup", err)
	}
}
