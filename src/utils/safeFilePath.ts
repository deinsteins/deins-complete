import * as path from "node:path";
import * as vscode from "vscode";

export function getSafeFilePath(document: vscode.TextDocument): string {
  if (vscode.workspace.getWorkspaceFolder(document.uri) !== undefined) {
    return vscode.workspace.asRelativePath(document.uri, false).replace(/\\/g, "/");
  }
  return path.basename(document.uri.fsPath) || document.uri.path.split("/").pop() || "untitled";
}
