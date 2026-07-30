import * as vscode from "vscode";
import { ContextLimits, ContextLimitsProvider, defaultContextLimits, normalizeContextLimits } from "../context/contextConfig";
import { EnabledConfiguration } from "./configTypes";

const namespace = "deinscomplete";
const enabledSetting = "enabled";

export class ConfigService implements EnabledConfiguration, ContextLimitsProvider {
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

  onDidChangeEnabled(listener: () => void): vscode.Disposable {
    return vscode.workspace.onDidChangeConfiguration((event) => {
      if (event.affectsConfiguration(`${namespace}.${enabledSetting}`)) {
        listener();
      }
    });
  }
}
