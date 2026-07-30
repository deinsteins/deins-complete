import * as vscode from "vscode";
import { registerCommands } from "./commands/registerCommands";
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
    statusBar.update(lifecycle.getState());

    context.subscriptions.push(
      statusBar,
      lifecycle.start(),
      lifecycle.onDidChangeState((state) => statusBar.update(state)),
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
