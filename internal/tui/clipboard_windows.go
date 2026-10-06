package tui

import (
	"encoding/binary"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	clipboardUser32        = windows.NewLazySystemDLL("user32.dll")
	clipboardKernel32      = windows.NewLazySystemDLL("kernel32.dll")
	clipboardCreateWindow  = clipboardUser32.NewProc("CreateWindowExW")
	clipboardDestroyWindow = clipboardUser32.NewProc("DestroyWindow")
	clipboardOpen          = clipboardUser32.NewProc("OpenClipboard")
	clipboardClose         = clipboardUser32.NewProc("CloseClipboard")
	clipboardEmpty         = clipboardUser32.NewProc("EmptyClipboard")
	clipboardSet           = clipboardUser32.NewProc("SetClipboardData")
	clipboardAlloc         = clipboardKernel32.NewProc("GlobalAlloc")
	clipboardLock          = clipboardKernel32.NewProc("GlobalLock")
	clipboardUnlock        = clipboardKernel32.NewProc("GlobalUnlock")
	clipboardFree          = clipboardKernel32.NewProc("GlobalFree")
	clipboardCopy          = windows.NewLazySystemDLL("ntdll.dll").NewProc("RtlMoveMemory")
)

// CF_UNICODETEXT requires UTF16LE terminated by NUL, without a byte-order mark.
func nativeClipboardBytes(value string) []byte {
	units := utf16.Encode([]rune(value))
	data := make([]byte, 2*(len(units)+1))
	for i, unit := range units {
		binary.LittleEndian.PutUint16(data[2*i:], unit)
	}
	return data
}

func clipboardResult(name string, result uintptr, err error) error {
	if result != 0 {
		return nil
	}
	return fmt.Errorf("Windows clipboard %s failed: %w", name, err)
}

func copyNativeClipboard(value string) (bool, error) {
	// Remote terminals must update the terminal client's clipboard through OSC52.
	if os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_TTY") != "" {
		return false, nil
	}
	if strings.ContainsRune(value, '\x00') {
		return true, fmt.Errorf("clipboard text contains NUL")
	}
	// Clipboard ownership and the hidden message-only window belong to this thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	class, _ := windows.UTF16PtrFromString("STATIC")
	window, _, err := clipboardCreateWindow.Call(0, uintptr(unsafe.Pointer(class)), 0, 0, 0, 0, 0, 0, ^uintptr(2), 0, 0, 0)
	if err := clipboardResult("owner window", window, err); err != nil {
		return true, err
	}
	defer clipboardDestroyWindow.Call(window)
	data := nativeClipboardBytes(value)
	memory, _, err := clipboardAlloc.Call(0x0002, uintptr(len(data))) // GMEM_MOVEABLE
	if err := clipboardResult("allocate", memory, err); err != nil {
		return true, err
	}
	transferred := false
	defer func() {
		if !transferred {
			clipboardFree.Call(memory)
		}
	}()
	address, _, err := clipboardLock.Call(memory)
	if err := clipboardResult("lock memory", address, err); err != nil {
		return true, err
	}
	clipboardCopy.Call(address, uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)))
	runtime.KeepAlive(data)
	clipboardUnlock.Call(memory)
	deadline := time.Now().Add(2 * time.Second)
	for {
		opened, _, openErr := clipboardOpen.Call(window)
		if opened != 0 {
			break
		}
		if !time.Now().Before(deadline) {
			return true, clipboardResult("open", opened, openErr)
		}
		time.Sleep(10 * time.Millisecond)
	}
	defer clipboardClose.Call()
	emptied, _, err := clipboardEmpty.Call()
	if err := clipboardResult("empty", emptied, err); err != nil {
		return true, err
	}
	result, _, err := clipboardSet.Call(13, memory) // CF_UNICODETEXT
	if err := clipboardResult("set text", result, err); err != nil {
		return true, err
	}
	transferred = true // Windows now owns and eventually frees the allocation.
	return true, nil
}
