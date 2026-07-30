import { ApiError, CancelledError, TimeoutError } from "../api/apiErrors";
import { ApiCompletionRequest, ApiLogger } from "../api/apiTypes";
import { BackendClient } from "../api/deinsCompleteClient";
import { CompletionEngine } from "./completionEngine";
import { CompletionRequest, CompletionResult } from "./completionTypes";

export class BackendCompletionEngine implements CompletionEngine {
  constructor(
    private readonly client: BackendClient,
    private readonly extensionVersion: string,
    private readonly logger: ApiLogger,
  ) {}

  async complete(request: CompletionRequest, signal: AbortSignal): Promise<CompletionResult | null> {
    try {
      const response = await this.client.complete(toApiCompletionRequest(request, this.extensionVersion), signal);
      return response.completion.text === "" ? null : { text: response.completion.text };
    } catch (error) {
      if (error instanceof CancelledError || signal.aborted) {
        this.logger.debug("Backend completion cancelled");
        return null;
      }
      if (error instanceof TimeoutError) {
        this.logger.debug("Backend completion timed out");
        return null;
      }
      const requestId = error instanceof ApiError && error.requestId ? ` requestId=${error.requestId}` : "";
      this.logger.debug(`Backend completion unavailable${requestId}`);
      return null;
    }
  }
}

export function toApiCompletionRequest(request: CompletionRequest, version: string): ApiCompletionRequest {
  return {
    context: {
      prefix: request.prefix,
      suffix: request.suffix,
      language: request.language,
      filePath: request.safeFilePath,
      cursorOffset: request.cursorOffset,
    },
    client: { name: "deinscomplete-vscode", version },
  };
}
