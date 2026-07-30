import * as vscode from "vscode";
import { ConfigService } from "../config/configService";
import { BackendClient } from "../api/deinsCompleteClient";
import { DeinsCompleteLifecycle } from "../core/lifecycle";
import { Logger } from "../logging/logger";

export function registerCommands(
  config: ConfigService,
  lifecycle: DeinsCompleteLifecycle,
  logger: Logger,
  backendClient: BackendClient,
): vscode.Disposable[] {
  return [
    vscode.commands.registerCommand("deinscomplete.enable", async () => {
      await config.setEnabled(true);
      lifecycle.refresh();
      logger.info("DeinsComplete enabled.");
    }),
    vscode.commands.registerCommand("deinscomplete.disable", async () => {
      await config.setEnabled(false);
      lifecycle.refresh();
      logger.info("DeinsComplete disabled.");
    }),
    vscode.commands.registerCommand("deinscomplete.showLogs", () => logger.show()),
    vscode.commands.registerCommand("deinscomplete.checkBackend", async () => {
      try {
        const result = await backendClient.health();
        logger.info(`Backend health check completed latencyMs=${result.latencyMs} requestId=${result.requestId ?? "none"}`);
        void vscode.window.showInformationMessage(`DeinsComplete backend connected · ${result.latencyMs} ms`);
      } catch (error) {
        logger.error("Backend health check failed", error);
        void vscode.window.showErrorMessage("DeinsComplete backend unavailable. See Output → DeinsComplete.");
      }
    }),
  ];
}
