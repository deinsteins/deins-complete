import assert from "node:assert/strict";
import test from "node:test";
import { EnabledConfiguration } from "../src/config/configTypes";
import { DeinsCompleteLifecycle } from "../src/core/lifecycle";

class TestConfig implements EnabledConfiguration {
  private readonly listeners = new Set<() => void>();

  constructor(private enabled: boolean) {}

  isEnabled(): boolean { return this.enabled; }
  onDidChangeEnabled(listener: () => void): { dispose(): void } {
    this.listeners.add(listener);
    return { dispose: () => this.listeners.delete(listener) };
  }
  setEnabled(enabled: boolean): void {
    this.enabled = enabled;
    this.listeners.forEach((listener) => listener());
  }
}

test("lifecycle starts enabled when configuration is enabled", () => {
  const lifecycle = new DeinsCompleteLifecycle(new TestConfig(true));
  assert.equal(lifecycle.getState(), "enabled");
});

test("lifecycle starts disabled when configuration is disabled", () => {
  const lifecycle = new DeinsCompleteLifecycle(new TestConfig(false));
  assert.equal(lifecycle.getState(), "disabled");
});

test("lifecycle publishes configuration transitions", () => {
  const config = new TestConfig(true);
  const lifecycle = new DeinsCompleteLifecycle(config);
  const states: string[] = [];
  lifecycle.onDidChangeState((state) => states.push(state));
  lifecycle.start();

  config.setEnabled(false);
  config.setEnabled(true);

  assert.deepEqual(states, ["disabled", "enabled"]);
});
