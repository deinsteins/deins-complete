import { CompletionContext, RepositoryContext } from "../context/contextTypes";
import { CompletionRequestMode } from "./contextComplexity";

export interface CompletionRequest extends CompletionContext {
  repositoryContext?: RepositoryContext;
  repositoryContextTask?: Promise<RepositoryContext | undefined>;
  mode?: CompletionRequestMode;
}

export interface CompletionResult {
  text: string;
}
