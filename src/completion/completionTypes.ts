export interface CompletionRequest {
  language: string;
  filePath: string;
  prefix: string;
  suffix: string;
}

export interface CompletionResult {
  text: string;
}
