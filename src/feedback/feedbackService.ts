export type Feedback = "helpful" | "not-helpful";

/** Local aggregate only; completion text, paths, and code are never retained. */
export class FeedbackService {
  private helpful = 0;
  private notHelpful = 0;
  private shown = 0;
  private accepted = 0;
  private readonly byFocus = new Map<string, { shown: number; accepted: number }>();

  record(feedback: Feedback): void { if (feedback === "helpful") this.helpful++; else this.notHelpful++; }
  recordShown(focus: string): void { this.shown++; this.increment(focus, "shown"); }
  recordAccepted(focus: string): void { this.accepted++; this.increment(focus, "accepted"); }
  getStats() { return { helpful: this.helpful, notHelpful: this.notHelpful, shown: this.shown, accepted: this.accepted, byFocus: Object.fromEntries(this.byFocus) }; }

  private increment(focus: string, field: "shown" | "accepted"): void {
    const value = this.byFocus.get(focus) ?? { shown: 0, accepted: 0 };
    value[field]++;
    this.byFocus.set(focus, value);
  }
}
