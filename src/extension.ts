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
    const requests = new RequestManager(engine, config);
    const repositoryContext = new RepositoryContextBuilder(config);
    const feedback = new FeedbackService();
    const completionProvider = new DeinsCompleteInlineCompletionProvider(lifecycle, new ContextBuilder(config, undefined, getSafeFilePath), repositoryContext, requests, logger);
    statusBar.update(lifecycle.getState());

    context.subscriptions.push(
      statusBar,
      { dispose: () => requests.dispose() },
      lifecycle.start(),
      lifecycle.onDidChangeState((state) => statusBar.update(state)),
      vscode.workspace.onDidChangeTextDocument((event) => repositoryContext.invalidate(event.document.uri)),
      vscode.window.onDidChangeActiveTextEditor((editor) => { if (editor !== undefined) repositoryContext.record(editor.document); }),
      vscode.languages.registerInlineCompletionItemProvider({ scheme: "file" }, completionProvider),
      ...registerCommands(config, lifecycle, logger, backendClient, requests, installation, repositoryContext, engine, feedback),
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
