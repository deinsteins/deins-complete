import assert from "node:assert/strict";
import test from "node:test";
import { normalizeBackendUrl } from "../src/api/backendUrl";

test("backend URL normalization removes trailing slashes and rejects invalid protocols", () => {
  assert.equal(normalizeBackendUrl("http://127.0.0.1:3001/"), "http://127.0.0.1:3001");
  assert.equal(normalizeBackendUrl("ftp://example.com"), null);
});
