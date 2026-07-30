import assert from "node:assert/strict";
import test from "node:test";
import { defaultRepositoryContextLimits, normalizeRepositoryContextLimits } from "../src/context/repository/repositoryContextConfig";

test("repository context limits stay bounded", () => {
  assert.deepEqual(normalizeRepositoryContextLimits({ enabled: true, maxFiles: 4, maxCharacters: 12000, timeoutMs: 40 }), { enabled: true, maxFiles: 4, maxCharacters: 12000, timeoutMs: 40 });
  assert.deepEqual(normalizeRepositoryContextLimits({ enabled: false, maxFiles: 99, maxCharacters: 1, timeoutMs: 999 }), { ...defaultRepositoryContextLimits, enabled: false });
});
