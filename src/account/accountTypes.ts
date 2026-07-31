import { AccountDetails, AccountEntitlements, AccountInstallation } from "../api/apiTypes";

export interface AccountStatus {
  account: AccountDetails;
  entitlements: AccountEntitlements;
  installations: AccountInstallation[];
}
