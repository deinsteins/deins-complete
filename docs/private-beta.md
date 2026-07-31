# Private beta installation

1. Download the supplied `deinscomplete-<version>.vsix`.
2. In VS Code, open Extensions, choose **Install from VSIX…**, then reload.
3. Confirm the DeinsComplete status bar item appears.
4. Use **DeinsComplete: Diagnostics** if completion is unavailable.

CLI alternative: `code --install-extension deinscomplete-<version>.vsix`.

This private beta sends bounded cursor-adjacent code context to the DeinsComplete backend, which may forward it to configured AI providers. Repository Context is enabled by default and may add small snippets from relevant local imports and open workspace files, plus relevant package names from `package.json`; it never reads `node_modules`, uploads lockfiles, or indexes the entire repository. Disable it per workspace with `deinscomplete.repositoryContext.enabled`. Do not use it with code you are not authorized to send for AI processing.

An account is optional: a fresh installation receives anonymous Free access. Signing in links the current installation to an account so plan usage can be shared across a user's devices. Account and installation credentials are stored by VS Code's secure SecretStorage; neither is written to settings. See `docs/accounts-and-database.md` for account, privacy, and entitlement details.
