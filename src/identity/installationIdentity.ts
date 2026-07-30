import * as crypto from "node:crypto";
import * as vscode from "vscode";
const key="deinscomplete.installationId";
export function getInstallationId(state:vscode.Memento):Promise<string>{const existing=state.get<string>(key);if(existing)return Promise.resolve(existing);const id=crypto.randomUUID();return Promise.resolve(state.update(key,id)).then(()=>id)}
