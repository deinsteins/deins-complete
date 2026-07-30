import * as vscode from "vscode";
import { EnabledConfiguration } from "./configTypes";

const namespace = "deinscomplete";
const enabledSetting = "enabled";

export class ConfigService implements EnabledConfiguration {
  isEnabled(): boolean {
    return vscode.workspace.getConfiguration(namespace).get<boolean>(enabledSetting, true);
  }

  async setEnabled(enabled: boolean): Promise<void> {
    await vscode.workspace.getConfiguration(namespace).update(enabledSetting, enabled, vscode.ConfigurationTarget.Global);
  }

  onDidChangeEnabled(listener: () => void): vscode.Disposable {
    return vscode.workspace.onDidChangeConfiguration((event) => {
      if (event.affectsConfiguration(`${namespace}.${enabledSetting}`)) {
        listener();
      }
    });
  }
}
