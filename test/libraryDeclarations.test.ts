import assert from "node:assert/strict";
import test from "node:test";
import { declarationEntry, extractDeclarationSymbols, packageRoot } from "../src/context/repository/libraryDeclarations";

test("library declaration helpers remain package-bounded", () => {
  assert.equal(packageRoot("@mui/material/Button"), "@mui/material");
  assert.equal(packageRoot("zod/v4"), "zod");
  assert.equal(declarationEntry('{"types":"dist/index.d.ts"}'), "dist/index.d.ts");
  assert.equal(declarationEntry('{"types":"../../secret"}'), "index.d.ts");
});

test("library declaration extraction keeps only requested signatures", () => {
  const symbols = extractDeclarationSymbols("export interface ButtonProps { disabled?: boolean }\n\nexport interface Hidden {}", ["ButtonProps"], "@mui/material");
  assert.equal(symbols.length, 1);
  assert.equal(symbols[0].signature, "export interface ButtonProps { disabled?: boolean }");
  assert.equal(symbols[0].filePath, "package:@mui/material");
});
