export const defaultBackendSettings = {
  url: "http://127.0.0.1:3001",
  timeoutMs: 5000,
} as const;

export function normalizeBackendTimeout(value: number): number {
  return Number.isInteger(value) && value >= 500 && value <= 30000 ? value : defaultBackendSettings.timeoutMs;
}
