import type * as vscode from "vscode";
import { ContextLimitsProvider } from "./contextConfig";
import { CompletionContext } from "./contextTypes";
import { ApproximateTokenEstimator, TokenEstimator } from "./tokenEstimator";
import { takePrefixWindow, takeSuffixWindow } from "./textWindow";
import { inferCodeStyle } from "./codeStyle";

export class ContextBuilder {
  constructor(
    private readonly limits: ContextLimitsProvider,
    private readonly tokenEstimator: TokenEstimator = new ApproximateTokenEstimator(),
    private readonly safeFilePath: (document: vscode.TextDocument) => string = (document) => document.uri.fsPath,
  ) {}

  build(document: vscode.TextDocument, position: vscode.Position): CompletionContext {
    const startedAt = Date.now();
    const documentText = document.getText();
    const cursorOffset = document.offsetAt(position);
    const limits = this.limits.getContextLimits();
    const prefixWindow = takePrefixWindow(documentText.slice(0, cursorOffset), limits.maxPrefixCharacters);
    const suffixWindow = takeSuffixWindow(documentText.slice(cursorOffset), limits.maxSuffixCharacters);
    const currentLine = document.lineAt(position.line).text;
    const imports = extractImportBlock(documentText);
    const preservedImports = imports !== "" && !containsImport(prefixWindow.text) ? imports : undefined;

    return {
      prefix: prefixWindow.text,
      suffix: suffixWindow.text,
      language: document.languageId,
      filePath: document.uri.fsPath,
      safeFilePath: this.safeFilePath(document),
      cursorOffset,
      documentVersion: document.version,
      currentLine,
      textBeforeCursorOnLine: currentLine.slice(0, position.character),
      textAfterCursorOnLine: currentLine.slice(position.character),
      indentation: currentLine.match(/^[\t ]*/)?.[0] ?? "",
      imports: preservedImports,
      style: inferCodeStyle(documentText),
      metadata: {
        totalDocumentCharacters: documentText.length,
        prefixCharacters: prefixWindow.text.length,
        suffixCharacters: suffixWindow.text.length,
        truncatedPrefix: prefixWindow.truncated,
        truncatedSuffix: suffixWindow.truncated,
        estimatedPrefixTokens: this.tokenEstimator.estimate(prefixWindow.text),
        estimatedSuffixTokens: this.tokenEstimator.estimate(suffixWindow.text),
        estimatedTotalTokens: this.tokenEstimator.estimate(`${prefixWindow.text}${suffixWindow.text}${preservedImports ?? ""}`),
        buildDurationMilliseconds: Date.now() - startedAt,
      },
    };
  }
}

function containsImport(text: string): boolean {
  return /(^|\n)\s*(?:import\s|from\s+.+\s+import\s|const\s+\w+\s*=\s*require\s*\()/m.test(text);
}

function extractImportBlock(text: string): string {
  const lines = text.slice(0, 10000).split(/\r?\n/);
  const imports: string[] = [];
  let inGoImportBlock = false;

  for (const line of lines) {
    const trimmed = line.trim();
    if (inGoImportBlock) {
      imports.push(line);
      if (trimmed === ")") {
        inGoImportBlock = false;
      }
      continue;
    }
    if (trimmed === "" || trimmed.startsWith("//") || trimmed.startsWith("#")) {
      continue;
    }
    if (trimmed === "import (") {
      imports.push(line);
      inGoImportBlock = true;
      continue;
    }
    if (/^(import\s|from\s+.+\s+import\s|const\s+\w+\s*=\s*require\s*\()/.test(trimmed)) {
      imports.push(line);
      continue;
    }
    break;
  }

  return imports.join("\n");
}
