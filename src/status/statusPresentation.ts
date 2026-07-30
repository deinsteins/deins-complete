import { DeinsCompleteState } from "../config/configTypes";

export interface StatusBarPresentation {
  text: string;
  tooltip: string;
  command: string;
}

export function getStatusBarPresentation(state: DeinsCompleteState): StatusBarPresentation {
  return state === "enabled"
    ? { text: "$(sparkle) DeinsComplete", tooltip: "DeinsComplete is enabled", command: "deinscomplete.disable" }
    : { text: "$(circle-slash) DeinsComplete", tooltip: "DeinsComplete is disabled", command: "deinscomplete.enable" };
}
