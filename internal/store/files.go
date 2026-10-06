package store

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// UpdateFile serializes MCPDeck writers using the resolved target path. Editors
// do not honor this lock, so their changes are also checked just before writing.
// An external writer can still race between that check and the atomic rename.
func UpdateFile(path, backupSuffix string, change func([]byte, bool) ([]byte, error)) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	if resolved, e := filepath.EvalSymlinks(path); e == nil {
		path = resolved
	} else {
		if _, e := os.Lstat(path); !os.IsNotExist(e) {
			return fmt.Errorf("cannot resolve configuration target")
		}
		parent, e := filepath.EvalSymlinks(filepath.Dir(path))
		if e != nil {
			return e
		}
		path = filepath.Join(parent, filepath.Base(path))
	}
	lock := path + ".mcpdeck-lock"
	deadline := time.Now().Add(3 * time.Second)
	for {
		if e := os.Mkdir(lock, 0700); e == nil {
			break
		} else if !errors.Is(e, os.ErrExist) {
			return e
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("configuration is locked; retry after other MCPDeck writers finish")
		}
		time.Sleep(25 * time.Millisecond)
	}
	defer os.Remove(lock)
	read := func() ([]byte, bool, error) {
		b, e := os.ReadFile(path)
		if os.IsNotExist(e) {
			return nil, false, nil
		}
		return b, e == nil, e
	}
	before, existed, err := read()
	if err != nil {
		return err
	}
	after, err := change(before, existed)
	if err != nil {
		return err
	}
	if existed && bytes.Equal(before, after) {
		return nil
	}
	unchanged := func() error {
		current, exists, err := read()
		if err != nil || exists != existed || !bytes.Equal(before, current) {
			return fmt.Errorf("agent settings changed during update; retry with the latest settings")
		}
		return nil
	}
	if err = unchanged(); err != nil {
		return err
	}
	if existed && backupSuffix != "" {
		if err = AtomicWrite(path+backupSuffix, before); err != nil {
			return fmt.Errorf("configuration backup failed: %w", err)
		}
	}
	if err = unchanged(); err != nil {
		return err
	}
	return AtomicWrite(path, after)
}
