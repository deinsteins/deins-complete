package completion

type CodeStyle struct {
	Indentation string `json:"indentation"`
	IndentSize  int    `json:"indentSize,omitempty"`
	Quote       string `json:"quote,omitempty"`
	Semicolons  string `json:"semicolons,omitempty"`
}

type Context struct {
	Prefix       string     `json:"prefix"`
	Suffix       string     `json:"suffix"`
	Language     string     `json:"language"`
	FilePath     string     `json:"filePath"`
	CursorOffset int        `json:"cursorOffset"`
	Style        *CodeStyle `json:"style,omitempty"`
}

type Client struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type RepositoryContextFile struct {
	Path     string `json:"path"`
	Language string `json:"language"`
	Content  string `json:"content"`
	Reason   string `json:"reason"`
}

type RepositorySymbol struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	FilePath  string `json:"filePath"`
	Signature string `json:"signature,omitempty"`
}

type SignatureHelp struct {
	Label           string `json:"label"`
	ActiveParameter int    `json:"activeParameter,omitempty"`
	Parameter       string `json:"parameter,omitempty"`
}

type RepositoryContext struct {
	Files         []RepositoryContextFile `json:"files"`
	Symbols       []RepositorySymbol      `json:"symbols,omitempty"`
	Dependencies  []string                `json:"dependencies,omitempty"`
	Focus         string                  `json:"focus,omitempty"`
	Diagnostics   []string                `json:"diagnostics,omitempty"`
	SignatureHelp *SignatureHelp          `json:"signatureHelp,omitempty"`
}

type Request struct {
	Context           Context            `json:"context"`
	Intent            string             `json:"intent,omitempty"`
	RepositoryContext *RepositoryContext `json:"repositoryContext,omitempty"`
	Client            *Client            `json:"client,omitempty"`
}

type Result struct {
	Text             string
	FinishReason     string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}
