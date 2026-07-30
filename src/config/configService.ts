import * as vscode from "vscode";
import { BackendSettingsProvider } from "../api/apiTypes";
import { defaultBackendSettings, normalizeBackendTimeout } from "./backendConfig";
import { ContextLimits, ContextLimitsProvider, defaultContextLimits, normalizeContextLimits } from "../context/contextConfig";
import { EnabledConfiguration } from "./configTypes";

const namespace = "deinscomplete";
const enabledSetting = "enabled";

export class ConfigService implements EnabledConfiguration, ContextLimitsProvider, BackendSettingsProvider {
  debounceMs(){return Math.max(0,Math.min(2000,vscode.workspace.getConfiguration(namespace).get<number>("completion.debounceMs",150)))}
  cacheEnabled(){return vscode.workspace.getConfiguration(namespace).get<boolean>("cache.enabled",true)}
  cacheTtlMs(){const v=vscode.workspace.getConfiguration(namespace).get<number>("cache.ttlMs",60000);return v>=1000&&v<=300000?v:60000}
  cacheMaxEntries(){const v=vscode.workspace.getConfiguration(namespace).get<number>("cache.maxEntries",100);return v>=10&&v<=1000?v:100}
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
