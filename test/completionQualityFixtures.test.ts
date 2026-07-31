import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import test from "node:test";
import { CompletionFocus, CompletionRequestMode, completionFocus, completionRequestMode } from "../src/completion/contextComplexity";
import { CompletionContext } from "../src/context/contextTypes";

interface QualityFixture {
  name: string;
  language: string;
  filePath: string;
  prefix: string;
  imports: string;
  expectedMode: CompletionRequestMode;
  expectedFocus: CompletionFocus;
}

const fixtures = JSON.parse(readFileSync(resolve(__dirname, "../../test/fixtures/completion-quality.json"), "utf8")) as QualityFixture[];

for (const fixture of fixtures) {
  test(`quality fixture: ${fixture.name}`, () => {
    const context = qualityContext(fixture);
    assert.equal(completionRequestMode(context), fixture.expectedMode);
    assert.equal(completionFocus(context), fixture.expectedFocus);
    assert.equal(fixture.filePath.startsWith("/") || /^[A-Za-z]:/.test(fixture.filePath), false);
  });
}

function qualityContext(fixture: QualityFixture): CompletionContext {
  return {
    prefix: fixture.prefix,
    suffix: "",
    language: fixture.language,
    filePath: fixture.filePath,
    safeFilePath: fixture.filePath,
    cursorOffset: fixture.prefix.length,
    documentVersion: 1,
    currentLine: fixture.prefix,
    textBeforeCursorOnLine: fixture.prefix,
    textAfterCursorOnLine: "",
    indentation: "",
    imports: fixture.imports,
    metadata: {
      totalDocumentCharacters: fixture.prefix.length,
      prefixCharacters: fixture.prefix.length,
      suffixCharacters: 0,
      truncatedPrefix: false,
      truncatedSuffix: false,
      estimatedPrefixTokens: Math.ceil(fixture.prefix.length / 4),
      estimatedSuffixTokens: 0,
      estimatedTotalTokens: Math.ceil(fixture.prefix.length / 4),
      buildDurationMilliseconds: 0,
    },
  };
}
