import * as vscode from "vscode";

export interface AutoImportPlan { uri: vscode.Uri; edits: vscode.TextEdit[]; }

/** Uses only language-provider supplied import edits; it never synthesizes imports. */
export class AutoImportResolver {
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
}
function label(item: vscode.CompletionItem): string { return typeof item.label === "string" ? item.label : item.label.label; }
