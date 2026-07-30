import * as vscode from "vscode";
const key="deinscomplete.installationToken";
export class CredentialStore { constructor(private readonly secrets:vscode.SecretStorage){} getInstallationToken(){return this.secrets.get(key)} setInstallationToken(token:string){return this.secrets.store(key,token)} deleteInstallationToken(){return this.secrets.delete(key)} }
