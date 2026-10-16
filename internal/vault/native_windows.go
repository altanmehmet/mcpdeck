package vault

import (
	"fmt"
	"golang.org/x/sys/windows"
	"runtime"
	"unsafe"
)

var credentialDLL = windows.NewLazySystemDLL("advapi32.dll")
var credWrite = credentialDLL.NewProc("CredWriteW")
var credRead = credentialDLL.NewProc("CredReadW")
var credFree = credentialDLL.NewProc("CredFree")

// Layout of CREDENTIALW from wincred.h; pointer alignment is platform-native.
type credential struct {
	Flags, Type             uint32
	TargetName, Comment     *uint16
	LastWritten             windows.Filetime
	BlobSize                uint32
	Blob                    *byte
	Persist, AttributeCount uint32
	Attributes              unsafe.Pointer
	TargetAlias, UserName   *uint16
}

func readSecret(id string) ([]byte, error) {
	target, _ := windows.UTF16PtrFromString("MCPDeck/" + id)
	var result *credential
	ok, _, _ := credRead.Call(uintptr(unsafe.Pointer(target)), 1, 0, uintptr(unsafe.Pointer(&result)))
	runtime.KeepAlive(target)
	if ok == 0 || result == nil {
		return nil, fmt.Errorf("Windows credential unavailable")
	}
	defer credFree.Call(uintptr(unsafe.Pointer(result)))
	if result.BlobSize == 0 || result.BlobSize > MaxSecret || result.Blob == nil {
		return nil, fmt.Errorf("invalid Windows credential")
	}
	return append([]byte(nil), unsafe.Slice(result.Blob, int(result.BlobSize))...), nil
}
func writeSecret(id string, value []byte) error {
	target, _ := windows.UTF16PtrFromString("MCPDeck/" + id)
	c := credential{Type: 1, TargetName: target, BlobSize: uint32(len(value)), Blob: &value[0], Persist: 2}
	ok, _, _ := credWrite.Call(uintptr(unsafe.Pointer(&c)), 0)
	runtime.KeepAlive(c)
	runtime.KeepAlive(value)
	runtime.KeepAlive(target)
	if ok == 0 {
		return fmt.Errorf("cannot save credential in Windows Credential Manager; no plaintext fallback")
	}
	return nil
}

func deleteSecret(id string) error {
	target, _ := windows.UTF16PtrFromString("MCPDeck/" + id)
	ok, _, _ := credentialDLL.NewProc("CredDeleteW").Call(uintptr(unsafe.Pointer(target)), 1, 0)
	runtime.KeepAlive(target)
	if ok == 0 {
		return fmt.Errorf("cannot remove Windows credential")
	}
	return nil
}
