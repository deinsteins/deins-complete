# DeinsComplete

Fast AI-powered inline code completion for VS Code.

The VS Code extension remains at the repository root. The standalone Go gateway lives in `api/`, and the language-neutral HTTP contract lives in `contracts/openapi.yaml`. The extension intentionally continues using its local mock engine until a later phase connects it to the API.

## Backend development

```bash
cd api
go run ./cmd/server
go test ./...
```

To test the extension against the mock backend, start the API above, launch an Extension Development Host, then type `const user =` in a file. The expected inline suggestion is `await getUser();`. The backend URL defaults to `http://127.0.0.1:3001` and may be overridden with `deinscomplete.backend.url`. No real AI provider is used yet.
