import assert from "node:assert/strict";
import test from "node:test";
import type * as vscode from "vscode";
import { ContextBuilder } from "../src/context/contextBuilder";
import { ContextLimits } from "../src/context/contextConfig";
import { ApproximateTokenEstimator } from "../src/context/tokenEstimator";

function build(text: string, offset: number, limits: ContextLimits = { maxPrefixCharacters: 4000, maxSuffixCharacters: 2000 }) {
  const lines = text.split("\n");
  const beforeCursor = text.slice(0, offset);
  const line = beforeCursor.split("\n").length - 1;
  const character = beforeCursor.length - beforeCursor.lastIndexOf("\n") - 1;
  const document = {
    languageId: "typescript",
    uri: { fsPath: "/test.ts" },
    version: 7,
    getText: () => text,
    offsetAt: () => offset,
    lineAt: (lineNumber: number) => ({ text: lines[lineNumber] }),
  } as unknown as vscode.TextDocument;
  const position = { line, character } as vscode.Position;
  return new ContextBuilder({ getContextLimits: () => limits }).build(document, position);
}

test("context builder preserves exact cursor boundaries in a small file", () => {
  const context = build("abcdef", 3);
  assert.equal(context.prefix, "abc");
  assert.equal(context.suffix, "def");
  assert.equal(context.cursorOffset, 3);
  assert.equal(context.currentLine, "abcdef");
});

test("context builder bounds large cursor windows", () => {
  const before = `${"x".repeat(600)}\nconst user =`;
  const after = `\n${"y".repeat(300)}`;
  const context = build(`${before}${after}`, before.length, { maxPrefixCharacters: 500, maxSuffixCharacters: 200 });
  assert.ok(context.prefix.length <= 500);
  assert.ok(context.suffix.length <= 200);
  assert.ok(context.prefix.endsWith("const user ="));
  assert.ok(context.suffix.startsWith("\n"));
  assert.equal(context.metadata.truncatedPrefix, true);
  assert.equal(context.metadata.truncatedSuffix, true);
});

test("context builder keeps a 50k-character document bounded", () => {
  const before = `${"x".repeat(50000)}const user =`;
  const context = build(`${before}${"y".repeat(50000)}`, before.length, { maxPrefixCharacters: 500, maxSuffixCharacters: 200 });
  assert.ok(context.prefix.length <= 500);
  assert.ok(context.suffix.length <= 200);
  assert.equal(context.metadata.totalDocumentCharacters, 100012);
});

test("context builder handles beginning, end, and empty documents", () => {
  const beginning = build("hello", 0);
  const end = build("hello", 5);
  const empty = build("", 0);
  assert.equal(beginning.prefix, "");
  assert.equal(beginning.suffix, "hello");
  assert.equal(end.prefix, "hello");
  assert.equal(end.suffix, "");
  assert.equal(empty.currentLine, "");
  assert.equal(empty.indentation, "");
});

test("context builder preserves imports outside the prefix window and derives indentation", () => {
  const text = `import { getUser } from "./user";\n\n${"x".repeat(600)}\nfunction run() {\n    const user =`;
  const context = build(text, text.length, { maxPrefixCharacters: 500, maxSuffixCharacters: 200 });
  assert.equal(context.imports, 'import { getUser } from "./user";');
  assert.equal(context.indentation, "    ");
  assert.equal(context.textBeforeCursorOnLine, "    const user =");
});

test("context builder does not duplicate imports already in the prefix", () => {
  const text = `import { getUser } from "./user";\nconst user =`;
  assert.equal(build(text, text.length).imports, undefined);
});

test("approximate token estimator rounds characters up by four", () => {
  assert.equal(new ApproximateTokenEstimator().estimate("x".repeat(400)), 100);
  assert.equal(new ApproximateTokenEstimator().estimate("x"), 1);
});
