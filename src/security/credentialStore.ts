import * as vscode from "vscode";
const key="deinscomplete.installationToken";
const accountAccessKey = "deinscomplete.accountAccessToken";
const accountRefreshKey = "deinscomplete.accountRefreshToken";

export class CredentialStore {
  constructor(private readonly secrets: vscode.SecretStorage) {}
  getInstallationToken() { return this.secrets.get(key); }
  setInstallationToken(token: string) { return this.secrets.store(key, token); }
  deleteInstallationToken() { return this.secrets.delete(key); }
  getAccountAccessToken() { return this.secrets.get(accountAccessKey); }
  getAccountRefreshToken() { return this.secrets.get(accountRefreshKey); }
  async setAccountTokens(accessToken: string, refreshToken: string): Promise<void> {
    await Promise.all([this.secrets.store(accountAccessKey, accessToken), this.secrets.store(accountRefreshKey, refreshToken)]);
  }
  async deleteAccountTokens(): Promise<void> {
    await Promise.all([this.secrets.delete(accountAccessKey), this.secrets.delete(accountRefreshKey)]);
  }
}
