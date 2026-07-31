import assert from "node:assert/strict";
import test from "node:test";
import { FeedbackService } from "../src/feedback/feedbackService";

test("completion outcome metrics retain only aggregate counters", () => {
  const feedback = new FeedbackService();
  feedback.recordShown("member-access");
  feedback.recordShown("member-access");
  feedback.recordAccepted("member-access");
  assert.deepEqual(feedback.getStats().byFocus["member-access"], { shown: 2, accepted: 1 });
});
