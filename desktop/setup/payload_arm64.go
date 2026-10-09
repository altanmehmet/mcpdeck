//go:build windows && arm64

package main

import _ "embed"

//go:embed payload_arm64.zip
var payload []byte
