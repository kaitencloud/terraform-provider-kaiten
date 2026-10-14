// Copyright 2026 KAITEN INC
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSourceHeaders fails on a Go file that does not open with the licence
// header every Kaiten repository carries; without a check, headers drift.
func TestSourceHeaders(t *testing.T) {
	const header = "// Copyright 2026 KAITEN INC\n// SPDX-License-Identifier: Apache-2.0\n"

	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == "dist") {
			return filepath.SkipDir
		}
		if d.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}

		data, err := os.ReadFile(path) //nolint:gosec // walking this repository's own tree
		if err != nil {
			return err
		}
		if !strings.HasPrefix(string(data), header) {
			t.Errorf("%s does not start with the licence header:\n%s", path, header)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
