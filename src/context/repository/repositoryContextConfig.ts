export const defaultRepositoryContextLimits = {
  enabled: true,
  maxFiles: 4,
  maxCharacters: 12000,
  timeoutMs: 40,
} as const;

export interface RepositoryContextLimits {
  enabled: boolean;
  maxFiles: number;
  maxCharacters: number;
  timeoutMs: number;
}

export interface RepositoryContextSettings {
  getRepositoryContextLimits(): RepositoryContextLimits;
}

export function normalizeRepositoryContextLimits(value: RepositoryContextLimits): RepositoryContextLimits {
  return {
    enabled: value.enabled,
    maxFiles: integerWithin(value.maxFiles, 1, 8) ? value.maxFiles : defaultRepositoryContextLimits.maxFiles,
    maxCharacters: integerWithin(value.maxCharacters, 1000, 30000) ? value.maxCharacters : defaultRepositoryContextLimits.maxCharacters,
    timeoutMs: integerWithin(value.timeoutMs, 10, 100) ? value.timeoutMs : defaultRepositoryContextLimits.timeoutMs,
  };
}

function integerWithin(value: number, min: number, max: number): boolean {
  return Number.isInteger(value) && value >= min && value <= max;
}
