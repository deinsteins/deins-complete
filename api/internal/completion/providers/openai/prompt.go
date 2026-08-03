package openai

import (
	"fmt"

	"deinscomplete/api/internal/completion"
)

const systemInstruction = `Complete code at the cursor. Return only inserted code. Do not repeat prefix or suffix. Do not explain or use Markdown. Never invent an API when repository declarations provide the answer.`

func BuildMessages(request completion.Request) []Message {
	fileName := completion.SafeFileName(request.Context.FilePath)
	return []Message{
		{Role: "system", Content: systemInstruction},
		{Role: "user", Content: fmt.Sprintf("File: %s\nLanguage: %s\nCompletion intent: %s\nIntent guidance: %s\nCode style: %s\n\n<PREFIX>\n%s\n</PREFIX>\n\n<SUFFIX>\n%s\n</SUFFIX>%s\n\nReturn only the code inserted between PREFIX and SUFFIX.", fileName, request.Context.Language, request.Intent, completion.IntentInstruction(request.Intent), completion.StyleInstruction(request.Context.Style), request.Context.Prefix, request.Context.Suffix, repositoryPrompt(request))},
	}
}

func repositoryPrompt(request completion.Request) string {
	if request.RepositoryContext == nil || (len(request.RepositoryContext.Files) == 0 && len(request.RepositoryContext.Dependencies) == 0 && len(request.RepositoryContext.Symbols) == 0 && request.RepositoryContext.Focus == "" && request.RepositoryContext.SignatureHelp == nil) {
		return ""
	}
	text := "\n\n<REPOSITORY_CONTEXT>"
	if request.RepositoryContext.Focus != "" {
		text += fmt.Sprintf("\n<COMPLETION_FOCUS>%s</COMPLETION_FOCUS>", request.RepositoryContext.Focus)
	}
	if len(request.RepositoryContext.Diagnostics) > 0 {
		text += fmt.Sprintf("\n<DIAGNOSTICS>%v</DIAGNOSTICS>", request.RepositoryContext.Diagnostics)
	}
	if len(request.RepositoryContext.Dependencies) > 0 {
		text += fmt.Sprintf("\n<DEPENDENCIES>%v</DEPENDENCIES>", request.RepositoryContext.Dependencies)
	}
	if signature := request.RepositoryContext.SignatureHelp; signature != nil {
		text += fmt.Sprintf("\n<SIGNATURE_HELP>\nLabel: %s\nActive parameter: %d\nParameter: %s\n</SIGNATURE_HELP>", signature.Label, signature.ActiveParameter, signature.Parameter)
	}
	for _, file := range request.RepositoryContext.Files {
		text += fmt.Sprintf("\n<FILE path=%q language=%q reason=%q>\n%s\n</FILE>", file.Path, file.Language, file.Reason, file.Content)
	}
	for _, symbol := range request.RepositoryContext.Symbols {
		text += fmt.Sprintf("\n<SYMBOL name=%q kind=%q source=%q>%s</SYMBOL>", symbol.Name, symbol.Kind, symbol.FilePath, symbol.Signature)
	}
	return text + "\n</REPOSITORY_CONTEXT>"
}
