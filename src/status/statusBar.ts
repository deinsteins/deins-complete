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

  setActivity(text: "Thinking…" | "Cached" | "Offline" | "Ready"): void {
    const icon = text === "Thinking…" ? "$(sync~spin)" : text === "Cached" ? "$(database)" : text === "Offline" ? "$(warning)" : "$(check)";
    this.item.text = `${icon} DeinsComplete: ${text}`;
    this.item.tooltip = text === "Offline" ? "The DeinsComplete backend is temporarily unavailable. Completion will retry automatically." : text === "Thinking…" ? "Generating an inline completion…" : `DeinsComplete ${text.toLowerCase()}`;
    this.item.command = text === "Thinking…" ? undefined : "deinscomplete.accountCenter";
    this.item.show();
  }

  setQuotaExceeded(): void {
    this.item.text = "$(warning) DeinsComplete: Quota used";
    this.item.tooltip = "Monthly completion quota is exhausted. Open Account Center for details.";
    this.item.command = "deinscomplete.accountCenter";
    this.item.show();
  }

  dispose(): void { this.item.dispose(); }
}
