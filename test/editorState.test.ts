import assert from "node:assert/strict";
import test from "node:test";
import { isCurrentEditorState } from "../src/completion/editorState";

test("editor state remains current when URI and version match", () => {
  assert.equal(isCurrentEditorState({ uri: "file:///test.ts", version: 1 }, { uri: "file:///test.ts", version: 1 }), true);
});

test("editor state becomes stale when document version changes", () => {
  assert.equal(isCurrentEditorState({ uri: "file:///test.ts", version: 1 }, { uri: "file:///test.ts", version: 2 }), false);
});
