import * as vscode from "vscode";
import { registerCommands } from "./commands/registerCommands";
import { DeinsCompleteClient } from "./api/deinsCompleteClient";
import { ContextBuilder } from "./context/contextBuilder";
import { RepositoryContextBuilder } from "./context/repository/repositoryContextBuilder";
import { DeinsCompleteInlineCompletionProvider } from "./completion/inlineCompletionProvider";
import { BackendCompletionEngine } from "./completion/backendCompletionEngine";
import { RequestManager } from "./completion/requestManager";
import { ConfigService } from "./config/configService";
import { DeinsCompleteLifecycle } from "./core/lifecycle";
import { Logger } from "./logging/logger";
import { DeinsCompleteStatusBar } from "./status/statusBar";
import { getSafeFilePath } from "./utils/safeFilePath";
import { CredentialStore } from "./security/credentialStore";
import { getInstallationId } from "./identity/installationIdentity";
import { InstallationService } from "./identity/installationService";
import { FeedbackService } from "./feedback/feedbackService";
import { AutoImportResolver } from "./completion/autoImportResolver";
import { AccountService } from "./account/accountService";

export function activate(context: vscode.ExtensionContext): void {
  const logger = new Logger();
  context.subscriptions.push(logger);

  try {
    const config = new ConfigService();
    const lifecycle = new DeinsCompleteLifecycle(config);
    const statusBar = new DeinsCompleteStatusBar();
    const extensionVersion = String(context.extension.packageJSON.version);
    const backendClient = new DeinsCompleteClient(config, globalThis.fetch, logger, extensionVersion);
    const credentials = new CredentialStore(context.secrets);
    const installation = new InstallationService(() => getInstallationId(context.globalState), credentials, backendClient);
    const account = new AccountService(backendClient, credentials);
    let accountNoticeShown = false;
    const ensureAccountLinked = async (): Promise<boolean> => {
      try {
        await installation.ensureRegistered();
        const token = await installation.getToken();
        if (token === undefined) return false;
        await account.ensureLinked(token);
        return true;
      } catch (error) {
        logger.debug("Account installation linking is unavailable");
        return false;
      }
    };
    const canComplete = async (): Promise<boolean> => {
      try {
        if (!await account.isRequired()) return true;
        if (await account.isSignedIn() && await ensureAccountLinked()) return true;
        statusBar.setSignInRequired();
        if (!accountNoticeShown) { accountNoticeShown = true; void vscode.window.showInformationMessage("Sign in to DeinsComplete to enable inline completions.", "Sign In").then((choice) => { if (choice === "Sign In") void vscode.commands.executeCommand("deinscomplete.signIn"); }); }
        return false;
      } catch { return true; } // Existing/temporarily unavailable backend retains its normal behavior.
    };
    const refreshAccountStatus = async (): Promise<void> => {
      try {
        const status = await account.getStatus();
        if (status === undefined) { statusBar.setQuota(undefined, lifecycle.getState()); return; }
        const limits = status.entitlements.limits;
        statusBar.setQuota({ plan: status.account.plan.code, used: limits.used, limit: limits.monthlyCompletions }, lifecycle.getState());
      } catch { /* account data is optional; keep the status bar responsive */ }
    };
    const authenticate = async (signal: AbortSignal): Promise<void> => {
      await installation.ensureRegistered(signal);
      backendClient.setInstallationToken(await installation.getToken());
    };
    const refreshAuthentication = async (signal: AbortSignal): Promise<void> => {
      await installation.reset();
      await authenticate(signal);
    };
    void authenticate(new AbortController().signal).catch(() => logger.debug("Installation registration deferred"));
    void account.isSignedIn().then((signedIn) => { if (signedIn) void ensureAccountLinked(); });
    let quotaNotified = false;
    const engine = new BackendCompletionEngine(backendClient, extensionVersion, logger, authenticate, refreshAuthentication, () => {
      if (!quotaNotified) { quotaNotified = true; void vscode.window.showWarningMessage("DeinsComplete daily completion limit reached."); }
    }, () => config.streamingEnabled());
    let statusReset: ReturnType<typeof setTimeout> | undefined;
    const requests = new RequestManager(engine, config, (activity) => {
      if (statusReset !== undefined) clearTimeout(statusReset);
      if (activity === "thinking") {
        statusBar.setActivity("Thinking…");
      } else if (activity === "cached") {
        statusBar.setActivity("Cached");
        statusReset = setTimeout(() => statusBar.update(lifecycle.getState()), 800);
      } else {
        statusBar.update(lifecycle.getState());
      }
    });
    const repositoryContext = new RepositoryContextBuilder(config);
    const feedback = new FeedbackService();
    const autoImports = new AutoImportResolver();
    const completionProvider = new DeinsCompleteInlineCompletionProvider(lifecycle, new ContextBuilder(config, undefined, getSafeFilePath), repositoryContext, requests, logger, autoImports, feedback, canComplete);
    statusBar.update(lifecycle.getState());
    void refreshAccountStatus();
    const accountRefresh = setInterval(() => void refreshAccountStatus(), 10 * 60 * 1000);
    if (!context.globalState.get<boolean>("deinscomplete.onboarding.seen")) {
      void context.globalState.update("deinscomplete.onboarding.seen", true);
      void vscode.window.showInformationMessage("DeinsComplete is ready. Pause after typing to see ghost text; press Tab to accept or Esc to dismiss.", "Open Diagnostics").then((choice) => { if (choice === "Open Diagnostics") void vscode.commands.executeCommand("deinscomplete.diagnostics"); });
    }

    context.subscriptions.push(
      statusBar,
      { dispose: () => { if (statusReset !== undefined) clearTimeout(statusReset); clearInterval(accountRefresh); } },
      { dispose: () => requests.dispose() },
      lifecycle.start(),
      lifecycle.onDidChangeState((state) => statusBar.update(state)),
      vscode.workspace.onDidChangeTextDocument((event) => repositoryContext.invalidate(event.document.uri)),
      vscode.window.onDidChangeActiveTextEditor((editor) => { if (editor !== undefined) repositoryContext.record(editor.document); }),
      // Remote WSL/SSH workspaces use vscode-remote rather than file URIs.
      vscode.languages.registerInlineCompletionItemProvider([{ scheme: "file" }, { scheme: "vscode-remote" }], completionProvider),
      ...registerCommands(config, lifecycle, logger, backendClient, requests, installation, repositoryContext, engine, feedback, autoImports, account, refreshAccountStatus),
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
