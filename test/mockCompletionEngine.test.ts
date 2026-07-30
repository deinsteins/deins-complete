import assert from "node:assert/strict";
import test from "node:test";
import { MockCompletionEngine } from "../src/completion/mockCompletionEngine";

const engine = new MockCompletionEngine();

test("mock engine completes a user assignment", async () => {
  const result = await engine.complete({ language: "typescript", filePath: "/test.ts", prefix: "const user =", suffix: "" }, new AbortController().signal);
  assert.deepEqual(result, { text: "await getUser();" });
});

test("mock engine completes console access", async () => {
  const result = await engine.complete({ language: "typescript", filePath: "/test.ts", prefix: "console.", suffix: "" }, new AbortController().signal);
  assert.deepEqual(result, { text: "log()" });
});

test("mock engine has no fallback completion", async () => {
  const result = await engine.complete({ language: "typescript", filePath: "/test.ts", prefix: "const user = await getUser();", suffix: "" }, new AbortController().signal);
  assert.equal(result, null);
});

test("mock engine respects an aborted request", async () => {
  const controller = new AbortController();
  controller.abort();
  const result = await engine.complete({ language: "typescript", filePath: "/test.ts", prefix: "const user =", suffix: "" }, controller.signal);
  assert.equal(result, null);
});
