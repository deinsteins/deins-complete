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
import { completionFocus } from "./completion/contextComplexity";
import { conflictingInlineCompletionExtensions } from "./completion/conflictDetection";
import { showAccountCenter, showWelcome } from "./ui/extensionPanels";
import { QualityReporter } from "./feedback/qualityReporter";
import { QualityConsent, shouldOfferQualityInsights } from "./feedback/qualityConsent";

export function activate(context: vscode.ExtensionContext): void {
  const logger = new Logger();
  context.subscriptions.push(logger);

  try {
    const config = new ConfigService();
    const lifecycle = new DeinsCompleteLifecycle(config);
    const statusBar = new DeinsCompleteStatusBar();
    const extensionVersion = String(context.extension.packageJSON.version);
    const credentials = new CredentialStore(context.secrets);
    const backendClient = new DeinsCompleteClient(config, globalThis.fetch, logger, extensionVersion, () => credentials.getInstallationToken());
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
    let quotaRefresh: ReturnType<typeof setTimeout> | undefined;
    const scheduleQuotaRefresh = (): void => {
      if (quotaRefresh !== undefined) return;
      quotaRefresh = setTimeout(async () => {
        try {
          const entitlements = await account.getEntitlements();
          const limits = entitlements.limits;
          statusBar.setQuota({ plan: entitlements.plan, used: limits.used, limit: limits.monthlyCompletions }, lifecycle.getState());
        } catch { /* the periodic refresh remains the fallback */ }
        finally { quotaRefresh = undefined; }
      }, 1_000);
    };
    const authenticate = async (signal: AbortSignal): Promise<void> => {
      await installation.ensureRegistered(signal);
    };
    let authenticationRefresh: Promise<void> | undefined;
    const refreshAuthentication = async (): Promise<void> => {
      if (authenticationRefresh === undefined) {
        authenticationRefresh = (async () => {
          await installation.reset();
          await authenticate(new AbortController().signal);
          if (await account.isSignedIn()) await ensureAccountLinked();
        })().finally(() => { authenticationRefresh = undefined; });
      }
      return authenticationRefresh;
    };
    void authenticate(new AbortController().signal).catch(() => logger.debug("Installation registration deferred"));
    void account.isSignedIn().then((signedIn) => { if (signedIn) void ensureAccountLinked(); });
    let quotaNotified = false;
    let completionAvailability: "offline" | "quota" | undefined;
    const engine = new BackendCompletionEngine(backendClient, extensionVersion, logger, authenticate, refreshAuthentication, () => {
      if (!quotaNotified) { quotaNotified = true; void vscode.window.showWarningMessage("DeinsComplete completion quota reached. Open Account Center for details.", "Account Center").then((choice) => { if (choice === "Account Center") void vscode.commands.executeCommand("deinscomplete.accountCenter"); }); }
    }, () => config.streamingEnabled(), scheduleQuotaRefresh, (state) => {
      completionAvailability = state === "ready" ? undefined : state;
    });
    const feedback = new FeedbackService(context.workspaceState);
    const quality = new QualityReporter(config, backendClient, logger);
    const qualityConsentKey = "deinscomplete.qualityInsights.consent.v1";
    let qualityConsentTimer: ReturnType<typeof setTimeout> | undefined;
    if (shouldOfferQualityInsights(context.globalState.get<QualityConsent>(qualityConsentKey), config.qualityInsightsConfigured(), config.qualityInsightsEnabled())) {
      qualityConsentTimer = setTimeout(() => {
        void vscode.window.showInformationMessage(
          "Help improve DeinsComplete with privacy-safe completion metrics? No source code or file paths are sent.",
          "Enable Quality Insights",
          "Not Now",
          "Learn More",
        ).then(async (choice) => {
          if (choice === "Enable Quality Insights") {
            await config.setQualityInsightsEnabled(true);
            await context.globalState.update(qualityConsentKey, "enabled" satisfies QualityConsent);
            logger.info("Privacy-safe quality insights enabled by user consent.");
            return;
          }
          if (choice === "Learn More") {
            await context.globalState.update(qualityConsentKey, "learned-more" satisfies QualityConsent);
            await vscode.env.openExternal(vscode.Uri.parse("https://github.com/deinsteins/deins-complete/blob/master/docs/privacy.md"));
            return;
          }
          await context.globalState.update(qualityConsentKey, "declined" satisfies QualityConsent);
        });
      }, 2_000);
    }
    let statusReset: ReturnType<typeof setTimeout> | undefined;
    const requests = new RequestManager(engine, config, (activity) => {
      if (statusReset !== undefined) clearTimeout(statusReset);
      if (activity === "thinking") {
        statusBar.setActivity("Thinking…");
      } else if (activity === "cached") {
        statusBar.setActivity("Cached");
        statusReset = setTimeout(() => statusBar.update(lifecycle.getState()), 800);
      } else {
        if (completionAvailability === "offline") statusBar.setActivity("Offline");
        else if (completionAvailability === "quota") statusBar.setQuotaExceeded();
        else statusBar.update(lifecycle.getState());
      }
    }, (request) => feedback.debounceAdjustment(completionFocus(request)));
    const repositoryContext = new RepositoryContextBuilder(config);
    const autoImports = new AutoImportResolver();
    const completionProvider = new DeinsCompleteInlineCompletionProvider(lifecycle, new ContextBuilder(config, undefined, getSafeFilePath), repositoryContext, requests, logger, autoImports, feedback, quality, canComplete);
    statusBar.update(lifecycle.getState());
    void refreshAccountStatus();
    const accountRefresh = setInterval(() => void refreshAccountStatus(), 10 * 60 * 1000);
    if (!context.globalState.get<boolean>("deinscomplete.onboarding.seen.v2")) {
      void context.globalState.update("deinscomplete.onboarding.seen.v2", true);
      showWelcome();
    }
    if (!context.globalState.get<boolean>("deinscomplete.conflictNotice.seen")) {
      const conflicts = conflictingInlineCompletionExtensions(vscode.extensions.all.map((extension) => extension.id));
      if (conflicts.length > 0) {
        void context.globalState.update("deinscomplete.conflictNotice.seen", true);
        void vscode.window.showInformationMessage(`Another inline completion extension is installed (${conflicts.join(", ")}). Disable one if ghost text conflicts.`, "Open Extensions").then((choice) => {
          if (choice === "Open Extensions") void vscode.commands.executeCommand("workbench.extensions.search", "@enabled");
        });
      }
    }
    context.subscriptions.push(
      statusBar,
      { dispose: () => { if (statusReset !== undefined) clearTimeout(statusReset); if (quotaRefresh !== undefined) clearTimeout(quotaRefresh); if (qualityConsentTimer !== undefined) clearTimeout(qualityConsentTimer); clearInterval(accountRefresh); } },
      { dispose: () => requests.dispose() },
      lifecycle.start(),
      lifecycle.onDidChangeState((state) => statusBar.update(state)),
      vscode.workspace.onDidChangeTextDocument((event) => repositoryContext.invalidate(event.document.uri)),
      vscode.window.onDidChangeActiveTextEditor((editor) => { if (editor !== undefined) repositoryContext.record(editor.document); }),
      // Remote WSL/SSH workspaces use vscode-remote rather than file URIs.
      vscode.languages.registerInlineCompletionItemProvider([{ scheme: "file" }, { scheme: "vscode-remote" }], completionProvider),
      vscode.commands.registerCommand("deinscomplete.accountCenter", () => showAccountCenter(account, refreshAccountStatus, logger)),
      vscode.commands.registerCommand("deinscomplete.welcome", () => showWelcome()),
      ...registerCommands(config, lifecycle, logger, backendClient, requests, installation, repositoryContext, engine, feedback, quality, autoImports, account, refreshAccountStatus, () => authenticate(new AbortController().signal)),
    );
    logger.info(`DeinsComplete activated version=${extensionVersion} backend=${safeBackendOrigin(config.getBackendUrl())}`);
  } catch (error) {
    logger.error("DeinsComplete failed to initialize", error);
    void vscode.window.showErrorMessage("DeinsComplete failed to initialize. See Output → DeinsComplete.");
  }
}

export function deactivate(): void {
  // All extension resources are owned by context.subscriptions.
}

function safeBackendOrigin(value: string): string {
  try { return new URL(value).origin; } catch { return "invalid"; }
}
