//go:build darwin

package main

/*
#cgo LDFLAGS: -framework UniformTypeIdentifiers
*/
import "C"

// Wails file dialogs use UTType from UniformTypeIdentifiers on current macOS SDKs.
