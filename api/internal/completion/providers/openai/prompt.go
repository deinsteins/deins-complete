package openai

import (
	"fmt"

	"deinscomplete/api/internal/completion"
)

const systemInstruction = `Complete code at the cursor. Return only inserted code. Do not repeat prefix or suffix. Do not explain.`

func BuildMessages(request completion.Request) []Message {
	return []Message{
		{Role: "system", Content: systemInstruction},
		{Role: "user", Content: fmt.Sprintf("Language: %s\n\n<PREFIX>\n%s\n</PREFIX>\n\n<SUFFIX>\n%s\n</SUFFIX>%s\n\nReturn only the code inserted between PREFIX and SUFFIX.", request.Context.Language, request.Context.Prefix, request.Context.Suffix, repositoryPrompt(request))},
	}
}

func repositoryPrompt(request completion.Request) string {
	if request.RepositoryContext == nil || (len(request.RepositoryContext.Files) == 0 && len(request.RepositoryContext.Dependencies) == 0) {
		return ""
	}
	text := "\n\n<REPOSITORY_CONTEXT>"
	if len(request.RepositoryContext.Dependencies) > 0 {
		text += fmt.Sprintf("\n<DEPENDENCIES>%v</DEPENDENCIES>", request.RepositoryContext.Dependencies)
	}
	for _, file := range request.RepositoryContext.Files {
		text += fmt.Sprintf("\n<FILE path=%q language=%q reason=%q>\n%s\n</FILE>", file.Path, file.Language, file.Reason, file.Content)
	}
	for _, symbol := range request.RepositoryContext.Symbols {
		text += fmt.Sprintf("\n<SYMBOL name=%q kind=%q source=%q>%s</SYMBOL>", symbol.Name, symbol.Kind, symbol.FilePath, symbol.Signature)
	}
	return text + "\n</REPOSITORY_CONTEXT>"
}
