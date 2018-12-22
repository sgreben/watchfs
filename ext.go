package main

import (
	"path/filepath"
	"strings"
)

func ext(path string) string {
	return strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
}
