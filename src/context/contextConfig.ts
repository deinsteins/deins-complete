export const defaultContextLimits = {
  maxPrefixCharacters: 4000,
  maxSuffixCharacters: 2000,
} as const;

export interface ContextLimits {
  maxPrefixCharacters: number;
  maxSuffixCharacters: number;
}

export interface ContextLimitsProvider {
  getContextLimits(): ContextLimits;
}

export function normalizeContextLimits(limits: ContextLimits): ContextLimits {
  return {
    maxPrefixCharacters: isWithin(limits.maxPrefixCharacters, 500, 20000)
      ? limits.maxPrefixCharacters
      : defaultContextLimits.maxPrefixCharacters,
    maxSuffixCharacters: isWithin(limits.maxSuffixCharacters, 200, 10000)
      ? limits.maxSuffixCharacters
      : defaultContextLimits.maxSuffixCharacters,
  };
}

function isWithin(value: number, minimum: number, maximum: number): boolean {
  return Number.isInteger(value) && value >= minimum && value <= maximum;
}
