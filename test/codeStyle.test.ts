import assert from "node:assert/strict";
import test from "node:test";
import { inferCodeStyle } from "../src/context/codeStyle";

test("code style detects bounded formatting preferences", () => {
  assert.deepEqual(inferCodeStyle("const name = 'Ada';\n  return name;\n"), {
    indentation: "spaces", indentSize: 2, quote: "single", semicolons: "always",
  });
  assert.deepEqual(inferCodeStyle("function value() {\n\treturn \"ok\"\n}\n"), {
    indentation: "tabs", quote: "double", semicolons: "never",
  });
});
