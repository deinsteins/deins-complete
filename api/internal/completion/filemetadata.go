package completion

import (
	"path"
	"strings"
	"unicode"
)

// SafeFileName returns only a prompt-safe basename; machine-local directories never reach providers.
func SafeFileName(filePath string) string {
	name := path.Base(strings.ReplaceAll(filePath, "\\", "/"))
	name = strings.Map(func(value rune) rune {
		if unicode.IsLetter(value) || unicode.IsNumber(value) || strings.ContainsRune("._- @()[]", value) {
			return value
		}
		return -1
	}, name)
	if len(name) > 255 {
		return name[:255]
	}
	if name == "." {
		return ""
	}
	return name
}
