const { readdirSync } = require("node:fs");
const { join } = require("node:path");
const { spawnSync } = require("node:child_process");

function collect(directory) {
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = join(directory, entry.name);
    return entry.isDirectory() ? collect(path) : entry.name.endsWith(".test.js") ? [path] : [];
  });
}

const tests = collect("out/test");
if (tests.length === 0) throw new Error("No compiled tests found.");
process.exit(spawnSync(process.execPath, ["--test", ...tests], { stdio: "inherit" }).status ?? 1);
