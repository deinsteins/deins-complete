package fim

import (
	"path"
	"strings"

	"deinscomplete/api/internal/completion"
)

type Config struct {
	PrefixToken, SuffixToken, MiddleToken, EndToken string
	FilenameContext                                 bool
}

func Format(request completion.Request, config Config) string {
	// Native FIM token layouts are model-specific. Keep their cursor structure exact;
	// chat targets receive repository context until a target explicitly supports it.
	return config.PrefixToken + filenameContext(request, config) + request.Context.Prefix + config.SuffixToken + request.Context.Suffix + config.MiddleToken
}
func StripEnd(text, end string) string {
	if end != "" && len(text) >= len(end) && text[len(text)-len(end):] == end {
		return text[:len(text)-len(end)]
	}
	return text
}

func filenameContext(request completion.Request, config Config) string {
	if !config.FilenameContext || request.Context.FilePath == "" {
		return ""
	}
	filename := path.Base(strings.ReplaceAll(request.Context.FilePath, "\\", "/"))
	switch strings.ToLower(request.Context.Language) {
	case "python", "shellscript", "ruby":
		return "# File: " + filename + "\n"
	case "html", "xml":
		return "<!-- File: " + filename + " -->\n"
	default:
		return "// File: " + filename + "\n"
	}
}
