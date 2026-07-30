import { CompletionEngine } from "./completionEngine";
import { CompletionRequest, CompletionResult } from "./completionTypes";

export class MockCompletionEngine implements CompletionEngine {
  async complete(request: CompletionRequest, signal: AbortSignal): Promise<CompletionResult | null> {
    if (signal.aborted) {
      return null;
    }

    if (request.prefix.endsWith("const user =")) {
      return { text: "await getUser();" };
    }
    if (request.prefix.endsWith("console.")) {
      return { text: "log()" };
    }
    return null;
  }
}
