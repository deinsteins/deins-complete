import assert from "node:assert/strict";
import test from "node:test";
import { InstallationService } from "../src/identity/installationService";
import { CredentialStore } from "../src/security/credentialStore";

test("installation registration is deduplicated and stores only the token in SecretStorage", async () => {
  let token: string | undefined;
  let registrations = 0;
  const secrets = {
    get: async () => token,
    store: async (_key: string, value: string) => { token = value; },
    delete: async () => { token = undefined; },
  };
  const service = new InstallationService(
    async () => "random-installation-id",
    new CredentialStore(secrets as never),
    { registerInstallation: async () => { registrations += 1; return "secret-token"; } },
  );
  await Promise.all([service.ensureRegistered(), service.ensureRegistered()]);
  assert.equal(registrations, 1);
  assert.equal(await service.getToken(), "secret-token");
  await service.reset();
  assert.equal(await service.getToken(), undefined);
});
