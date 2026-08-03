import assert from "node:assert/strict";
import test from "node:test";
import { getStatusBarPresentation } from "../src/status/statusPresentation";

test("enabled status bar presentation opens Account Center", () => {
  assert.deepEqual(getStatusBarPresentation("enabled"), {
    text: "$(sparkle) DeinsComplete",
    tooltip: "DeinsComplete is enabled",
    command: "deinscomplete.accountCenter",
  });
});

test("enabled status bar exposes remaining monthly quota in its hover tooltip", () => {
  const presentation = getStatusBarPresentation("enabled", { plan: "free", used: 371, limit: 2000 });
  assert.match(presentation.tooltip, /Plan: free/);
  assert.match(presentation.tooltip, /1,629 remaining \(371 \/ 2,000\)/);
});

test("disabled status bar presentation enables DeinsComplete", () => {
  assert.deepEqual(getStatusBarPresentation("disabled"), {
    text: "$(circle-slash) DeinsComplete",
    tooltip: "DeinsComplete is disabled",
    command: "deinscomplete.enable",
  });
});
