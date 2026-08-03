import { randomUUID } from "node:crypto";
import { BackendClient } from "../api/deinsCompleteClient";
import { QualityEvent } from "../api/apiTypes";
import { CompletionRequest, CompletionResult } from "../completion/completionTypes";
import { completionFocus } from "../completion/contextComplexity";

export interface QualitySettings { qualityInsightsEnabled(): boolean }
export interface QualityLog { debug(message: string): void }
export type QualityCompletion = Omit<QualityEvent, "eventId" | "type">;

/** Sends bounded outcome metadata only; failures never enter the completion path. */
export class QualityReporter {
  private available = true;

  constructor(private readonly settings: QualitySettings, private readonly client: BackendClient, private readonly logger?: QualityLog) {}

  shown(request: CompletionRequest, result: CompletionResult): QualityCompletion | undefined {
    if (!this.enabled()) return undefined;
    const context: QualityCompletion = {
      completionId: randomUUID(),
      ...(safeRequestID(result.requestId) === undefined ? {} : { requestId: safeRequestID(result.requestId) }),
      language: category(request.language),
      framework: framework(request),
      focus: category(completionFocus(request)),
      mode: request.mode === "full" ? "full" : "fast",
      source: result.source === "cache" ? "cache" : "backend",
      latencyMs: Math.min(30000, Math.max(0, Math.round(result.latencyMs ?? 0))),
    };
    this.send({ ...context, eventId: randomUUID(), type: "shown" });
    return context;
  }

  accepted(context?: QualityCompletion): void {
    if (context === undefined || !this.enabled()) return;
    this.send({ ...context, eventId: randomUUID(), type: "accepted" });
  }

  private enabled(): boolean { return this.available && this.settings.qualityInsightsEnabled() && this.client.sendQualityEvent !== undefined; }
  private send(event: QualityEvent): void {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 1500);
    void this.client.sendQualityEvent?.(event, controller.signal).catch((error: unknown) => {
      if (error instanceof Error && error.name === "EndpointNotFoundError") this.available = false;
      this.logger?.debug(`Quality insight delivery skipped type=${event.type}`);
    }).finally(() => clearTimeout(timeout));
  }
}

function category(value: string): string {
  const normalized = value.toLowerCase().replace(/[^a-z0-9+.#_-]/g, "-").replace(/^-+|-+$/g, "").slice(0, 40);
  return normalized || "other";
}
function safeRequestID(value?: string): string | undefined { return value !== undefined && /^[a-zA-Z0-9_-]{1,128}$/.test(value) ? value : undefined; }
function framework(request: CompletionRequest): string {
  const dependencies = request.repositoryContext?.dependencies ?? [];
  const joined = dependencies.join(" ").toLowerCase();
  if (joined.includes("@mui/") || joined.includes("material-ui")) return "mui";
  if (joined.includes("antd")) return "antd";
  if (joined.includes("tailwind")) return "tailwind";
  if (joined.includes("next")) return "next";
  if (joined.includes("vue")) return "vue";
  if (joined.includes("svelte")) return "svelte";
  if (joined.includes("angular")) return "angular";
  if (joined.includes("react") || /react$/.test(request.language)) return "react";
  return "none";
}
