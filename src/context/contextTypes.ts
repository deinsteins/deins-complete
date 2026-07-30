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
  metadata: CompletionContextMetadata;
}
