package openai

import (
	"fmt"

	"deinscomplete/api/internal/completion"
)

const systemInstruction = `You are a code completion engine.

Generate only the code that should be inserted at the cursor.

Rules:
- Return code only.
- Do not explain.
- Do not use markdown.
- Do not repeat existing surrounding code.
- Continue naturally from the prefix.
- Respect the suffix.
- Prefer the smallest useful completion.`

func BuildMessages(request completion.Request) []Message {
	return []Message{
		{Role: "system", Content: systemInstruction},
		{Role: "user", Content: fmt.Sprintf("Language: %s\n\n<PREFIX>\n%s\n</PREFIX>\n\n<SUFFIX>\n%s\n</SUFFIX>\n\nReturn only the code inserted between PREFIX and SUFFIX.", request.Context.Language, request.Context.Prefix, request.Context.Suffix)},
	}
}
