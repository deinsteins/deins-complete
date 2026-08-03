import { AccountApiClient } from "../api/deinsCompleteClient";
import { AccountEntitlements } from "../api/apiTypes";
import { EndpointNotFoundError, UnauthorizedError } from "../api/apiErrors";
import { CredentialStore } from "../security/credentialStore";
import { AccountStatus } from "./accountTypes";

export class AccountService {
  private required: boolean | undefined;
  private requirementRequest: Promise<boolean> | undefined;
  private linkedInstallationToken: string | undefined;
  constructor(private readonly client: AccountApiClient, private readonly store: CredentialStore) {}

  async isSignedIn(): Promise<boolean> { return (await this.store.getAccountRefreshToken()) !== undefined; }
  async isRequired(signal?: AbortSignal): Promise<boolean> {
    if (this.required !== undefined) return this.required;
    if (this.requirementRequest === undefined) {
      this.requirementRequest = this.loadRequirement(signal).finally(() => { this.requirementRequest = undefined; });
    }
    return this.requirementRequest;
  }
  requestMagicCode(email: string, inviteCode?: string, signal?: AbortSignal): Promise<void> { return this.client.requestMagicCode(email, inviteCode, signal); }
  async verifyMagicCode(email: string, code: string, installationToken: string | undefined, signal?: AbortSignal): Promise<void> {
    const tokens = await this.client.verifyMagicCode(email, code, signal);
    await this.store.setAccountTokens(tokens.accessToken, tokens.refreshToken);
    if (installationToken !== undefined) { await this.client.linkInstallation(tokens.accessToken, installationToken, signal); this.linkedInstallationToken = installationToken; }
  }
  async signOut(signal?: AbortSignal): Promise<void> {
    const refreshToken = await this.store.getAccountRefreshToken();
    try { if (refreshToken !== undefined) await this.client.logoutAccount(refreshToken, signal); } finally { this.linkedInstallationToken = undefined; await this.store.deleteAccountTokens(); }
  }
  async getStatus(signal?: AbortSignal): Promise<AccountStatus | undefined> {
    if (!await this.isSignedIn()) return undefined;
    const [account, entitlements, installations] = await this.withAccess((token) => Promise.all([
      this.client.getAccount(token, signal),
      this.client.getEntitlements(token, signal),
      this.client.getAccountInstallations(token, signal),
    ]));
    return { account, entitlements, installations };
  }
  getEntitlements(signal?: AbortSignal): Promise<AccountEntitlements> { return this.withAccess((token) => this.client.getEntitlements(token, signal)); }
  revokeInstallation(installationID: string, signal?: AbortSignal): Promise<void> { return this.withAccess((token) => this.client.revokeAccountInstallation(token, installationID, signal)); }
  async linkInstallation(installationToken: string, signal?: AbortSignal): Promise<void> { await this.withAccess((token) => this.client.linkInstallation(token, installationToken, signal)); this.linkedInstallationToken = installationToken; }
  async ensureLinked(installationToken: string, signal?: AbortSignal): Promise<void> { if (this.linkedInstallationToken !== installationToken) await this.linkInstallation(installationToken, signal); }

  private async withAccess<T>(operation: (token: string) => Promise<T>): Promise<T> {
    let token = await this.store.getAccountAccessToken();
    if (token === undefined) token = await this.refresh();
    try { return await operation(token); } catch (error) {
      if (!(error instanceof UnauthorizedError)) throw error;
      return operation(await this.refresh());
    }
  }
  private async refresh(): Promise<string> {
    const refreshToken = await this.store.getAccountRefreshToken();
    if (refreshToken === undefined) throw new UnauthorizedError("Sign in to your DeinsComplete account.");
    const tokens = await this.client.refreshAccount(refreshToken);
    await this.store.setAccountTokens(tokens.accessToken, tokens.refreshToken);
    return tokens.accessToken;
  }
  private async loadRequirement(signal?: AbortSignal): Promise<boolean> {
    try { return this.required = await this.client.getAccountRequirement(signal); }
    catch (error) { if (error instanceof EndpointNotFoundError) return this.required = false; throw error; }
  }
}
