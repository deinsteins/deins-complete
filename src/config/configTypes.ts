export type DeinsCompleteState = "enabled" | "disabled";

export interface EnabledConfiguration {
  isEnabled(): boolean;
  onDidChangeEnabled(listener: () => void): { dispose(): void };
}
