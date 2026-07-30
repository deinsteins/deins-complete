import * as vscode from "vscode";

export function bridgeCancellation(token: vscode.CancellationToken): { signal: AbortSignal; dispose(): void } {
  const controller = new AbortController();
  const subscription = token.onCancellationRequested(() => controller.abort());
  if (token.isCancellationRequested) {
    controller.abort();
  }

  return {
    signal: controller.signal,
    dispose: () => subscription.dispose(),
  };
}
