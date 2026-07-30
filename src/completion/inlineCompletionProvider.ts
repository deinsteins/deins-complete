import * as vscode from "vscode";
import { ContextBuilder } from "../context/contextBuilder";
import { DeinsCompleteLifecycle } from "../core/lifecycle";
import { Logger } from "../logging/logger";
import { bridgeCancellation } from "../utils/cancellation";
import { RequestManager } from "./requestManager";
import { EditorStateSnapshot, isCurrentEditorState } from "./editorState";

export class DeinsCompleteInlineCompletionProvider implements vscode.InlineCompletionItemProvider {
  constructor(
    private readonly lifecycle: DeinsCompleteLifecycle,
    private readonly contextBuilder: ContextBuilder,
    private readonly requests: RequestManager,
    private readonly logger: Logger,
  ) {}

  async provideInlineCompletionItems(
    document: vscode.TextDocument,
    position: vscode.Position,
    _context: vscode.InlineCompletionContext,
    token: vscode.CancellationToken,
  ): Promise<vscode.InlineCompletionItem[]> {
    if (this.lifecycle.getState() !== "enabled" || token.isCancellationRequested) {
      return [];
    }

    const snapshot = this.getEditorState(document);
    const request = this.contextBuilder.build(document, position);
    const cancellation = bridgeCancellation(token);
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
      return [new vscode.InlineCompletionItem(result.text, new vscode.Range(position, position))];
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
