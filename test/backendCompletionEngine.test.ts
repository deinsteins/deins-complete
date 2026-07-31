import assert from "node:assert/strict";
import test from "node:test";
import { AccountRequiredError, CancelledError, EndpointNotFoundError, NetworkError } from "../src/api/apiErrors";
import { ApiCompletionResponse } from "../src/api/apiTypes";
import { BackendClient } from "../src/api/deinsCompleteClient";
import { BackendCompletionEngine, toApiCompletionRequest } from "../src/completion/backendCompletionEngine";
import { CompletionRequest } from "../src/completion/completionTypes";

const request: CompletionRequest = {
  language: "typescript", filePath: "/private/home/project/test.ts", safeFilePath: "src/test.ts", prefix: "const user =", suffix: "",
  cursorOffset: 12, documentVersion: 1, currentLine: "const user =", textBeforeCursorOnLine: "const user =", textAfterCursorOnLine: "", indentation: "",
  metadata: { totalDocumentCharacters: 12, prefixCharacters: 12, suffixCharacters: 0, truncatedPrefix: false, truncatedSuffix: false, estimatedPrefixTokens: 3, estimatedSuffixTokens: 0, estimatedTotalTokens: 3, buildDurationMilliseconds: 0 },
};

class TestClient implements BackendClient {
  constructor(private readonly result: ApiCompletionResponse | Error) {}
  async complete(): Promise<ApiCompletionResponse> {
    if (this.result instanceof Error) { throw this.result; }
    return this.result;
  }
  async health() { return { healthy: true, latencyMs: 1 }; }
}

const logger = { debug: () => undefined };

test("backend completion engine maps backend text", async () => {
  const engine = new BackendCompletionEngine(new TestClient({ completion: { text: "await getUser();" } }), "0.0.1", logger);
  assert.deepEqual(await engine.complete(request, new AbortController().signal), { text: "await getUser();" });
});

test("backend completion engine maps empty completion to null", async () => {
  const engine = new BackendCompletionEngine(new TestClient({ completion: { text: "" } }), "0.0.1", logger);
  assert.equal(await engine.complete(request, new AbortController().signal), null);
});

test("backend completion engine prefers streaming and falls back when unavailable", async () => {
  let standardCalls = 0;
  const streamClient: BackendClient = {
    complete: async () => { standardCalls++; return { completion: { text: "standard" } }; },
    streamComplete: async () => ({ completion: { text: "streamed" } }),
    health: async () => ({ healthy: true, latencyMs: 1 }),
  };
  assert.deepEqual(await new BackendCompletionEngine(streamClient, "0.0.1", logger).complete(request, new AbortController().signal), { text: "streamed" });
  const fallbackClient: BackendClient = { ...streamClient, streamComplete: async () => { throw new EndpointNotFoundError("not enabled"); } };
  assert.deepEqual(await new BackendCompletionEngine(fallbackClient, "0.0.1", logger).complete(request, new AbortController().signal), { text: "standard" });
  assert.equal(standardCalls, 1);
});

test("backend completion engine handles backend and cancellation errors silently", async () => {
  const offline = new BackendCompletionEngine(new TestClient(new NetworkError("offline")), "0.0.1", logger);
  const cancelled = new BackendCompletionEngine(new TestClient(new CancelledError("cancelled")), "0.0.1", logger);
  assert.equal(await offline.complete(request, new AbortController().signal), null);
  assert.equal(await cancelled.complete(request, new AbortController().signal), null);
});

test("account-required response does not reset installation authentication", async () => {
  let refreshes = 0;
  const engine = new BackendCompletionEngine(
    new TestClient(new AccountRequiredError("sign in required")),
    "0.0.1",
    logger,
    undefined,
    async () => { refreshes++; },
  );
  assert.equal(await engine.complete(request, new AbortController().signal), null);
  assert.equal(refreshes, 0);
});

test("API request mapping sends only contract fields and a safe path", () => {
  assert.deepEqual(toApiCompletionRequest(request, "0.0.1"), {
    context: { prefix: "const user =", suffix: "", language: "typescript", filePath: "src/test.ts", cursorOffset: 12 },
    client: { name: "deinscomplete-vscode", version: "0.0.1" },
  });
});

test("API request mapping includes bounded repository context but not its cache fingerprint", () => {
  const withRepository: CompletionRequest = {
    ...request,
    repositoryContext: {
      files: [{ path: "src/types/user.ts", language: "typescript", content: "export interface User { id: string }", reason: "import" }],
      symbols: [{ name: "User", kind: "interface", filePath: "src/types/user.ts", signature: "export interface User" }],
      dependencies: ["@mui/material"],
      focus: "component-props",
      fingerprint: "private-cache-key",
      durationMs: 12,
      timedOut: false,
    },
  };
  assert.deepEqual(toApiCompletionRequest(withRepository, "0.0.1").repositoryContext, {
    files: [{ path: "src/types/user.ts", language: "typescript", content: "export interface User { id: string }", reason: "import" }],
    symbols: [{ name: "User", kind: "interface", filePath: "src/types/user.ts", signature: "export interface User" }],
    dependencies: ["@mui/material"],
    focus: "component-props",
  });
  assert.equal(JSON.stringify(toApiCompletionRequest(withRepository, "0.0.1")).includes("private-cache-key"), false);
});

test("API request mapping uses smaller member-access context budgets", () => {
  const large: CompletionRequest = { ...request, prefix: "p".repeat(5000), suffix: "s".repeat(3000), repositoryContext: { files: [{ path: "a.ts", language: "ts", content: "r".repeat(5000), reason: "import" }], focus: "member-access", fingerprint: "x", durationMs: 1, timedOut: false } };
  const mapped = toApiCompletionRequest(large, "0.0.1");
  assert.equal(mapped.context.prefix.length, 1500); assert.equal(mapped.context.suffix.length, 500); assert.equal(mapped.repositoryContext?.files[0].content.length, 2000);
});

test("API request mapping adapts fast and timed-out context budgets", () => {
  const fast = toApiCompletionRequest({ ...request, mode: "fast", prefix: "p".repeat(5000), suffix: "s".repeat(3000) }, "0.0.1");
  assert.equal(fast.context.prefix.length, 1800);
  assert.equal(fast.context.suffix.length, 700);

  const timedOut = toApiCompletionRequest({
    ...request,
    mode: "full",
    repositoryContext: {
      files: [{ path: "a.ts", language: "ts", content: "r".repeat(5000), reason: "import" }],
      focus: "component-props",
      fingerprint: "x",
      durationMs: 40,
      timedOut: true,
    },
  }, "0.0.1");
  assert.equal(timedOut.repositoryContext?.files[0].content.length, 2000);
});
