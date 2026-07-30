package fim

import "deinscomplete/api/internal/completion"

type Config struct{ PrefixToken, SuffixToken, MiddleToken, EndToken string }

func Format(request completion.Request, config Config) string {
	// Native FIM token layouts are model-specific. Keep their cursor structure exact;
	// chat targets receive repository context until a target explicitly supports it.
	return config.PrefixToken + request.Context.Prefix + config.SuffixToken + request.Context.Suffix + config.MiddleToken
}
func StripEnd(text, end string) string {
	if end != "" && len(text) >= len(end) && text[len(text)-len(end):] == end {
		return text[:len(text)-len(end)]
	}
	return text
}
