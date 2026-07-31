import * as vscode from "vscode";
import { ConfigService } from "../config/configService";
import { BackendClient } from "../api/deinsCompleteClient";
import { DeinsCompleteLifecycle } from "../core/lifecycle";
import { Logger } from "../logging/logger";
import { RequestManager } from "../completion/requestManager";
import { InstallationService } from "../identity/installationService";
import { RepositoryContextBuilder } from "../context/repository/repositoryContextBuilder";
import { BackendCompletionEngine } from "../completion/backendCompletionEngine";
import { FeedbackService } from "../feedback/feedbackService";
import { AutoImportResolver } from "../completion/autoImportResolver";

export function registerCommands(
  config: ConfigService,
  lifecycle: DeinsCompleteLifecycle,
  logger: Logger,
  backendClient: BackendClient,
  requests: RequestManager,
  installation: InstallationService,
  repositoryContext: RepositoryContextBuilder,
  engine: BackendCompletionEngine,
  feedback: FeedbackService,
  autoImports: AutoImportResolver,
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
    vscode.commands.registerCommand("deinscomplete.diagnostics", async () => {
      const stats = requests.getStats();
      let backend = "Unavailable";
      try { const health = await backendClient.health(); backend = `Reachable (${health.latencyMs} ms)`; } catch { /* diagnostic remains safe */ }
      const authentication = await installation.getToken() ? "Ready" : "Not registered";
      const average = stats.requested ? Math.round(stats.totalLatencyMs / stats.requested) : 0;
      const averageDebounce = stats.backendRequests ? Math.round(stats.totalDebounceMs / stats.backendRequests) : 0;
      const averageBackend = stats.backendRequests ? Math.round(stats.totalBackendMs / stats.backendRequests) : 0;
      const repository = repositoryContext.getStats();
      const streaming = engine.getStats(); const ttfb = streaming.firstChunkSamples ? Math.round(streaming.totalFirstChunkMs / streaming.firstChunkSamples) : 0;
      const votes = feedback.getStats(); const report = `DeinsComplete Diagnostics\nStatus: ${lifecycle.getState()}\nBackend: ${backend}\nAuthentication: ${authentication}\nRequests: ${stats.requested}\nCache hits: ${stats.cacheHits}\nNegative cache hits: ${stats.negativeCacheHits}\nCancelled: ${stats.cancelled}\nLatency\nAverage debounce: ${averageDebounce} ms\nAverage backend: ${averageBackend} ms\nAverage total: ${average} ms\nFeedback (local): helpful ${votes.helpful}, not helpful ${votes.notHelpful}\nRepository Context\nFocus: ${repository.lastFocus}\nDependencies: ${repository.lastDependencies}\nLast files included: ${repository.lastFiles}\nLast chars: ${repository.lastCharacters}\nLast build time: ${repository.lastDurationMs} ms\nTimeouts: ${repository.timedOut}\nStreaming\nStarted: ${streaming.streamsStarted}\nSucceeded: ${streaming.streamsSucceeded}\nFallback: ${streaming.streamsFallback}\nAverage first chunk: ${ttfb} ms`;
      logger.info(report); void vscode.window.showInformationMessage(`DeinsComplete diagnostics: ${backend}`);
    }),
    vscode.commands.registerCommand("deinscomplete.resetAuthentication", async () => {
      await installation.reset();
      logger.info("Installation authentication reset.");
      void vscode.window.showInformationMessage("DeinsComplete authentication reset. It will be renewed when needed.");
    }),
    vscode.commands.registerCommand("deinscomplete.clearCompletionCache", () => { requests.clearCache(); logger.info("Completion cache cleared."); void vscode.window.showInformationMessage("DeinsComplete completion cache cleared."); }),
    vscode.commands.registerCommand("deinscomplete.feedbackHelpful", () => { feedback.record("helpful"); logger.info("Completion feedback=helpful"); }),
    vscode.commands.registerCommand("deinscomplete.feedbackNotHelpful", () => { feedback.record("not-helpful"); logger.info("Completion feedback=not-helpful"); }),
    vscode.commands.registerCommand("deinscomplete.applyAutoImport", (uri: vscode.Uri, position: vscode.Position, completion: string) => autoImports.resolveAndApply(uri, position, completion).catch(() => logger.debug("Verified auto-import was not applied"))),
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
