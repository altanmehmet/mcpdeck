package client

import (
	"github.com/altanmehmet/mcpdeck/internal/instructions"
	"os"
)

// InstructionFile contains metadata only; contents are fetched after selection.
type InstructionFile struct {
	Path   string `json:"path"`
	Status string `json:"status"`
	Size   int64  `json:"size"`
}

func instructionFileStates(documents []instructions.Document) []InstructionFile {
	result := []InstructionFile{}
	seen := map[string]bool{}
	for _, doc := range documents {
		if seen[doc.Path] {
			continue
		}
		seen[doc.Path] = true
		item := InstructionFile{Path: doc.Path, Status: "not created"}
		info, err := os.Lstat(doc.Path)
		if err == nil {
			if info.Mode().IsRegular() {
				item.Status = "saved"
				item.Size = info.Size()
				if item.Size == 0 {
					item.Status = "empty"
				}
			} else {
				item.Status = "unsupported file type"
			}
		} else if !os.IsNotExist(err) {
			item.Status = "unreadable"
		}
		result = append(result, item)
	}
	return result
}
