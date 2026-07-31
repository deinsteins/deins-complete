#!/usr/bin/env node
"use strict";

const { randomUUID } = require("node:crypto");
const { readFileSync } = require("node:fs");
const { resolve } = require("node:path");

if (process.argv.includes("--help")) {
  console.log("EVALUATION_CONFIRM_PROVIDER_COST=yes EVALUATION_BACKEND_URL=https://... npm run evaluate:provider");
  process.exit(0);
}
if (process.env.EVALUATION_CONFIRM_PROVIDER_COST !== "yes") {
  throw new Error("Set EVALUATION_CONFIRM_PROVIDER_COST=yes to acknowledge one provider request per fixture.");
}

const baseURL = (process.env.EVALUATION_BACKEND_URL ?? "").replace(/\/+$/, "");
if (!/^https?:\/\/[^/]+/.test(baseURL)) throw new Error("EVALUATION_BACKEND_URL is required.");
const fixtures = JSON.parse(readFileSync(resolve("test/fixtures/completion-quality.json"), "utf8"));

async function json(path, init) {
  const response = await fetch(baseURL + path, { ...init, signal: AbortSignal.timeout(15_000) });
  const body = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(`${path} failed with HTTP ${response.status}`);
  return { body, requestId: response.headers.get("x-request-id") ?? undefined };
}

async function main() {
  const installationId = `evaluation-${randomUUID()}`;
  const registration = await json("/v1/installations/register", {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ installationId, client: { name: "deinscomplete-evaluation", version: "1" } }),
  });
  const token = registration.body.token;
  if (typeof token !== "string" || token === "") throw new Error("Registration returned no token.");

  const results = [];
  for (const fixture of fixtures) {
    const prefix = `${fixture.imports ? fixture.imports + "\n" : ""}${fixture.prefix}`;
    const started = performance.now();
    const response = await json("/v1/completions", {
      method: "POST",
      headers: { authorization: `Bearer ${token}`, "content-type": "application/json" },
      body: JSON.stringify({
        context: { prefix, suffix: "", language: fixture.language, filePath: fixture.filePath, cursorOffset: prefix.length },
        repositoryContext: { files: [], focus: fixture.expectedFocus },
        client: { name: "deinscomplete-evaluation", version: "1" },
      }),
    });
    const text = typeof response.body?.completion?.text === "string" ? response.body.completion.text : "";
    results.push({
      name: fixture.name,
      latencyMs: Math.round(performance.now() - started),
      characters: text.length,
      nonEmpty: text.length > 0,
      clean: !/```|here is|completion:/i.test(text),
      requestId: response.requestId,
    });
  }
  const latencies = results.map((result) => result.latencyMs).sort((a, b) => a - b);
  console.log(JSON.stringify({
    summary: {
      fixtures: results.length,
      nonEmpty: results.filter((result) => result.nonEmpty).length,
      clean: results.filter((result) => result.clean).length,
      p50Ms: percentile(latencies, 0.5),
      p95Ms: percentile(latencies, 0.95),
    },
    results,
  }, null, 2));
}

function percentile(values, percentileValue) {
  return values.length === 0 ? 0 : values[Math.min(values.length - 1, Math.ceil(values.length * percentileValue) - 1)];
}

main().catch((error) => {
  console.error(error instanceof Error ? error.message : "Provider evaluation failed.");
  process.exitCode = 1;
});
