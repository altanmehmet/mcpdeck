//go:build !windows

package main

import "fmt"

func main() { fmt.Println("MCPDeck Setup is a Windows installer. Build it with GOOS=windows.") }
