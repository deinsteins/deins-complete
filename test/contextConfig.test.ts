import assert from "node:assert/strict";
import test from "node:test";
import { defaultContextLimits, normalizeContextLimits } from "../src/context/contextConfig";

test("invalid context limits fall back to safe defaults", () => {
  assert.deepEqual(normalizeContextLimits({ maxPrefixCharacters: -1, maxSuffixCharacters: 10001 }), defaultContextLimits);
});
