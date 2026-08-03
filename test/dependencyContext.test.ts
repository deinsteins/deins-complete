import assert from "node:assert/strict";
import test from "node:test";
import { relevantDependencies } from "../src/context/repository/dependencyContext";

const manifest = JSON.stringify({
  dependencies: { antd: "x", "@mui/material": "x", lodash: "x" },
  devDependencies: { tailwindcss: "x" },
});

test("dependency context includes imported packages and Tailwind class context", () => {
  assert.deepEqual(relevantDependencies(manifest, 'import { Button } from "antd"; import debounce from "lodash/debounce"; <div className="" />'), ["antd", "lodash", "tailwindcss"]);
});

test("dependency context does not include unrelated packages or parse invalid manifests", () => {
  assert.deepEqual(relevantDependencies(manifest, "const value = 1;"), []);
  assert.deepEqual(relevantDependencies("not-json", 'import "antd"'), []);
});

test("dependency context recognizes declared framework from source signals", () => {
  const frameworks = JSON.stringify({ dependencies: { react: "x", next: "x", vue: "x", "@angular/core": "x" } });
  assert.deepEqual(relevantDependencies(frameworks, "const [value, setValue] = useState(0); return <ProductCard />"), ["next", "react"]);
  assert.deepEqual(relevantDependencies(frameworks, "@Component({ selector: 'app-root' })"), ["@angular/core"]);
});
