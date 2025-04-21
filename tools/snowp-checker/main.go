// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	rootDir := "."
	pattern := regexp.MustCompile(`@0x[a-fA-F0-9]{8}`)

	found := make(map[string]string)
	hasDuplicate := false

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || filepath.Ext(d.Name()) != ".snowp" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("error reading file %s: %w", path, err)
		}

		matches := pattern.FindAllString(string(data), -1)
		for _, match := range matches {
			match = strings.ToLower(match)
			if existingFile, exists := found[match]; exists {
				fmt.Fprintf(os.Stderr, "Duplicate found: %s in both %s and %s\n", match, existingFile, path)
				hasDuplicate = true
			} else {
				found[match] = path
			}
		}
		return nil
	})

	rc := 0

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error walking the path %q: %v\n", rootDir, err)
		rc = 2
	} else if hasDuplicate {
		rc = 1
	}
	os.Exit(rc)
}
