//go:build !windows

package tui

func copyNativeClipboard(value string) (bool, error) { return false, nil }
