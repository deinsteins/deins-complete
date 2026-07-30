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
test("completion focus distinguishes props, members, calls, classes, and object fields", () => {
  assert.equal(completionFocus(context("<Button ")), "component-props");
  assert.equal(completionFocus(context("service.")), "member-access");
  assert.equal(completionFocus(context("createOrder(")), "function-arguments");
  assert.equal(completionFocus(context('className="')), "tailwind-class");
  assert.equal(completionFocus(context("{ name")), "object-fields");
});
