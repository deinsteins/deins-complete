import * as vscode from "vscode";
import { BackendSettingsProvider } from "../api/apiTypes";
import { defaultBackendSettings, normalizeBackendTimeout } from "./backendConfig";
import { ContextLimits, ContextLimitsProvider, defaultContextLimits, normalizeContextLimits } from "../context/contextConfig";
import { EnabledConfiguration } from "./configTypes";

const namespace = "deinscomplete";
const enabledSetting = "enabled";

export class ConfigService implements EnabledConfiguration, ContextLimitsProvider, BackendSettingsProvider {
  isEnabled(): boolean {
    return vscode.workspace.getConfiguration(namespace).get<boolean>(enabledSetting, true);
  }

  async setEnabled(enabled: boolean): Promise<void> {
    await vscode.workspace.getConfiguration(namespace).update(enabledSetting, enabled, vscode.ConfigurationTarget.Global);
  }

  getContextLimits(): ContextLimits {
    const configuration = vscode.workspace.getConfiguration(namespace);
    return normalizeContextLimits({
      maxPrefixCharacters: configuration.get<number>("context.maxPrefixCharacters", defaultContextLimits.maxPrefixCharacters),
      maxSuffixCharacters: configuration.get<number>("context.maxSuffixCharacters", defaultContextLimits.maxSuffixCharacters),
    });
  }

  getBackendUrl(): string {
    return vscode.workspace.getConfiguration(namespace).get<string>("backend.url", defaultBackendSettings.url);
  }

  getBackendTimeoutMs(): number {
    return normalizeBackendTimeout(vscode.workspace.getConfiguration(namespace).get<number>("backend.timeoutMs", defaultBackendSettings.timeoutMs));
  }

  onDidChangeEnabled(listener: () => void): vscode.Disposable {
    return vscode.workspace.onDidChangeConfiguration((event) => {
      if (event.affectsConfiguration(`${namespace}.${enabledSetting}`)) {
        listener();
      }
    });
  }
}
