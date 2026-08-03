import assert from "node:assert/strict";
import test from "node:test";
import type * as vscode from "vscode";
import { compactSignatureHelp, identifiersForDefinition } from "../src/context/repository/compilerSignals";
import { CompletionContext } from "../src/context/contextTypes";

function context(prefix: string): CompletionContext {
  return { prefix, suffix: "", language: "typescript", filePath: "a.ts", safeFilePath: "a.ts", cursorOffset: prefix.length, documentVersion: 1, currentLine: prefix, textBeforeCursorOnLine: prefix, textAfterCursorOnLine: "", indentation: "", metadata: { totalDocumentCharacters: prefix.length, prefixCharacters: prefix.length, suffixCharacters: 0, truncatedPrefix: false, truncatedSuffix: false, estimatedPrefixTokens: 1, estimatedSuffixTokens: 0, estimatedTotalTokens: 1, buildDurationMilliseconds: 0 } };
}

test("compiler signals select the nearest bounded identifiers", () => {
  assert.deepEqual(identifiersForDefinition(context("const result = userService.getUser(" )).map((item) => item.name), ["getUser", "userService"]);
});

test("signature help is reduced to bounded active signature metadata", () => {
  const help = { activeSignature: 0, activeParameter: 1, signatures: [{ label: "createOrder(user: User, options: Options)", documentation: undefined, parameters: [{ label: "user: User" }, { label: [24, 40] }] }] } as unknown as vscode.SignatureHelp;
  assert.deepEqual(compactSignatureHelp(help), { label: "createOrder(user: User, options: Options)", activeParameter: 1, parameter: "options: Options" });
});
