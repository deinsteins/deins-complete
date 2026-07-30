# DeinsComplete

Fast AI-powered inline code completion for VS Code.

The VS Code extension remains at the repository root. The standalone Go gateway lives in `api/`, and the language-neutral HTTP contract lives in `contracts/openapi.yaml`. The extension intentionally continues using its local mock engine until a later phase connects it to the API.

## Backend development

```bash
cd api
go run ./cmd/server
go test ./...
```

To test the extension against the mock backend, start the API above, launch an Extension Development Host, then type `const user =` in a file. The expected inline suggestion is `await getUser();`. The backend URL defaults to `http://127.0.0.1:3001` and may be overridden with `deinscomplete.backend.url`.

## Backend AI provider setup

The backend defaults to `AI_PROVIDER=mock` for local development. To use an OpenAI-compatible API, set these server-side environment variables (for example in an uncommitted `api/.env` managed by your process runner):

```env
AI_PROVIDER=openai-compatible
AI_BASE_URL=https://provider.example.com/v1
AI_API_KEY=
AI_MODEL=
AI_TIMEOUT_MS=10000
AI_MAX_TOKENS=128
AI_TEMPERATURE=0.1
```

These are backend-operator settings only. VS Code users do not configure an upstream provider, model, base URL, or API key. The backend sends only bounded `language`, `prefix`, and `suffix` context to the configured provider.
