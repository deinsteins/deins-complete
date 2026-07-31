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
import { AutoImportPlan, AutoImportResolver } from "../completion/autoImportResolver";
import { AccountService } from "../account/accountService";

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
  account: AccountService,
  refreshAccountStatus: () => Promise<void>,
  syncInstallationAuthentication: () => Promise<void>,
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
      const votes = feedback.getStats(); const acceptance = votes.shown ? Math.round(votes.accepted / votes.shown * 100) : 0; const report = `DeinsComplete Diagnostics\nStatus: ${lifecycle.getState()}\nBackend: ${backend}\nAuthentication: ${authentication}\nRequests: ${stats.requested}\nCache hits: ${stats.cacheHits}\nNegative cache hits: ${stats.negativeCacheHits}\nCancelled: ${stats.cancelled}\nLatency\nAverage debounce: ${averageDebounce} ms\nAverage backend: ${averageBackend} ms\nAverage total: ${average} ms\nCompletion outcomes (local only)\nShown: ${votes.shown}\nAccepted: ${votes.accepted}\nAcceptance: ${acceptance}%\nManual feedback: helpful ${votes.helpful}, not helpful ${votes.notHelpful}\nRepository Context\nFocus: ${repository.lastFocus}\nDependencies: ${repository.lastDependencies}\nLast files included: ${repository.lastFiles}\nLast chars: ${repository.lastCharacters}\nLast build time: ${repository.lastDurationMs} ms\nTimeouts: ${repository.timedOut}\nStreaming\nStarted: ${streaming.streamsStarted}\nSucceeded: ${streaming.streamsSucceeded}\nFallback: ${streaming.streamsFallback}\nAverage first chunk: ${ttfb} ms`;
      logger.info(report); void vscode.window.showInformationMessage(`DeinsComplete diagnostics: ${backend}`);
    }),
    vscode.commands.registerCommand("deinscomplete.resetAuthentication", async () => {
      await installation.reset();
      await syncInstallationAuthentication();
      logger.info("Installation authentication reset.");
      void vscode.window.showInformationMessage("DeinsComplete authentication reset and renewed.");
    }),
    vscode.commands.registerCommand("deinscomplete.signIn", async () => {
      const email = await vscode.window.showInputBox({ prompt: "DeinsComplete account email", placeHolder: "you@example.com", ignoreFocusOut: true, validateInput: (value) => value.trim().includes("@") ? undefined : "Enter a valid email address." });
      if (email === undefined) return;
      try {
        await installation.ensureRegistered();
        const inviteCode = await vscode.window.showInputBox({ prompt: "Invite code (leave blank if you already have an account)", ignoreFocusOut: true, password: true });
        if (inviteCode === undefined) return;
        await account.requestMagicCode(email.trim(), inviteCode.trim());
        const code = await vscode.window.showInputBox({ prompt: "Enter the code sent to your email", ignoreFocusOut: true, password: true, validateInput: (value) => value.trim() === "" ? "Enter the sign-in code." : undefined });
        if (code === undefined) return;
        await account.verifyMagicCode(email.trim(), code.trim(), await installation.getToken());
        await syncInstallationAuthentication();
        await refreshAccountStatus();
        logger.info("Account sign-in completed.");
        void vscode.window.showInformationMessage("Signed in to DeinsComplete. This installation is linked to your account.");
      } catch (error) {
        logger.error("Account sign-in failed", error);
        void vscode.window.showErrorMessage("DeinsComplete sign-in failed. Check the code and try again.");
      }
    }),
    vscode.commands.registerCommand("deinscomplete.signOut", async () => {
      try { await account.signOut(); logger.info("Account sign-out completed."); void vscode.window.showInformationMessage("Signed out of DeinsComplete. This installation remains linked to its account."); }
      catch (error) { logger.error("Account sign-out failed", error); void vscode.window.showErrorMessage("DeinsComplete sign-out failed. Your local session was cleared."); }
    }),
    vscode.commands.registerCommand("deinscomplete.accountStatus", async () => {
      try {
        const status = await account.getStatus();
        if (status === undefined) { void vscode.window.showInformationMessage("DeinsComplete account: not signed in.", "Sign In").then((choice) => { if (choice === "Sign In") void vscode.commands.executeCommand("deinscomplete.signIn"); }); return; }
        const limits = status.entitlements.limits;
        await refreshAccountStatus();
        const report = `DeinsComplete Account\nSigned in: Yes\nEmail: ${status.account.user.email}\nPlan: ${status.account.plan.code}\nMonthly usage: ${limits.used} / ${limits.monthlyCompletions}\nInstallations: ${status.installations.length}`;
        logger.info(report); void vscode.window.showInformationMessage(`DeinsComplete ${status.account.plan.code}: ${limits.used} / ${limits.monthlyCompletions} completions used`);
      } catch (error) { logger.error("Account status failed", error); void vscode.window.showErrorMessage("Unable to load DeinsComplete account status."); }
    }),
    vscode.commands.registerCommand("deinscomplete.clearCompletionCache", () => { requests.clearCache(); logger.info("Completion cache cleared."); void vscode.window.showInformationMessage("DeinsComplete completion cache cleared."); }),
    vscode.commands.registerCommand("deinscomplete.feedbackHelpful", () => { feedback.record("helpful"); logger.info("Completion feedback=helpful"); }),
    vscode.commands.registerCommand("deinscomplete.feedbackNotHelpful", () => { feedback.record("not-helpful"); logger.info("Completion feedback=not-helpful"); }),
    vscode.commands.registerCommand("deinscomplete.applyAutoImport", (uri: vscode.Uri, position: vscode.Position, completion: string) => autoImports.resolveAndApply(uri, position, completion).catch(() => logger.debug("Verified auto-import was not applied"))),
    vscode.commands.registerCommand("deinscomplete.applyPrefetchedAutoImport", (plan: AutoImportPlan) => autoImports.apply(plan).catch(() => logger.debug("Prefetched auto-import was not applied"))),
    vscode.commands.registerCommand("deinscomplete.completionAccepted", (focus: string, plan?: AutoImportPlan, uri?: vscode.Uri, position?: vscode.Position, completion?: string) => {
      feedback.recordAccepted(focus);
      if (plan !== undefined) return autoImports.apply(plan).catch(() => logger.debug("Prefetched auto-import was not applied"));
      if (uri !== undefined && position !== undefined && completion !== undefined) return autoImports.resolveAndApply(uri, position, completion).catch(() => logger.debug("Verified auto-import was not applied"));
      return undefined;
    }),
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
