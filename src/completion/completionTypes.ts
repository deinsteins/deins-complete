import { CompletionContext, RepositoryContext } from "../context/contextTypes";

export interface CompletionRequest extends CompletionContext {
  repositoryContext?: RepositoryContext;
}

export interface CompletionResult {
  text: string;
}
