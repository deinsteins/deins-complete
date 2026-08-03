export type Feedback = "helpful" | "not-helpful";
type StoredStats = { helpful: number; notHelpful: number; shown: number; accepted: number; byFocus: Record<string, { shown: number; accepted: number }> };
type StateStore = { get<T>(key: string): T | undefined; update(key: string, value: unknown): PromiseLike<void> };

const storageKey = "deinscomplete.completionExperience.v1";

/** Local aggregate only; completion text, paths, and code are never retained. */
export class FeedbackService {
  private helpful = 0;
  private notHelpful = 0;
  private shown = 0;
  private accepted = 0;
  private readonly byFocus = new Map<string, { shown: number; accepted: number }>();

  constructor(private readonly store?: StateStore) {
    const saved = store?.get<StoredStats>(storageKey);
    if (saved === undefined) return;
    this.helpful = saved.helpful; this.notHelpful = saved.notHelpful; this.shown = saved.shown; this.accepted = saved.accepted;
    for (const [focus, value] of Object.entries(saved.byFocus)) this.byFocus.set(focus, value);
  }

  record(feedback: Feedback): void { if (feedback === "helpful") this.helpful++; else this.notHelpful++; this.persist(); }
  recordShown(focus: string): void { this.shown++; this.increment(focus, "shown"); this.persist(); }
  recordAccepted(focus: string): void { this.accepted++; this.increment(focus, "accepted"); this.persist(); }
  getStats() { return { helpful: this.helpful, notHelpful: this.notHelpful, shown: this.shown, accepted: this.accepted, byFocus: Object.fromEntries(this.byFocus) }; }
  debounceAdjustment(focus: string): number {
    const value = this.byFocus.get(focus);
    if (value === undefined || value.shown < 5) return 0;
    const acceptance = value.accepted / value.shown;
    return acceptance >= 0.6 ? -20 : acceptance <= 0.2 ? 30 : 0;
  }

  private increment(focus: string, field: "shown" | "accepted"): void {
    const value = this.byFocus.get(focus) ?? { shown: 0, accepted: 0 };
    value[field]++;
    this.byFocus.set(focus, value);
  }

  private persist(): void {
    if (this.store === undefined) return;
    void this.store.update(storageKey, this.getStats());
  }
}
