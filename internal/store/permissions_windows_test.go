package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

func TestWindowsPrivateFileAndReplacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret.json")
	for _, value := range []string{"before", "after"} {
		if err := AtomicWrite(path, []byte(value)); err != nil {
			t.Fatal(err)
		}
		sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
		if err != nil {
			t.Fatal(err)
		}
		control, _, err := sd.Control()
		if err != nil || control&windows.SE_DACL_PROTECTED == 0 {
			t.Fatal("DACL inheritance was not disabled", err)
		}
		if strings.Contains(sd.String(), ";;;WD)") || strings.Contains(sd.String(), ";;;BU)") {
			t.Fatal("broad file access")
		}
		content, err := os.ReadFile(path)
		if err != nil || string(content) != value {
			t.Fatal("replacement failed", err)
		}
	}
}

func TestWindowsLoadRepairsBroadDACL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "deck.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"servers":{},"profiles":{}}`), 0600); err != nil {
		t.Fatal(err)
	}
	sd, err := windows.SecurityDescriptorFromString("D:P(A;;FA;;;WD)")
	if err != nil {
		t.Fatal(err)
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		t.Fatal(err)
	}
	if err = windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, dacl, nil); err != nil {
		t.Fatal(err)
	}
	if _, err = (Store{Path: path}).Load(); err != nil {
		t.Fatal(err)
	}
	secured, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil || strings.Contains(secured.String(), ";;;WD)") {
		t.Fatal("broad access was not removed", err)
	}
}
