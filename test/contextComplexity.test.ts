import assert from "node:assert/strict";
import test from "node:test";
import { completionFocus, completionRequestMode } from "../src/completion/contextComplexity";
import { CompletionContext } from "../src/context/contextTypes";

const context = (line: string, imports?: string): CompletionContext => ({ prefix: line, suffix: "", language: "typescript", filePath: "a.ts", safeFilePath: "a.ts", cursorOffset: line.length, documentVersion: 1, currentLine: line, textBeforeCursorOnLine: line, textAfterCursorOnLine: "", indentation: "", imports, metadata: { totalDocumentCharacters: line.length, prefixCharacters: line.length, suffixCharacters: 0, truncatedPrefix: false, truncatedSuffix: false, estimatedPrefixTokens: 1, estimatedSuffixTokens: 0, estimatedTotalTokens: 1, buildDurationMilliseconds: 0 } });

test("simple syntax uses the fast completion path", () => assert.equal(completionRequestMode(context("console.")), "fast"));
test("imported member access uses the full completion path", () => assert.equal(completionRequestMode(context("userService.", 'import { userService } from "./user";')), "full"));
test("imports retained in the prefix also use the full completion path", () => assert.equal(completionRequestMode({ ...context('import { userService } from "./user";\nuserService.'), imports: undefined }), "full"));
test("JSX and Tailwind class editing use the full completion path", () => {
  assert.equal(completionRequestMode(context("<Button", 'import { Button } from "antd";')), "full");
  assert.equal(completionRequestMode(context('className="')), "full");
});
test("imports and imported object types use repository context", () => {
  assert.equal(completionRequestMode(context("import { use")), "full");
  assert.equal(completionRequestMode(context("const user: User = {", 'import type { User } from "./types";')), "full");
});
test("function arguments use compiler signature context even without an import", () => {
  assert.equal(completionRequestMode(context("createOrder(")), "full");
});
test("completion focus distinguishes common completion intents", () => {
  assert.equal(completionFocus(context("<Button ")), "component-props");
  assert.equal(completionFocus(context("service.")), "member-access");
  assert.equal(completionFocus(context("createOrder(")), "function-arguments");
  assert.equal(completionFocus(context('className="')), "tailwind-class");
  assert.equal(completionFocus(context("{ name")), "object-fields");
  assert.equal(completionFocus(context("import { use")), "import");
  assert.equal(completionFocus(context("function total() {")), "function-body");
  assert.equal(completionFocus(context("if (user && ")), "condition-expression");
  assert.equal(completionFocus(context("interface Product {")), "type-definition");
});
