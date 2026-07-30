export type Feedback = "helpful" | "not-helpful";

/** Local aggregate only; completion text, paths, and code are never retained. */
export class FeedbackService {
  private helpful = 0;
  private notHelpful = 0;
  record(feedback: Feedback): void { if (feedback === "helpful") this.helpful++; else this.notHelpful++; }
  getStats() { return { helpful: this.helpful, notHelpful: this.notHelpful }; }
}
