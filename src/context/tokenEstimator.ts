export interface TokenEstimator {
  estimate(text: string): number;
}

export class ApproximateTokenEstimator implements TokenEstimator {
  estimate(text: string): number {
    return Math.ceil(text.length / 4);
  }
}
