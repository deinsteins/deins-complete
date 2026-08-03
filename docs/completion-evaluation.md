# Completion evaluation

Use only synthetic or explicitly shareable fixtures. The deterministic fixture suite in
`test/fixtures/completion-quality.json` protects fast/full routing and completion-focus
classification for local TypeScript, cross-file services, React props, Material UI,
Tailwind, and imported function calls.

The extension classifies each cursor position into a bounded completion intent:
component props, member access, function arguments, Tailwind classes, object fields,
imports, function bodies, condition expressions, type definitions, or general. The
backend uses that intent to select concise prompt guidance and output limits. It is a
hint only: unknown values are rejected, while older clients without an intent remain
compatible.

For model-quality A/B evaluation, keep the cursor input and target configuration fixed,
then record:

- time to first useful completion and total latency;
- whether the suggestion compiles/parses;
- correct library API, symbol, props, or type fields;
- suffix repetition and unnecessary explanation;
- useful/incorrect/empty result.

Also cover Ant Design, TypeScript object initialization, Go/Python calls, and mid-line
FIM manually when a compatible provider is available. Never run fixture load tests
against a paid production provider.

Do not include customer code, absolute paths, credentials, prompts, or raw proprietary
completion text in recorded results.

When comparing quality, report results per intent. A global acceptance rate can hide a
regression in JSX props or imported API completion.

## Controlled provider run

The evaluation command uses the managed backend, registers an ephemeral installation,
and makes exactly one paid provider request per fixture. It prints only aggregate
quality signals, character counts, latency, and request IDs—not completion content or
the installation token.

```bash
EVALUATION_CONFIRM_PROVIDER_COST=yes \
EVALUATION_BACKEND_URL=https://api.deinscomplete.web.id \
npm run evaluate:provider
```

Use a staging backend where possible. The explicit cost acknowledgement prevents CI or
an accidental local command from calling a paid provider.
