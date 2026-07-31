# Completion evaluation

Use only synthetic or explicitly shareable fixtures. The deterministic fixture suite in
`test/fixtures/completion-quality.json` protects fast/full routing and completion-focus
classification for local TypeScript, cross-file services, React props, Material UI,
Tailwind, and imported function calls.

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
