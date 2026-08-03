import { CompletionContext, RepositoryContext } from "../context/contextTypes";
import { CompletionFocus, CompletionRequestMode } from "./contextComplexity";

export interface CompletionRequest extends CompletionContext {
  repositoryContext?: RepositoryContext;
  repositoryContextTask?: Promise<RepositoryContext | undefined>;
  mode?: CompletionRequestMode;
  intent?: CompletionFocus;
}

export interface CompletionResult {
  text: string;
  requestId?: string;
  source?: "backend" | "cache";
  latencyMs?: number;
}
