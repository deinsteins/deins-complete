import assert from "node:assert/strict";
import test from "node:test";
import { conflictingInlineCompletionExtensions } from "../src/completion/conflictDetection";

test("known inline completion extensions are reported without duplicate heuristics", () => {
  assert.deepEqual(conflictingInlineCompletionExtensions(["github.copilot", "publisher.unrelated", "CODEIUM.CODEIUM"]), ["GitHub Copilot", "Windsurf/Codeium"]);
});
