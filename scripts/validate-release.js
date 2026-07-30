const fs = require("node:fs");
const manifest = JSON.parse(fs.readFileSync("package.json", "utf8"));
const url = manifest.contributes.configuration.properties["deinscomplete.backend.url"].default;
if (url !== "https://api.deinscomplete.web.id" || /localhost|127\.0\.0\.1/.test(url)) throw new Error("production backend URL must be https://api.deinscomplete.web.id");
for (const file of ["package.json", ".vscodeignore"]) if (/AUTH_TOKEN_SECRET=\S+|AI_API_KEY=\S+/.test(fs.readFileSync(file,"utf8"))) throw new Error(`possible secret in ${file}`);
console.log("release validation passed");
