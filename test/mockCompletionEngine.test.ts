import assert from "node:assert/strict";
import test from "node:test";
import { MockCompletionEngine } from "../src/completion/mockCompletionEngine";
import { CompletionRequest } from "../src/completion/completionTypes";

const engine = new MockCompletionEngine();
const request = (prefix: string): CompletionRequest => ({
  language: "typescript",
  filePath: "/test.ts",
  prefix,
  suffix: "",
  cursorOffset: prefix.length,
  documentVersion: 1,
  currentLine: prefix,
  textBeforeCursorOnLine: prefix,
  textAfterCursorOnLine: "",
  indentation: "",
  metadata: {
    totalDocumentCharacters: prefix.length,
    prefixCharacters: prefix.length,
    suffixCharacters: 0,
    truncatedPrefix: false,
    truncatedSuffix: false,
    estimatedPrefixTokens: 0,
    estimatedSuffixTokens: 0,
    estimatedTotalTokens: 0,
    buildDurationMilliseconds: 0,
  },
});

test("mock engine completes a user assignment", async () => {
  const result = await engine.complete(request("const user ="), new AbortController().signal);
  assert.deepEqual(result, { text: "await getUser();" });
});

test("mock engine completes console access", async () => {
  const result = await engine.complete(request("console."), new AbortController().signal);
  assert.deepEqual(result, { text: "log()" });
});

test("mock engine has no fallback completion", async () => {
  const result = await engine.complete(request("const user = await getUser();"), new AbortController().signal);
  assert.equal(result, null);
});

test("mock engine respects an aborted request", async () => {
  const controller = new AbortController();
  controller.abort();
  const result = await engine.complete(request("const user ="), controller.signal);
  assert.equal(result, null);
});
