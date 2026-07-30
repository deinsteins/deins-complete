# Private beta installation

1. Download the supplied `deinscomplete-<version>.vsix`.
2. In VS Code, open Extensions, choose **Install from VSIX…**, then reload.
3. Confirm the DeinsComplete status bar item appears.
4. Use **DeinsComplete: Diagnostics** if completion is unavailable.

CLI alternative: `code --install-extension deinscomplete-<version>.vsix`.

This private beta sends bounded cursor-adjacent code context to the DeinsComplete backend, which may forward it to configured AI providers. Do not use it with code you are not authorized to send for AI processing.
