export interface CompletionContextMetadata {
  totalDocumentCharacters: number;
  prefixCharacters: number;
  suffixCharacters: number;
  truncatedPrefix: boolean;
  truncatedSuffix: boolean;
  estimatedPrefixTokens: number;
  estimatedSuffixTokens: number;
  estimatedTotalTokens: number;
  buildDurationMilliseconds: number;
}

export interface CodeStyle {
  indentation: "tabs" | "spaces";
  indentSize?: number;
  quote?: "single" | "double";
  semicolons?: "always" | "never";
}

export interface SignatureHelpContext {
  label: string;
  activeParameter?: number;
  parameter?: string;
}

export interface CompletionContext {
  prefix: string;
  suffix: string;
  language: string;
  filePath: string;
  safeFilePath: string;
  cursorOffset: number;
  documentVersion: number;
  currentLine: string;
  textBeforeCursorOnLine: string;
  textAfterCursorOnLine: string;
  indentation: string;
  imports?: string;
  style?: CodeStyle;
  metadata: CompletionContextMetadata;
}

export type RepositoryContextReason = "import" | "open-file" | "recent-file" | "symbol-reference" | "framework-config";

export interface RepositoryContextFile {
  path: string;
  language: string;
  content: string;
  reason: RepositoryContextReason;
}

export interface RepositorySymbol {
  name: string;
  kind: string;
  filePath: string;
  signature?: string;
}

// fingerprint is extension-only: it participates in the completion cache key
// but is deliberately not sent to the API/provider.
export interface RepositoryContext {
  files: RepositoryContextFile[];
  symbols?: RepositorySymbol[];
  dependencies?: string[];
  focus?: string;
  diagnostics?: string[];
  signatureHelp?: SignatureHelpContext;
  fingerprint: string;
  durationMs: number;
  timedOut: boolean;
}
