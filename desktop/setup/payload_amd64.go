//go:build windows && amd64

package main

import _ "embed"

//go:embed payload_amd64.zip
var payload []byte
