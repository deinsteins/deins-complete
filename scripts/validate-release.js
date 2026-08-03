const fs = require("node:fs");
const path = require("node:path");
const { execFileSync } = require("node:child_process");
const manifest = JSON.parse(fs.readFileSync("package.json", "utf8"));
const url = manifest.contributes.configuration.properties["deinscomplete.backend.url"].default;
if (url !== "https://api.deinscomplete.web.id" || /localhost|127\.0\.0\.1/.test(url)) throw new Error("production backend URL must be https://api.deinscomplete.web.id");
for (const file of ["package.json", ".vscodeignore"]) if (/AUTH_TOKEN_SECRET=\S+|AI_API_KEY=\S+/.test(fs.readFileSync(file,"utf8"))) throw new Error(`possible secret in ${file}`);

const packaged = execFileSync("npx", ["--no-install", "vsce", "ls"], { encoding: "utf8" }).trim().split(/\r?\n/);
const forbidden = packaged.find((file) => file.startsWith("api/") || file.startsWith("out/test/") || /(^|\/)\.env(?:\.|$)/.test(file) || file.endsWith(".vsix"));
if (forbidden !== undefined) throw new Error(`forbidden VSIX file: ${forbidden}`);

const runtimeFiles = ["package.json", ...walk("out/src")];
const secretPatterns = [/-----BEGIN (?:[A-Z ]+ )?PRIVATE KEY-----/, /\bsk-[A-Za-z0-9_-]{20,}\b/, /\bBearer\s+[A-Za-z0-9._-]{24,}\b/, /AUTH_TOKEN_SECRET\s*=\s*[^<\s][^\s]*/];
for (const file of runtimeFiles) {
  const content = fs.readFileSync(file, "utf8");
  if (secretPatterns.some((pattern) => pattern.test(content))) throw new Error(`possible credential in runtime file: ${file}`);
}
console.log("release validation passed");

function walk(directory) {
  return fs.readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const file = path.join(directory, entry.name);
    return entry.isDirectory() ? walk(file) : [file];
  });
}
