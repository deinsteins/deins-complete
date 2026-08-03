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

test("completion experience persists aggregate counters and adapts debounce", () => {
  let saved: unknown;
  const store = { get: <T>() => saved as T | undefined, update: async (_key: string, value: unknown) => { saved = value; } };
  const feedback = new FeedbackService(store);
  for (let index = 0; index < 5; index++) { feedback.recordShown("member-access"); feedback.recordAccepted("member-access"); }
  assert.equal(feedback.debounceAdjustment("member-access"), -20);
  const restored = new FeedbackService(store);
  assert.equal(restored.getStats().accepted, 5);
});
