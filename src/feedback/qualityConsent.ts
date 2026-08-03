export type QualityConsent = "enabled" | "declined" | "learned-more";

export function shouldOfferQualityInsights(consent: QualityConsent | undefined, explicitlyConfigured: boolean, enabled: boolean): boolean {
  return consent === undefined && !explicitlyConfigured && !enabled;
}
