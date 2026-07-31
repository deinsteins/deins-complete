import assert from "node:assert/strict";
import test from "node:test";
import { AccountService } from "../src/account/accountService";
import { AccountApiClient } from "../src/api/deinsCompleteClient";
import { CredentialStore } from "../src/security/credentialStore";

function credentials(): CredentialStore {
  const values = new Map<string, string>();
  return new CredentialStore({ get: async (key: string) => values.get(key), store: async (key: string, value: string) => { values.set(key, value); }, delete: async (key: string) => { values.delete(key); } } as never);
}

function client(): AccountApiClient {
  return {
    requestMagicCode: async () => undefined,
    verifyMagicCode: async () => ({ accessToken: "access", refreshToken: "refresh", expiresIn: 1800 }),
    refreshAccount: async () => ({ accessToken: "new-access", refreshToken: "new-refresh", expiresIn: 1800 }),
    logoutAccount: async () => undefined,
    getAccount: async () => ({ user: { id: "user-1", email: "user@example.com" }, plan: { code: "free" } }),
    getEntitlements: async () => ({ plan: "free", features: { repositoryContext: false, streaming: true, premiumRouting: false }, limits: { monthlyCompletions: 2000, used: 3, remaining: 1997 } }),
    linkInstallation: async () => undefined,
    getAccountInstallations: async () => [{ id: "installation-1", status: "active", createdAt: "2026-01-01T00:00:00Z" }],
    revokeAccountInstallation: async () => undefined,
    getAccountRequirement: async () => false,
  };
}

test("account service stores only account tokens in SecretStorage and links after magic verification", async () => {
  const store = credentials(); let linked = "";
  const api = client(); api.linkInstallation = async (_access, installation) => { linked = installation; };
  const service = new AccountService(api, store);
  await service.verifyMagicCode("user@example.com", "123456", "installation-token");
  assert.equal(await store.getAccountAccessToken(), "access");
  assert.equal(await store.getAccountRefreshToken(), "refresh");
  assert.equal(linked, "installation-token");
  await service.signOut();
  assert.equal(await service.isSignedIn(), false);
});

test("account requirement is discovered once and cached", async () => {
  let calls = 0; const api = client(); api.getAccountRequirement = async () => { calls++; await new Promise((resolve) => setTimeout(resolve, 5)); return true; };
  const service = new AccountService(api, credentials());
  assert.deepEqual(await Promise.all([service.isRequired(), service.isRequired(), service.isRequired()]), [true, true, true]);
  assert.equal(await service.isRequired(), true); assert.equal(calls, 1);
});

test("account service links a persisted sign-in only once per activation", async () => {
  let links = 0; const api = client(); api.linkInstallation = async () => { links++; };
  const store = credentials(); await store.setAccountTokens("access", "refresh");
  const service = new AccountService(api, store);
  await service.ensureLinked("installation-1"); await service.ensureLinked("installation-1");
  await service.ensureLinked("installation-2");
  assert.equal(links, 2);
});

test("account status resolves account, entitlements, and installations", async () => {
  const service = new AccountService(client(), credentials());
  await service.verifyMagicCode("user@example.com", "123456", undefined);
  const status = await service.getStatus();
  assert.equal(status?.account.plan.code, "free");
  assert.equal(status?.entitlements.limits.remaining, 1997);
  assert.equal(status?.installations.length, 1);
});
