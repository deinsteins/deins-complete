import * as vscode from "vscode";
import { registerCommands } from "./commands/registerCommands";
import { ContextBuilder } from "./context/contextBuilder";
import { DeinsCompleteInlineCompletionProvider } from "./completion/inlineCompletionProvider";
import { MockCompletionEngine } from "./completion/mockCompletionEngine";
import { ConfigService } from "./config/configService";
import { DeinsCompleteLifecycle } from "./core/lifecycle";
import { Logger } from "./logging/logger";
import { DeinsCompleteStatusBar } from "./status/statusBar";

export function activate(context: vscode.ExtensionContext): void {
  const logger = new Logger();
  context.subscriptions.push(logger);

  try {
    const config = new ConfigService();
    const lifecycle = new DeinsCompleteLifecycle(config);
    const statusBar = new DeinsCompleteStatusBar();
    const completionProvider = new DeinsCompleteInlineCompletionProvider(lifecycle, new ContextBuilder(config), new MockCompletionEngine(), logger);
    statusBar.update(lifecycle.getState());

    context.subscriptions.push(
      statusBar,
      lifecycle.start(),
      lifecycle.onDidChangeState((state) => statusBar.update(state)),
      vscode.languages.registerInlineCompletionItemProvider({ scheme: "file" }, completionProvider),
      ...registerCommands(config, lifecycle, logger),
    );
    logger.info("DeinsComplete activated.");
  } catch (error) {
    logger.error("DeinsComplete failed to initialize", error);
    void vscode.window.showErrorMessage("DeinsComplete failed to initialize. See Output → DeinsComplete.");
  }
}

export function deactivate(): void {
  // All extension resources are owned by context.subscriptions.
}
