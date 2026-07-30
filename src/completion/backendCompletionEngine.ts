import { ApiError, CancelledError, QuotaExceededError, RateLimitError, TimeoutError, UnauthorizedError } from "../api/apiErrors";
import { ApiCompletionRequest, ApiLogger } from "../api/apiTypes";
import { BackendClient } from "../api/deinsCompleteClient";
import { CompletionEngine } from "./completionEngine";
import { CompletionRequest, CompletionResult } from "./completionTypes";

export class BackendCompletionEngine implements CompletionEngine {
  constructor(
    private readonly client: BackendClient,
    private readonly extensionVersion: string,
    private readonly logger: ApiLogger,
    private readonly ensureAuthentication?: (signal: AbortSignal) => Promise<void>,
    private readonly refreshAuthentication?: (signal: AbortSignal) => Promise<void>,
    private readonly onQuotaExceeded?: () => void,
  ) {}

  async complete(request: CompletionRequest, signal: AbortSignal): Promise<CompletionResult | null> {
    try {
      await this.ensureAuthentication?.(signal);
      const response = await this.client.complete(toApiCompletionRequest(request, this.extensionVersion), signal);
      return response.completion.text === "" ? null : { text: response.completion.text };
    } catch (error) {
      if (error instanceof UnauthorizedError && this.refreshAuthentication !== undefined && !signal.aborted) {
        try {
          await this.refreshAuthentication(signal);
          const response = await this.client.complete(toApiCompletionRequest(request, this.extensionVersion), signal);
          return response.completion.text === "" ? null : { text: response.completion.text };
        } catch (refreshError) {
          if (refreshError instanceof CancelledError || signal.aborted) return null;
          this.logger.debug("Backend authentication refresh failed");
          return null;
        }
      }
      if (error instanceof CancelledError || signal.aborted) {
        this.logger.debug("Backend completion cancelled");
        return null;
      }
      if (error instanceof TimeoutError) {
        this.logger.debug("Backend completion timed out");
        return null;
      }
      if (error instanceof RateLimitError || error instanceof QuotaExceededError) {
        this.logger.debug(error instanceof QuotaExceededError ? "Backend daily quota exceeded" : "Backend completion rate limited");
        if (error instanceof QuotaExceededError) this.onQuotaExceeded?.();
        return null;
      }
      const requestId = error instanceof ApiError && error.requestId ? ` requestId=${error.requestId}` : "";
      this.logger.debug(`Backend completion unavailable${requestId}`);
      return null;
    }
  }
}

export function toApiCompletionRequest(request: CompletionRequest, version: string): ApiCompletionRequest {
  const apiRequest: ApiCompletionRequest = {
    context: {
      prefix: request.prefix,
      suffix: request.suffix,
      language: request.language,
      filePath: request.safeFilePath,
      cursorOffset: request.cursorOffset,
    },
    client: { name: "deinscomplete-vscode", version },
  };
  if (request.repositoryContext !== undefined) {
    apiRequest.repositoryContext = {
      files: request.repositoryContext.files,
      symbols: request.repositoryContext.symbols,
    };
  }
  return apiRequest;
}
