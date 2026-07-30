import { DeinsCompleteState, EnabledConfiguration } from "../config/configTypes";

export class DeinsCompleteLifecycle {
  private state: DeinsCompleteState;
  private readonly listeners = new Set<(state: DeinsCompleteState) => void>();

  constructor(private readonly config: EnabledConfiguration) {
    this.state = this.deriveState();
  }

  getState(): DeinsCompleteState {
    return this.state;
  }

  start(): { dispose(): void } {
    return this.config.onDidChangeEnabled(() => this.refresh());
  }

  onDidChangeState(listener: (state: DeinsCompleteState) => void): { dispose(): void } {
    this.listeners.add(listener);
    return { dispose: () => this.listeners.delete(listener) };
  }

  refresh(): void {
    const nextState = this.deriveState();
    if (nextState === this.state) {
      return;
    }
    this.state = nextState;
    this.listeners.forEach((listener) => listener(this.state));
  }

  private deriveState(): DeinsCompleteState {
    return this.config.isEnabled() ? "enabled" : "disabled";
  }
}
