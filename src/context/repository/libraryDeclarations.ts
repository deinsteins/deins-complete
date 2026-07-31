import { RepositorySymbol } from "../contextTypes";

export function packageRoot(specifier: string): string {
  const parts = specifier.split("/");
  return specifier.startsWith("@") ? parts.slice(0, 2).join("/") : parts[0];
}

export function declarationEntry(packageJSON: string): string {
  try {
    const manifest = JSON.parse(packageJSON) as { types?: unknown; typings?: unknown };
    const entry = typeof manifest.types === "string" ? manifest.types : typeof manifest.typings === "string" ? manifest.typings : "index.d.ts";
    return entry.startsWith("/") || entry.split("/").includes("..") ? "index.d.ts" : entry.replace(/^\.\//, "");
  } catch {
    return "index.d.ts";
  }
}

export function extractDeclarationSymbols(text: string, names: string[], source: string): RepositorySymbol[] {
  const symbols: RepositorySymbol[] = [];
  for (const name of new Set(names)) {
    const escaped = name.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    const match = new RegExp(`(?:export\\s+)?(?:declare\\s+)?(?:default\\s+)?(?:function|class|interface|type|const|let|var)\\s+${escaped}\\b`).exec(text);
    if (match === null) continue;
    const end = Math.min(text.length, text.indexOf("\n\n", match.index) < 0 ? match.index + 700 : text.indexOf("\n\n", match.index));
    symbols.push({ name, kind: "library-declaration", filePath: `package:${source}`, signature: text.slice(match.index, Math.min(end, match.index + 700)).trim() });
  }
  return symbols;
}
