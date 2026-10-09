package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var expectedPayloadHash string

func extractPackage(payload []byte, destination, architecture string) (string, error) {
	digest := sha256.Sum256(payload)
	if expectedPayloadHash == "" || hex.EncodeToString(digest[:]) != expectedPayloadHash {
		return "", fmt.Errorf("installer payload integrity check failed")
	}
	reader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		return "", err
	}
	if len(reader.File) > 50 {
		return "", fmt.Errorf("unexpected installer payload")
	}
	var total uint64
	var packageRoot string
	for _, f := range reader.File {
		name := f.Name
		if strings.ContainsAny(name, ":\\") || strings.HasPrefix(name, "/") || name == "" || strings.Contains(name, "\x00") {
			return "", fmt.Errorf("unsafe payload path")
		}
		clean := filepath.Clean(filepath.FromSlash(name))
		if filepath.ToSlash(clean) != strings.TrimSuffix(name, "/") {
			return "", fmt.Errorf("non-canonical payload path")
		}
		if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || filepath.IsAbs(clean) {
			return "", fmt.Errorf("unsafe payload path")
		}
		root := strings.Split(name, "/")[0]
		if packageRoot == "" {
			packageRoot = root
		}
		if root != packageRoot {
			return "", fmt.Errorf("payload must contain a single package")
		}
		if f.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("linked payload files are unsupported")
		}
		total += f.UncompressedSize64
		if total > 64*1024*1024 || f.UncompressedSize64 > 32*1024*1024 {
			return "", fmt.Errorf("installer payload exceeds size limits")
		}
		target := filepath.Join(destination, clean)
		if f.FileInfo().IsDir() {
			if err = os.MkdirAll(target, 0700); err != nil {
				return "", err
			}
			continue
		}
		if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return "", err
		}
		input, e := f.Open()
		if e != nil {
			return "", e
		}
		output, e := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			input.Close()
			return "", e
		}
		n, e := io.Copy(output, io.LimitReader(input, 32*1024*1024+1))
		input.Close()
		closeErr := output.Close()
		if e != nil {
			return "", e
		}
		if closeErr != nil {
			return "", closeErr
		}
		if n > 32*1024*1024 {
			return "", fmt.Errorf("payload file exceeds size limit")
		}
	}
	root := filepath.Join(destination, packageRoot)
	raw, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		return "", err
	}
	var manifest struct {
		Application  string `json:"application"`
		Platform     string `json:"platform"`
		Architecture string `json:"architecture"`
		SHA256       string `json:"sha256"`
	}
	if json.Unmarshal(raw, &manifest) != nil || manifest.Application != "MCPDeck Desktop" || manifest.Platform != "windows" || manifest.Architecture != architecture {
		return "", fmt.Errorf("this installer does not match the Windows architecture")
	}
	binary, err := os.Open(filepath.Join(root, "MCPDeck.exe"))
	if err != nil {
		return "", err
	}
	h := sha256.New()
	_, err = io.Copy(h, binary)
	binary.Close()
	if err != nil {
		return "", err
	}
	if hex.EncodeToString(h.Sum(nil)) != manifest.SHA256 {
		return "", fmt.Errorf("application checksum mismatch")
	}
	return root, nil
}
