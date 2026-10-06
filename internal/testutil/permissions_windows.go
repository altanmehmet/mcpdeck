package testutil

import (
	"golang.org/x/sys/windows"
	"testing"
	"unsafe"
)

func AssertPrivateFile(t *testing.T, path string) {
	t.Helper()
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	control, _, err := sd.Control()
	if err != nil || control&windows.SE_DACL_PROTECTED == 0 {
		t.Fatal("unprotected DACL", err)
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	system, err := windows.StringToSid("S-1-5-18")
	if err != nil {
		t.Fatal(err)
	}
	dacl, _, err := sd.DACL()
	if err != nil || dacl == nil || dacl.AceCount != 2 {
		t.Fatal("unexpected DACL entries", err)
	}
	for i := uint32(0); i < uint32(dacl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err = windows.GetAce(dacl, i, &ace); err != nil {
			t.Fatal(err)
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
			t.Fatal("unexpected DACL entry type")
		}
		// GetAce returns a validated ACCESS_ALLOWED_ACE with the variable-length
		// SID beginning at SidStart. Compare SIDs, since SDDL uses aliases for
		// built-in administrator accounts rather than their numeric SID string.
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		if !sid.Equals(user.User.Sid) && !sid.Equals(system) {
			t.Fatal("unsafe DACL principal")
		}
	}
}
