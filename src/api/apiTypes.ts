// Keep these thin contract types aligned with contracts/openapi.yaml.
export interface ApiCompletionRequest {
  context: {
    prefix: string;
    suffix: string;
    language: string;
    filePath: string;
    cursorOffset: number;
  };
  repositoryContext?: {
    files: Array<{ path: string; language: string; content: string; reason: string }>;
    symbols?: Array<{ name: string; kind: string; filePath: string; signature?: string }>;
    dependencies?: string[];
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
