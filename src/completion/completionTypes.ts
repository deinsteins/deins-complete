import { CompletionContext } from "../context/contextTypes";

export interface CompletionRequest extends CompletionContext {}

export interface CompletionResult {
  text: string;
}
