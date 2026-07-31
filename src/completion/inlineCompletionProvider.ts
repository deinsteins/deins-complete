import * as vscode from "vscode";
import { ContextBuilder } from "../context/contextBuilder";
import { RepositoryContextBuilder } from "../context/repository/repositoryContextBuilder";
import { DeinsCompleteLifecycle } from "../core/lifecycle";
import { Logger } from "../logging/logger";
import { bridgeCancellation } from "../utils/cancellation";
import { RequestManager } from "./requestManager";
import { CompletionRequest } from "./completionTypes";
import { completionRequestMode } from "./contextComplexity";
import { EditorStateSnapshot, isCurrentEditorState } from "./editorState";
import { AutoImportResolver } from "./autoImportResolver";
import { FeedbackService } from "../feedback/feedbackService";
import { completionFocus } from "./contextComplexity";

export class DeinsCompleteInlineCompletionProvider implements vscode.InlineCompletionItemProvider {
  constructor(
    private readonly lifecycle: DeinsCompleteLifecycle,
    private readonly contextBuilder: ContextBuilder,
    private readonly repositoryContextBuilder: RepositoryContextBuilder,
    private readonly requests: RequestManager,
    private readonly logger: Logger,
    private readonly autoImports: AutoImportResolver,
    private readonly feedback: FeedbackService,
    private readonly canComplete: () => Promise<boolean> = async () => true,
  ) {}

  async provideInlineCompletionItems(
    document: vscode.TextDocument,
    position: vscode.Position,
    _context: vscode.InlineCompletionContext,
    token: vscode.CancellationToken,
  ): Promise<vscode.InlineCompletionItem[]> {
    if (this.lifecycle.getState() !== "enabled" || token.isCancellationRequested || !await this.canComplete()) {
      return [];
    }

    const snapshot = this.getEditorState(document);
    const request: CompletionRequest = this.contextBuilder.build(document, position);
    const cancellation = bridgeCancellation(token);
    void this.autoImports.prefetch(document, position);
    request.mode = completionRequestMode(request);
    if (request.mode === "full") request.repositoryContextTask = this.repositoryContextBuilder.build(document, request, cancellation.signal);
    this.logger.debug(`Inline completion requested (language=${request.language}, version=${snapshot.version}, prefixChars=${request.metadata.prefixCharacters}, suffixChars=${request.metadata.suffixCharacters})`);

    try {
      const result = await this.requests.complete(document.uri.toString(), request, cancellation.signal);
      if (cancellation.signal.aborted) {
        this.logger.debug("Completion discarded: request cancelled");
        return [];
      }
      if (!isCurrentEditorState(snapshot, this.getEditorState(document))) {
        this.logger.debug("Completion discarded: document changed");
        return [];
      }
      if (this.lifecycle.getState() !== "enabled" || result === null || result.text.length === 0) {
        return [];
      }

      this.logger.debug("Inline completion produced result");
      const focus = completionFocus(request);
      this.feedback.recordShown(focus);
      const prefetchedImport = this.autoImports.getPrefetched(document, position, result.text);
      const command = {
        command: "deinscomplete.completionAccepted",
        title: "Record completion acceptance",
        arguments: [focus, prefetchedImport, document.uri, position, this.autoImports.canResolve(result.text) ? result.text : undefined],
      };
      return [new vscode.InlineCompletionItem(result.text, new vscode.Range(position, position), command)];
    } catch (error) {
      this.logger.error("Inline completion failed", error);
      return [];
    } finally {
      cancellation.dispose();
    }
  }

  private getEditorState(document: vscode.TextDocument): EditorStateSnapshot {
    return { uri: document.uri.toString(), version: document.version };
  }
}
