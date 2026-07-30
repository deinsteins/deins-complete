import {
  ApiCompletionRequest,
  ApiCompletionResponse,
  ApiLogger,
  BackendHealthResult,
  BackendSettingsProvider,
} from "./apiTypes";
import {
	ApiError,
  BackendUnavailableError,
  CancelledError,
  ConfigurationError,
  EndpointNotFoundError,
  ForbiddenError,
  InvalidRequestError,
  InvalidResponseError,
  NetworkError,
  PayloadTooLargeError,
  QuotaExceededError,
  RateLimitError,
  TimeoutError,
  UnauthorizedError,
} from "./apiErrors";
import { normalizeBackendUrl } from "./backendUrl";

export interface BackendClient {
  complete(request: ApiCompletionRequest, signal: AbortSignal): Promise<ApiCompletionResponse>;
  streamComplete?(request: ApiCompletionRequest, signal: AbortSignal): Promise<ApiCompletionResponse>;
  health(signal?: AbortSignal): Promise<BackendHealthResult>;
}

type FetchFunction = (input: string, init?: RequestInit) => Promise<Response>;

export class DeinsCompleteClient implements BackendClient {
  private token?: string;
  private unavailableUntil = 0;
  constructor(
    private readonly settings: BackendSettingsProvider,
    private readonly fetchFunction: FetchFunction = globalThis.fetch,
    private readonly logger?: ApiLogger,
    private readonly clientVersion = "0.0.0",
  ) {}

  async complete(request: ApiCompletionRequest, signal: AbortSignal): Promise<ApiCompletionResponse> {
    if (Date.now() < this.unavailableUntil) throw new RateLimitError("Backend completion is cooling down.");
    const startedAt = Date.now();
    const response = await this.send("/v1/completions", {
      method: "POST",
      headers: this.headers({ "Content-Type": "application/json", Accept: "application/json" }),
      body: JSON.stringify(request),
    }, signal);
    const payload = await this.parseCompletionResponse(response);
    this.logger?.debug(`Backend completion completed status=${response.status} durationMs=${Date.now() - startedAt} requestId=${payload.requestId ?? "none"}`);
    return payload;
  }
  async streamComplete(request: ApiCompletionRequest, signal: AbortSignal): Promise<ApiCompletionResponse> {
    const response = await this.send("/v1/completions/stream", { method: "POST", headers: this.headers({ "Content-Type": "application/json", Accept: "text/event-stream" }), body: JSON.stringify(request) }, signal);
    if (response.body === null) throw new InvalidResponseError("Backend stream is unavailable.", this.requestId(response));
    const startedAt = performance.now(); const reader = response.body.getReader(); const decoder = new TextDecoder(); let buffer = ""; let text = ""; let requestId = this.requestId(response); let firstChunkMs: number | undefined;
    for (;;) {
      const { done, value } = await reader.read(); buffer += decoder.decode(value, { stream: !done });
      const events = buffer.split("\n\n"); buffer = events.pop() ?? "";
      for (const event of events) {
        const type = event.match(/^event:\s*(.+)$/m)?.[1]; const data = event.match(/^data:\s*(.+)$/m)?.[1]; if (data === undefined) continue;
        let payload: unknown; try { payload = JSON.parse(data); } catch { throw new InvalidResponseError("Backend stream event is invalid.", requestId); }
        if (!isRecord(payload)) continue;
        if (type === "chunk" && typeof payload.text === "string") { if (firstChunkMs === undefined) firstChunkMs = Math.round(performance.now() - startedAt); text += payload.text; }
        if (type === "error") throw new BackendUnavailableError("Backend stream failed.", requestId);
        if (type === "done") { if (typeof payload.text === "string") text = payload.text; if (typeof payload.requestId === "string") requestId = payload.requestId; return { completion: { text }, requestId, metadata: { requestId }, streaming: { firstChunkMs } }; }
      }
      if (done) break;
    }
    throw new InvalidResponseError("Backend stream ended without completion.", requestId);
  }
  setInstallationToken(token: string | undefined): void { this.token = token; }
  async registerInstallation(installationId: string, signal?: AbortSignal): Promise<string> {
    const response = await this.send("/v1/installations/register", {
      method: "POST",
      headers: { "Content-Type": "application/json", Accept: "application/json" },
      body: JSON.stringify({ installationId, client: { name: "deinscomplete-vscode", version: this.clientVersion } }),
    }, signal);
    const payload=await this.parseJSON(response);if(!isRecord(payload)||!isRecord(payload.installation)||payload.installation.id!==installationId||typeof payload.token!=="string"||payload.token==="")throw new InvalidResponseError("Installation response is invalid.");return payload.token;
  }

