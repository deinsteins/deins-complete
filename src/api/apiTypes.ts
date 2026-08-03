// Keep these thin contract types aligned with contracts/openapi.yaml.
export interface ApiCompletionRequest {
  context: {
    prefix: string;
    suffix: string;
    language: string;
    filePath: string;
    cursorOffset: number;
    style?: { indentation: "tabs" | "spaces"; indentSize?: number; quote?: "single" | "double"; semicolons?: "always" | "never" };
  };
  intent?: string;
  repositoryContext?: {
    files: Array<{ path: string; language: string; content: string; reason: string }>;
    symbols?: Array<{ name: string; kind: string; filePath: string; signature?: string }>;
    dependencies?: string[];
    focus?: string;
    diagnostics?: string[];
    signatureHelp?: { label: string; activeParameter?: number; parameter?: string };
  };
  client: {
    name: "deinscomplete-vscode";
    version: string;
  };
}

export interface ApiCompletionResponse {
  completion: { text: string };
  metadata?: { requestId?: string };
  requestId?: string;
  streaming?: { firstChunkMs?: number };
}

export interface BackendHealthResult {
  healthy: boolean;
  latencyMs: number;
  requestId?: string;
}

export interface BackendSettingsProvider {
  getBackendUrl(): string;
  getBackendTimeoutMs(): number;
}

export interface ApiLogger {
  debug(message: string): void;
}

export interface AccountTokens {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
}

export interface AccountDetails {
  user: { id: string; email: string; displayName?: string };
  plan: { code: string };
}

export interface AccountEntitlements {
  plan: string;
  features: { repositoryContext: boolean; streaming: boolean; premiumRouting: boolean };
  limits: { monthlyCompletions: number; used: number; remaining: number };
}

export interface AccountInstallation {
  id: string;
  status: string;
  createdAt: string;
  lastSeenAt?: string;
}

export interface QualityEvent {
  eventId: string;
  completionId: string;
  type: "shown" | "accepted" | "helpful" | "not-helpful";
  requestId?: string;
  language: string;
  framework: string;
  focus: string;
  mode: "fast" | "full";
  source: "backend" | "cache";
  latencyMs: number;
  clientVersion: string;
  feedbackReason?: "none" | "general" | "incorrect-api" | "irrelevant" | "too-slow" | "too-much-code" | "other";
}
