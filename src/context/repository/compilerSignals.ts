import type * as vscode from "vscode";
import { CompletionContext, SignatureHelpContext } from "../contextTypes";

const ignoredIdentifiers = new Set(["if", "for", "while", "switch", "return", "const", "let", "var", "new", "await", "function", "class", "interface", "type"]);

export function identifiersForDefinition(context: CompletionContext): Array<{ name: string; offset: number }> {
  const tail = context.prefix.slice(-400);
  const start = context.cursorOffset - tail.length;
  const matches = [...tail.matchAll(/[A-Za-z_$][\w$]*/g)].reverse();
  const seen = new Set<string>();
  const result: Array<{ name: string; offset: number }> = [];
  for (const match of matches) {
    const name = match[0];
    if (ignoredIdentifiers.has(name) || seen.has(name)) continue;
    seen.add(name);
    result.push({ name, offset: start + (match.index ?? 0) + Math.min(1, name.length - 1) });
    if (result.length === 2) break;
  }
  return result;
}

export function compactSignatureHelp(help: vscode.SignatureHelp | undefined): SignatureHelpContext | undefined {
  if (help === undefined || help.signatures.length === 0) return undefined;
  const signature = help.signatures[Math.min(Math.max(help.activeSignature, 0), help.signatures.length - 1)];
  const activeParameter = Math.min(Math.max(help.activeParameter, 0), Math.max(0, signature.parameters.length - 1));
  const parameter = signature.parameters[activeParameter];
  const label = signature.label.replace(/\s+/g, " ").slice(0, 500);
  if (label === "") return undefined;
  let parameterLabel: string | undefined;
  if (parameter !== undefined) {
    parameterLabel = typeof parameter.label === "string"
      ? parameter.label
      : signature.label.slice(parameter.label[0], parameter.label[1]);
  }
  return { label, ...(signature.parameters.length > 0 ? { activeParameter } : {}), ...(parameterLabel ? { parameter: parameterLabel.slice(0, 200) } : {}) };
}
