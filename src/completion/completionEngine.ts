import { CompletionRequest, CompletionResult } from "./completionTypes";

export interface CompletionEngine {
  complete(request: CompletionRequest, signal: AbortSignal): Promise<CompletionResult | null>;
}
