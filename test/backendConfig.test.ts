import assert from "node:assert/strict";
import test from "node:test";
import { defaultBackendSettings, normalizeBackendTimeout } from "../src/config/backendConfig";

test("backend timeout normalization retains only supported values", () => {
  assert.equal(normalizeBackendTimeout(500), 500);
  assert.equal(normalizeBackendTimeout(30001), defaultBackendSettings.timeoutMs);
});
