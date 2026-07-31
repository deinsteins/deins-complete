import assert from "node:assert/strict";
import test from "node:test";
import { BackendUnavailableError, CancelledError, EndpointNotFoundError, ForbiddenError, InvalidRequestError, InvalidResponseError, NetworkError, PayloadTooLargeError, RateLimitError, TimeoutError, UnauthorizedError } from "../src/api/apiErrors";
import { ApiCompletionRequest, BackendSettingsProvider } from "../src/api/apiTypes";
import { DeinsCompleteClient } from "../src/api/deinsCompleteClient";

const settings: BackendSettingsProvider = { getBackendUrl: () => "http://127.0.0.1:3001/", getBackendTimeoutMs: () => 500 };
const request: ApiCompletionRequest = { context: { prefix: "const user =", suffix: "", language: "typescript", filePath: "test.ts", cursorOffset: 12 }, client: { name: "deinscomplete-vscode", version: "0.0.1" } };

function response(status: number, payload: unknown, requestId = "request-1"): Response {
  return { ok: status >= 200 && status < 300, status, headers: { get: (name: string) => name === "X-Request-ID" ? requestId : null }, json: async () => payload } as unknown as Response;
}

test("client posts completion request and validates response", async () => {
  let url = "";
  const client = new DeinsCompleteClient(settings, async (input, init) => {
    url = input;
    assert.equal(init?.method, "POST");
    return response(200, { completion: { text: "await getUser();" }, metadata: { requestId: "request-1" } });
  });
  assert.deepEqual(await client.complete(request, new AbortController().signal), { completion: { text: "await getUser();" }, metadata: { requestId: "request-1" }, requestId: "request-1" });
  assert.equal(url, "http://127.0.0.1:3001/v1/completions");
});

test("client uses metadata request ID when the response header is absent", async () => {
  const client = new DeinsCompleteClient(settings, async () => response(200, { completion: { text: "log()" }, metadata: { requestId: "metadata-id" } }, ""));
  assert.equal((await client.complete(request, new AbortController().signal)).requestId, "metadata-id");
});

test("client normalizes HTTP and invalid response errors", async () => {
  const errors = [
    [400, InvalidRequestError], [401, UnauthorizedError], [403, ForbiddenError], [404, EndpointNotFoundError],
    [413, PayloadTooLargeError], [429, RateLimitError], [500, BackendUnavailableError],
  ] as const;
  for (const [status, errorType] of errors) {
    const client = new DeinsCompleteClient(settings, async () => response(status, {}));
    await assert.rejects(() => client.complete(request, new AbortController().signal), errorType);
  }
  const malformed = new DeinsCompleteClient(settings, async () => response(200, {}));
  await assert.rejects(() => malformed.complete(request, new AbortController().signal), InvalidResponseError);
});

test("client handles empty completions, malformed JSON, network failures, and health", async () => {
  const empty = new DeinsCompleteClient(settings, async () => response(200, { completion: { text: "" } }));
  const malformedJSON = new DeinsCompleteClient(settings, async () => ({ ...response(200, {}), json: async () => { throw new Error("bad JSON"); } } as Response));
  const offline = new DeinsCompleteClient(settings, async () => { throw new Error("offline"); });
  const healthy = new DeinsCompleteClient(settings, async () => response(200, { status: "ok" }));
  assert.equal((await empty.complete(request, new AbortController().signal)).completion.text, "");
  await assert.rejects(() => malformedJSON.complete(request, new AbortController().signal), InvalidResponseError);
  await assert.rejects(() => offline.complete(request, new AbortController().signal), NetworkError);
  assert.equal((await healthy.health()).healthy, true);
});

test("client aborts a timed out fetch", async () => {
  const client = new DeinsCompleteClient(settings, async (_input, init) => new Promise<Response>((_resolve, reject) => {
    init?.signal?.addEventListener("abort", () => reject(Object.assign(new Error("aborted"), { name: "AbortError" })));
  }));
  await assert.rejects(() => client.complete(request, new AbortController().signal), TimeoutError);
});

test("client preserves explicit cancellation", async () => {
  const controller = new AbortController();
  controller.abort();
  const client = new DeinsCompleteClient(settings, async () => { throw Object.assign(new Error("aborted"), { name: "AbortError" }); });
  await assert.rejects(() => client.complete(request, controller.signal), CancelledError);
});

test("client applies a short cooldown after infrastructure 503", async () => {
  let calls = 0;
  const client = new DeinsCompleteClient(settings, async () => {
    calls++;
    return response(503, { error: { code: "SERVICE_UNAVAILABLE" } });
  });
  await assert.rejects(() => client.complete(request, new AbortController().signal), BackendUnavailableError);
  await assert.rejects(() => client.complete(request, new AbortController().signal), BackendUnavailableError);
  assert.equal(calls, 1);
});

test("client supports the separate magic-code account authentication contract", async () => {
  const calls: Array<{ url: string; headers?: unknown; body?: unknown }> = [];
  const client = new DeinsCompleteClient(settings, async (url, init) => {
    calls.push({ url, headers: init?.headers, body: init?.body });
    if (url.endsWith("/magic/verify")) return response(200, { accessToken: "access", refreshToken: "refresh", expiresIn: 1800 });
    if (url.endsWith("/account/entitlements")) return response(200, { plan: "free", features: { repositoryContext: false, streaming: true, premiumRouting: false }, limits: { monthlyCompletions: 2000, used: 1, remaining: 1999 } });
    return response(204, {});
  });
  await client.requestMagicCode("user@example.com", "invite-code");
  assert.deepEqual(await client.verifyMagicCode("user@example.com", "123456"), { accessToken: "access", refreshToken: "refresh", expiresIn: 1800 });
  assert.equal((await client.getEntitlements("access")).limits.remaining, 1999);
  await client.linkInstallation("access", "installation-token");
  assert.equal(calls[2].url, "http://127.0.0.1:3001/v1/account/entitlements");
  assert.equal((calls[3].headers as Record<string, string>)["X-DeinsComplete-Installation-Token"], "installation-token");
});
