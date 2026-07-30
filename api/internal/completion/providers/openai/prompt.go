package openai

import (
	"fmt"

	"deinscomplete/api/internal/completion"
)

const systemInstruction = `Complete code at the cursor. Return only inserted code. Do not repeat prefix or suffix. Do not explain.`

func BuildMessages(request completion.Request) []Message {
	return []Message{
		{Role: "system", Content: systemInstruction},
		{Role: "user", Content: fmt.Sprintf("Language: %s\n\n<PREFIX>\n%s\n</PREFIX>\n\n<SUFFIX>\n%s\n</SUFFIX>\n\nReturn only the code inserted between PREFIX and SUFFIX.", request.Context.Language, request.Context.Prefix, request.Context.Suffix)},
	}
}
