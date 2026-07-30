package completion

type Context struct {
	Prefix       string `json:"prefix"`
	Suffix       string `json:"suffix"`
	Language     string `json:"language"`
	FilePath     string `json:"filePath"`
	CursorOffset int    `json:"cursorOffset"`
}

type Client struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Request struct {
	Context Context `json:"context"`
	Client  *Client `json:"client,omitempty"`
}

type Result struct {
	Text string
}
