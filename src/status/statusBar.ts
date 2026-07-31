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

  setSignInRequired(): void {
    this.item.text = "$(account) DeinsComplete: Sign in";
    this.item.tooltip = "Sign in is required to use DeinsComplete inline completions.";
    this.item.command = "deinscomplete.signIn";
    this.item.show();
  }

  setActivity(text: "Thinking…" | "Cached" | "Offline" | "Ready"): void { if (text === "Ready") return; this.item.text = `$(sync~spin) DeinsComplete: ${text}`; this.item.tooltip = "DeinsComplete inline completion in progress"; this.item.command = undefined; this.item.show(); }

  dispose(): void { this.item.dispose(); }
}
