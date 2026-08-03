import { DeinsCompleteState } from "../config/configTypes";

export interface StatusBarPresentation {
  text: string;
  tooltip: string;
  command: string;
}

export interface QuotaPresentation {
  plan: string;
  used: number;
  limit: number;
}

export function getStatusBarPresentation(state: DeinsCompleteState, quota?: QuotaPresentation): StatusBarPresentation {
  const quotaLine = quota === undefined ? "" : `\nPlan: ${quota.plan}\nMonthly quota: ${Math.max(0, quota.limit - quota.used).toLocaleString()} remaining (${quota.used.toLocaleString()} / ${quota.limit.toLocaleString()})`;
  return state === "enabled"
    ? { text: "$(sparkle) DeinsComplete", tooltip: `DeinsComplete is enabled${quotaLine}`, command: "deinscomplete.accountCenter" }
    : { text: "$(circle-slash) DeinsComplete", tooltip: `DeinsComplete is disabled${quotaLine}`, command: "deinscomplete.enable" };
}
