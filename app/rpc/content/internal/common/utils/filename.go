package utils

import (
	"path"
	"strings"
)

func SanitizeFileName(fileName, suffix string) string {
	name := path.Base(fileName)
	name = strings.ReplaceAll(name, "\\", "_")
	if name == "." {
		name = ""
	}
	if suffix != "" && !strings.HasSuffix(strings.ToLower(name), suffix) {
		name += suffix
	}
	return name
}
