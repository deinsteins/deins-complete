# DeinsComplete

Fast AI-powered inline code completion for VS Code.

Production deployment instructions are in `docs/aws-ec2-deployment.md`, `docs/private-beta.md`, and `docs/release-checklist.md`.

The VS Code extension remains at the repository root. The standalone Go gateway lives in `api/`, and the language-neutral HTTP contract lives in `contracts/openapi.yaml`. The extension sends bounded completion context to the gateway; `AI_PROVIDER=mock` keeps local development deterministic.

## Backend development

```bash
cd api
go run ./cmd/server
go test ./...
```

To test the extension against the mock backend, start the API above, launch an Extension Development Host, then type `const user =` in a file. The expected inline suggestion is `await getUser();`. The backend URL defaults to `http://127.0.0.1:3001` and may be overridden with `deinscomplete.backend.url`.

## Backend AI provider setup

The backend supports `AI_PROVIDER=mock`, `AI_PROVIDER=openai-compatible`, and `AI_PROVIDER=anthropic`. Provider setup is server-side only.

For an OpenAI-compatible API, set these server-side environment variables (for example in an uncommitted `api/.env` managed by your process runner):

```env
AI_PROVIDER=openai-compatible
AI_BASE_URL=https://provider.example.com/v1
AI_API_KEY=
AI_MODEL=
AI_TIMEOUT_MS=10000
AI_MAX_TOKENS=128
AI_TEMPERATURE=0.1
```

For Anthropic Messages API, use the shared timeout/token settings and configure:

```env
AI_PROVIDER=anthropic
ANTHROPIC_BASE_URL=https://api.anthropic.com
ANTHROPIC_API_KEY=
ANTHROPIC_MODEL=
ANTHROPIC_VERSION=2023-06-01
```

These are backend-operator settings only. VS Code users do not configure an upstream provider, model, base URL, or API key. The backend sends only bounded `language`, `prefix`, and `suffix` context to the configured provider.

Provider output is sanitized before it reaches ghost text: obvious Markdown fences, common explanation labels, surrounding-code overlap, excessive blank lines, and oversized completions are removed conservatively.

## Installation authentication

The extension creates a random installation ID and stores its issued credential in VS Code SecretStorage. With `AUTH_ENABLED=true`, the API requires that credential for completion requests while `/health`, `/ready`, and installation registration remain public. Configure the backend only:

```env
AUTH_ENABLED=true
AUTH_TOKEN_SECRET=
AUTH_TOKEN_TTL_HOURS=0
AUTH_TOKEN_VERSION=1
```

Use a high-entropy secret of at least 32 bytes (for example, `openssl rand -hex 32`). `AUTH_TOKEN_TTL_HOURS=0` disables expiry. Production refuses to start with authentication disabled. The installation ID is random and is not derived from a user, device, or workspace.

## Admission controls

Rate limiting and daily request quotas are backend-only, keyed by installation ID. Their in-memory state resets on process restart; daily quota boundaries are midnight UTC.

```env
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_MINUTE=60
RATE_LIMIT_BURST=10
USAGE_QUOTA_ENABLED=true
USAGE_QUOTA_DAILY_REQUESTS=2000
```

## Provider fallback

The backend uses the configured `AI_PROVIDER` as its primary target. Set `AI_FALLBACK_ENABLED=true` and configure one server-side fallback target to use sequential fallback for transient provider failures. The router has a shared time budget and never retries a target or races providers in parallel.

```env
AI_FALLBACK_ENABLED=true
AI_FALLBACK_PROVIDER=mock
AI_MAX_ROUTER_ATTEMPTS=2
AI_ROUTER_TIMEOUT_MS=8000
```

Provider targets and routing remain invisible to VS Code users.

## FIM-aware completion

Targets default to chat prompting. An OpenAI-compatible raw-completion target may use native FIM only when configured server-side with explicit tokens:

```env
AI_COMPLETION_MODE=fim
AI_API_MODE=completion
AI_FIM_PREFIX_TOKEN=
AI_FIM_SUFFIX_TOKEN=
AI_FIM_MIDDLE_TOKEN=
AI_FIM_END_TOKEN=
```

FIM token values are model-specific and must be supplied by the backend operator; they are never exposed to extension users.
