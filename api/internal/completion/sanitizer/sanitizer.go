package sanitizer

import (
	"strings"

	"deinscomplete/api/internal/completion"
)

type Sanitizer interface {
	Sanitize(completion.Request, string) string
}
type Config struct{ MaxLines, MaxChars int }
type CompletionSanitizer struct{ config Config }

func New(config Config) CompletionSanitizer { return CompletionSanitizer{config: config} }

func (s CompletionSanitizer) Sanitize(req completion.Request, raw string) string {
	text := strings.ReplaceAll(strings.ReplaceAll(raw, "\r\n", "\n"), "\r", "\n")
	req.Context.Prefix = strings.ReplaceAll(strings.ReplaceAll(req.Context.Prefix, "\r\n", "\n"), "\r", "\n")
	req.Context.Suffix = strings.ReplaceAll(strings.ReplaceAll(req.Context.Suffix, "\r\n", "\n"), "\r", "\n")
	text = stripFence(text)
	text = stripExplanation(text)
	text = trimBlankLines(text)
	text = removePrefixOverlap(req.Context.Prefix, text)
	text = removeSuffixOverlap(text, req.Context.Suffix)
	text = collapseBlankLines(trimBlankLines(text))
	text = limitLines(text, s.config.MaxLines)
	text = limitChars(text, s.config.MaxChars)
	return trimBlankLines(text)
}
func stripFence(s string) string {
	t := strings.TrimSpace(s)
	if !strings.HasPrefix(t, "```") {
		return strings.TrimSuffix(s, "\n```")
	}
	first := strings.Index(t, "\n")
	if first < 0 {
		return ""
	}
	rest := t[first+1:]
	if end := strings.Index(rest, "\n```"); end >= 0 {
		return rest[:end]
	}
	return strings.TrimSuffix(rest, "```")
}
func stripExplanation(s string) string {
	lines := strings.Split(s, "\n")
	if len(lines) > 1 {
		switch strings.ToLower(strings.TrimSpace(lines[0])) {
		case "here is the completion:", "the code should be:", "you can use:", "completion:", "typescript:", "typescript":
			return strings.Join(lines[1:], "\n")
		}
	}
	return s
}
func trimBlankLines(s string) string {
	lines := strings.Split(s, "\n")
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}
func collapseBlankLines(s string) string {
	var out []string
	blanks := 0
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) == "" {
			blanks++
			if blanks > 1 {
				continue
			}
		} else {
			blanks = 0
		}
		out = append(out, l)
	}
	return strings.Join(out, "\n")
}
func removePrefixOverlap(prefix, text string) string {
	n := overlap(prefix, text)
	if n == 0 {
		return text
	}
	return strings.TrimPrefix(text[n:], " ")
}
func removeSuffixOverlap(text, suffix string) string {
	n := overlap(suffix, text)
	_ = n
	max := min(1000, min(len(text), len(suffix)))
	for i := max; i >= 2; i-- {
		if text[len(text)-i:] == suffix[:i] {
			return text[:len(text)-i]
		}
	}
	return text
}
func overlap(a, b string) int {
	max := min(1000, min(len(a), len(b)))
	for i := max; i >= 1; i-- {
		if a[len(a)-i:] == b[:i] {
			return i
		}
	}
	return 0
}
func limitLines(s string, max int) string {
	if max < 1 {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) > max {
		lines = lines[:max]
	}
	return strings.Join(lines, "\n")
}
func limitChars(s string, max int) string {
	if len(s) <= max {
		return s
	}
	cut := strings.LastIndex(s[:max], "\n")
	if cut > 0 {
		return s[:cut]
	}
	return s[:max]
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
