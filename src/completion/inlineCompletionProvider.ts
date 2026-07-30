import * as vscode from "vscode";
import { DeinsCompleteLifecycle } from "../core/lifecycle";
import { Logger } from "../logging/logger";
import { bridgeCancellation } from "../utils/cancellation";
import { CompletionEngine } from "./completionEngine";
import { CompletionRequest } from "./completionTypes";
import { EditorStateSnapshot, isCurrentEditorState } from "./editorState";

export class DeinsCompleteInlineCompletionProvider implements vscode.InlineCompletionItemProvider {
  constructor(
    private readonly lifecycle: DeinsCompleteLifecycle,
    private readonly engine: CompletionEngine,
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
    const request = this.createRequest(document, position);
    const cancellation = bridgeCancellation(token);
    this.logger.debug(`Inline completion requested (language=${request.language}, version=${snapshot.version})`);

    try {
      const result = await this.engine.complete(request, cancellation.signal);
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

  private createRequest(document: vscode.TextDocument, position: vscode.Position): CompletionRequest {
    const start = new vscode.Position(0, 0);
    const end = document.lineAt(document.lineCount - 1).range.end;
    return {
      language: document.languageId,
      filePath: document.uri.fsPath,
      prefix: document.getText(new vscode.Range(start, position)),
      suffix: document.getText(new vscode.Range(position, end)),
    };
  }

  private getEditorState(document: vscode.TextDocument): EditorStateSnapshot {
    return { uri: document.uri.toString(), version: document.version };
  }
}
