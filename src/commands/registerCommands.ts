import * as vscode from "vscode";
import { ConfigService } from "../config/configService";
import { BackendClient } from "../api/deinsCompleteClient";
import { DeinsCompleteLifecycle } from "../core/lifecycle";
import { Logger } from "../logging/logger";
import { RequestManager } from "../completion/requestManager";

export function registerCommands(
  config: ConfigService,
  lifecycle: DeinsCompleteLifecycle,
  logger: Logger,
  backendClient: BackendClient,
  requests: RequestManager,
): vscode.Disposable[] {
  return [
    vscode.commands.registerCommand("deinscomplete.enable", async () => {
      await config.setEnabled(true);
      lifecycle.refresh();
      logger.info("DeinsComplete enabled.");
    }),
    vscode.commands.registerCommand("deinscomplete.disable", async () => {
      await config.setEnabled(false);
      requests.cancelAll();
      lifecycle.refresh();
      logger.info("DeinsComplete disabled.");
    }),
    vscode.commands.registerCommand("deinscomplete.showLogs", () => logger.show()),
    vscode.commands.registerCommand("deinscomplete.clearCompletionCache", () => { requests.clearCache(); logger.info("Completion cache cleared."); void vscode.window.showInformationMessage("DeinsComplete completion cache cleared."); }),
    vscode.commands.registerCommand("deinscomplete.showCompletionStats", () => { const stats=requests.getStats(); logger.info(`Completion stats requests=${stats.requested} backend=${stats.backendRequests} cacheHits=${stats.cacheHits} deduplicated=${stats.deduplicated} cancelled=${stats.cancelled}`); void vscode.window.showInformationMessage(`DeinsComplete: ${stats.backendRequests} backend requests · ${stats.cacheHits} cache hits`); }),
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
