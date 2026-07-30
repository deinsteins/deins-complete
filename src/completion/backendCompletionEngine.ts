import { ApiError, CancelledError, EndpointNotFoundError, QuotaExceededError, RateLimitError, TimeoutError, UnauthorizedError } from "../api/apiErrors";
import { ApiCompletionRequest, ApiLogger } from "../api/apiTypes";
import { BackendClient } from "../api/deinsCompleteClient";
import { CompletionEngine } from "./completionEngine";
import { CompletionRequest, CompletionResult } from "./completionTypes";

export class BackendCompletionEngine implements CompletionEngine {
  private readonly stats = { streamsStarted: 0, streamsSucceeded: 0, streamsFallback: 0, totalFirstChunkMs: 0, firstChunkSamples: 0 };
  constructor(
    private readonly client: BackendClient,
    private readonly extensionVersion: string,
    private readonly logger: ApiLogger,
    private readonly ensureAuthentication?: (signal: AbortSignal) => Promise<void>,
    private readonly refreshAuthentication?: (signal: AbortSignal) => Promise<void>,
    private readonly onQuotaExceeded?: () => void,
    private readonly streamingEnabled: () => boolean = () => true,
  ) {}

  async complete(request: CompletionRequest, signal: AbortSignal): Promise<CompletionResult | null> {
    try {
      await this.ensureAuthentication?.(signal);
      const response = await this.completeRequest(request, signal);
      return response.completion.text === "" ? null : { text: response.completion.text };
    } catch (error) {
      if (error instanceof UnauthorizedError && this.refreshAuthentication !== undefined && !signal.aborted) {
        try {
          await this.refreshAuthentication(signal);
          const response = await this.completeRequest(request, signal);
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
  getStats() { return { ...this.stats }; }

  private async completeRequest(request: CompletionRequest, signal: AbortSignal) {
    const apiRequest = toApiCompletionRequest(request, this.extensionVersion);
    if (this.streamingEnabled() && this.client.streamComplete !== undefined) {
      this.stats.streamsStarted++;
      try { const response = await this.client.streamComplete(apiRequest, signal); this.stats.streamsSucceeded++; if (response.streaming?.firstChunkMs !== undefined) { this.stats.totalFirstChunkMs += response.streaming.firstChunkMs; this.stats.firstChunkSamples++; } return response; } catch (error) { if (!(error instanceof EndpointNotFoundError)) throw error; this.stats.streamsFallback++; this.logger.debug("Backend streaming unavailable; using standard completion"); }
    }
    return this.client.complete(apiRequest, signal);
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
      ...(request.repositoryContext.dependencies !== undefined ? { dependencies: request.repositoryContext.dependencies } : {}),
      ...(request.repositoryContext.focus !== undefined ? { focus: request.repositoryContext.focus } : {}),
    };
  }
  return apiRequest;
}
