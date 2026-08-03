import assert from "node:assert/strict";
import test from "node:test";
import { shouldOfferQualityInsights } from "../src/feedback/qualityConsent";

test("quality insights consent is offered only once when no preference exists", () => {
  assert.equal(shouldOfferQualityInsights(undefined, false, false), true);
  assert.equal(shouldOfferQualityInsights("declined", false, false), false);
  assert.equal(shouldOfferQualityInsights("learned-more", false, false), false);
  assert.equal(shouldOfferQualityInsights(undefined, true, false), false);
  assert.equal(shouldOfferQualityInsights(undefined, false, true), false);
});
