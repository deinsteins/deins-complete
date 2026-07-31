import * as vscode from "vscode";
import { DeinsCompleteState } from "../config/configTypes";
import { getStatusBarPresentation, QuotaPresentation } from "./statusPresentation";

export class DeinsCompleteStatusBar implements vscode.Disposable {
  private readonly item = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 100);
  private quota: QuotaPresentation | undefined;

  update(state: DeinsCompleteState): void {
    const presentation = getStatusBarPresentation(state, this.quota);
    this.item.text = presentation.text;
    this.item.tooltip = presentation.tooltip;
    this.item.command = presentation.command;
    this.item.show();
  }

  setQuota(quota: QuotaPresentation | undefined, state: DeinsCompleteState): void { this.quota = quota; this.update(state); }

  setActivity(text: "Thinking…" | "Cached" | "Offline" | "Ready"): void { if (text === "Ready") return; this.item.text = `$(sparkle) DeinsComplete: ${text}`; this.item.tooltip = "DeinsComplete inline completion"; this.item.show(); }

  dispose(): void { this.item.dispose(); }
}
