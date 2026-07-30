# DeinsComplete

Fast AI-powered inline code completion for VS Code.

The VS Code extension remains at the repository root. The standalone Go gateway lives in `api/`, and the language-neutral HTTP contract lives in `contracts/openapi.yaml`. The extension intentionally continues using its local mock engine until a later phase connects it to the API.

## Backend development

```bash
cd api
go run ./cmd/server
go test ./...
```
