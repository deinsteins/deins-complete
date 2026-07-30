import assert from "node:assert/strict";
import test from "node:test";
import { tsconfigAliasTargets } from "../src/context/repository/projectConfig";

test("tsconfig aliases resolve exact and wildcard workspace targets", () => {
  const config = `{
    // application aliases
    "compilerOptions": { "baseUrl": ".", "paths": { "@/*": ["src/*"], "@shared": ["packages/shared/index"] } }
  }`;
  assert.deepEqual(tsconfigAliasTargets(config, "@/components/Button"), ["src/components/Button"]);
  assert.deepEqual(tsconfigAliasTargets(config, "@shared"), ["packages/shared/index"]);
});

test("invalid or unmatched tsconfig paths are ignored", () => {
  assert.deepEqual(tsconfigAliasTargets("not-json", "@/button"), []);
  assert.deepEqual(tsconfigAliasTargets('{"compilerOptions":{"paths":{"@/*":["src/*"]}}}', "react"), []);
});
