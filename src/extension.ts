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

export function activate(context: vscode.ExtensionContext): void {
  const logger = new Logger();
  context.subscriptions.push(logger);

  try {
    const config = new ConfigService();
    const lifecycle = new DeinsCompleteLifecycle(config);
    const statusBar = new DeinsCompleteStatusBar();
    const extensionVersion = String(context.extension.packageJSON.version);
    const backendClient = new DeinsCompleteClient(config, globalThis.fetch, logger, extensionVersion);
    const installation = new InstallationService(() => getInstallationId(context.globalState), new CredentialStore(context.secrets), backendClient);
    const authenticate = async (signal: AbortSignal): Promise<void> => {
      await installation.ensureRegistered(signal);
      backendClient.setInstallationToken(await installation.getToken());
    };
    const refreshAuthentication = async (signal: AbortSignal): Promise<void> => {
      await installation.reset();
      await authenticate(signal);
    };
    void authenticate(new AbortController().signal).catch(() => logger.debug("Installation registration deferred"));
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
    const completionProvider = new DeinsCompleteInlineCompletionProvider(lifecycle, new ContextBuilder(config, undefined, getSafeFilePath), repositoryContext, requests, logger, autoImports, feedback);
    statusBar.update(lifecycle.getState());
    if (!context.globalState.get<boolean>("deinscomplete.onboarding.seen")) {
      void context.globalState.update("deinscomplete.onboarding.seen", true);
      void vscode.window.showInformationMessage("DeinsComplete is ready. Pause after typing to see ghost text; press Tab to accept or Esc to dismiss.", "Open Diagnostics").then((choice) => { if (choice === "Open Diagnostics") void vscode.commands.executeCommand("deinscomplete.diagnostics"); });
    }

    context.subscriptions.push(
      statusBar,
      { dispose: () => { if (statusReset !== undefined) clearTimeout(statusReset); } },
      { dispose: () => requests.dispose() },
      lifecycle.start(),
      lifecycle.onDidChangeState((state) => statusBar.update(state)),
      vscode.workspace.onDidChangeTextDocument((event) => repositoryContext.invalidate(event.document.uri)),
      vscode.window.onDidChangeActiveTextEditor((editor) => { if (editor !== undefined) repositoryContext.record(editor.document); }),
      vscode.languages.registerInlineCompletionItemProvider({ scheme: "file" }, completionProvider),
      ...registerCommands(config, lifecycle, logger, backendClient, requests, installation, repositoryContext, engine, feedback, autoImports),
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
