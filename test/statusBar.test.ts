import assert from "node:assert/strict";
import test from "node:test";
import { getStatusBarPresentation } from "../src/status/statusPresentation";

test("enabled status bar presentation disables DeinsComplete", () => {
  assert.deepEqual(getStatusBarPresentation("enabled"), {
    text: "$(sparkle) DeinsComplete",
    tooltip: "DeinsComplete is enabled",
    command: "deinscomplete.disable",
  });
});

test("disabled status bar presentation enables DeinsComplete", () => {
  assert.deepEqual(getStatusBarPresentation("disabled"), {
    text: "$(circle-slash) DeinsComplete",
    tooltip: "DeinsComplete is disabled",
    command: "deinscomplete.enable",
  });
});
