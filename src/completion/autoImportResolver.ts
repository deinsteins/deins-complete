import * as vscode from "vscode";

export interface AutoImportPlan { uri: vscode.Uri; edits: vscode.TextEdit[]; }

/** Uses only language-provider supplied import edits; it never synthesizes imports. */
export class AutoImportResolver {
  private readonly prefetched = new Map<string, Map<string, AutoImportPlan>>();

  canResolve(completion: string): boolean { return /^\s*[A-Za-z_$][\w$]*/.test(completion); }

  async prefetch(document: vscode.TextDocument, position: vscode.Position): Promise<void> {
    if (!["javascript", "javascriptreact", "typescript", "typescriptreact"].includes(document.languageId)) return;
    const version = document.version;
    const key = cacheKey(document, position);
    try {
      const list = await vscode.commands.executeCommand<vscode.CompletionList>("vscode.executeCompletionItemProvider", document.uri, position);
      if (document.version !== version) return;
      const plans = new Map<string, AutoImportPlan>();
      for (const item of list?.items ?? []) {
        if (item.additionalTextEdits !== undefined && item.additionalTextEdits.length > 0) {
          plans.set(label(item), { uri: document.uri, edits: item.additionalTextEdits });
        }
      }
      this.prefetched.delete(key);
      this.prefetched.set(key, plans);
      while (this.prefetched.size > 20) this.prefetched.delete(this.prefetched.keys().next().value!);
    } catch {
      // Language providers are opportunistic and must never block completion.
    }
  }

  getPrefetched(document: vscode.TextDocument, position: vscode.Position, completion: string): AutoImportPlan | undefined {
    const identifier = completion.match(/^\s*([A-Za-z_$][\w$]*)/)?.[1];
    return identifier === undefined ? undefined : this.prefetched.get(cacheKey(document, position))?.get(identifier);
  }

  async resolve(document: vscode.TextDocument, position: vscode.Position, completion: string): Promise<AutoImportPlan | undefined> {
    const identifier = completion.match(/^\s*([A-Za-z_$][\w$]*)/)?.[1];
    if (identifier === undefined) return undefined;
    const list = await vscode.commands.executeCommand<vscode.CompletionList>("vscode.executeCompletionItemProvider", document.uri, position);
    const item = list?.items.find((candidate) => label(candidate) === identifier && candidate.additionalTextEdits !== undefined && candidate.additionalTextEdits.length > 0);
    return item?.additionalTextEdits === undefined ? undefined : { uri: document.uri, edits: item.additionalTextEdits };
  }

  async apply(plan: AutoImportPlan): Promise<void> {
    const edit = new vscode.WorkspaceEdit(); edit.set(plan.uri, plan.edits); await vscode.workspace.applyEdit(edit);
  }

  async resolveAndApply(uri: vscode.Uri, position: vscode.Position, completion: string): Promise<void> {
    const document = await vscode.workspace.openTextDocument(uri);
    const plan = await this.resolve(document, position, completion);
    if (plan !== undefined) await this.apply(plan);
  }
}
function label(item: vscode.CompletionItem): string { return typeof item.label === "string" ? item.label : item.label.label; }
function cacheKey(document: vscode.TextDocument, position: vscode.Position): string {
  return `${document.uri.toString()}:${document.version}:${position.line}:${position.character}`;
}