  async health(signal?: AbortSignal): Promise<BackendHealthResult> {
    const startedAt = Date.now();
    const response = await this.send("/health", { headers: { Accept: "application/json" } }, signal);
    const payload = await this.parseJSON(response);
    if (!isRecord(payload) || payload.status !== "ok") {
      throw new InvalidResponseError("Backend health response is invalid.", this.requestId(response));
    }
    return { healthy: true, latencyMs: Date.now() - startedAt, requestId: this.requestId(response) };
  }

  private async send(path: string, init: RequestInit, signal?: AbortSignal): Promise<Response> {
    const backendUrl = normalizeBackendUrl(this.settings.getBackendUrl());
    if (backendUrl === null) {
      throw new ConfigurationError("Backend URL is invalid.");
    }
    const cancellation = createRequestCancellation(signal, this.settings.getBackendTimeoutMs());
    try {
      const response = await this.fetchFunction(`${backendUrl}${path}`, { ...init, signal: cancellation.signal });
      if (!response.ok) {
        const error = await statusError(response, this.requestId(response));
        if (error instanceof RateLimitError || error instanceof QuotaExceededError) this.unavailableUntil = Date.now() + (error.retryAfterSeconds ?? 1) * 1000;
        throw error;
      }
      return response;
    } catch (error) {
      if (error instanceof ApiError) {
        throw error;
      }
      if (signal?.aborted) {
        throw new CancelledError("Backend request cancelled.");
      }
      if (cancellation.timedOut()) {
        throw new TimeoutError(`Backend completion timed out after ${this.settings.getBackendTimeoutMs()}ms.`);
      }
      if (error instanceof Error && error.name === "AbortError") {
        throw new CancelledError("Backend request cancelled.");
      }
      throw new NetworkError("Backend request failed.");
    } finally {
      cancellation.dispose();
    }
  }

  private async parseCompletionResponse(response: Response): Promise<ApiCompletionResponse> {
    const payload = await this.parseJSON(response);
    if (!isRecord(payload) || !isRecord(payload.completion) || typeof payload.completion.text !== "string") {
      throw new InvalidResponseError("Backend completion response is invalid.", this.requestId(response));
    }
    const metadata = payload.metadata;
    if (metadata !== undefined && (!isRecord(metadata) || (metadata.requestId !== undefined && typeof metadata.requestId !== "string"))) {
      throw new InvalidResponseError("Backend completion response is invalid.", this.requestId(response));
    }
    const requestId = this.requestId(response) || (metadata as { requestId?: string } | undefined)?.requestId;
    return {
      completion: { text: payload.completion.text },
      metadata: metadata as ApiCompletionResponse["metadata"],
      requestId,
    };
  }

  private async parseJSON(response: Response): Promise<unknown> {
    try {
      return await response.json();
    } catch {
      throw new InvalidResponseError("Backend response is not valid JSON.", this.requestId(response));
    }
  }

  private requestId(response: Response): string | undefined {
    return response.headers.get("X-Request-ID") ?? undefined;
  }
  private headers(headers: Record<string,string>): Record<string,string> { return this.token ? {...headers,Authorization:`Bearer ${this.token}`} : headers; }
}

function createRequestCancellation(source: AbortSignal | undefined, timeoutMs: number): { signal: AbortSignal; timedOut(): boolean; dispose(): void } {
  const controller = new AbortController();
  let didTimeout = false;
  const timeout = setTimeout(() => { didTimeout = true; controller.abort(); }, timeoutMs);
  const abort = () => controller.abort();
  source?.addEventListener("abort", abort, { once: true });
  if (source?.aborted) {
    controller.abort();
  }
  return {
    signal: controller.signal,
    timedOut: () => didTimeout,
    dispose: () => { clearTimeout(timeout); source?.removeEventListener("abort", abort); },
  };
}

async function statusError(response: Response, requestId?: string): Promise<Error> {
  const status = response.status;
  const retry = Number(response.headers.get("Retry-After"));
  const retryAfter = Number.isFinite(retry) && retry > 0 ? retry : undefined;
  let code = "";
  try { const body = await response.clone().json() as { error?: { code?: string } }; code = body.error?.code ?? ""; } catch { /* status fallback */ }
  switch (status) {
    case 400: return new InvalidRequestError("Backend rejected the completion request.", requestId, status);
    case 401: return new UnauthorizedError("Backend authorization failed.", requestId, status);
    case 403: return new ForbiddenError("Backend request was forbidden.", requestId, status);
    case 404: return new EndpointNotFoundError("Backend completion endpoint was not found.", requestId, status);
    case 413: return new PayloadTooLargeError("Backend rejected the completion payload.", requestId, status);
    case 429: return code === "QUOTA_EXCEEDED" ? new QuotaExceededError("Daily completion quota exceeded.", requestId, status, retryAfter) : new RateLimitError("Backend is rate limited.", requestId, status, retryAfter);
    default: return new BackendUnavailableError("Backend completion request failed.", requestId, status);
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}
