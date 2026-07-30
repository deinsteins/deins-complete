export function tsconfigAliasTargets(tsconfig: string, specifier: string): string[] {
  let config: { compilerOptions?: { baseUrl?: unknown; paths?: Record<string, unknown> } };
  try { config = JSON.parse(stripComments(tsconfig)) as typeof config; } catch { return []; }
  const options = config.compilerOptions;
  if (options?.paths === undefined) return [];
  const base = typeof options.baseUrl === "string" ? options.baseUrl : ".";
  const targets: string[] = [];
  for (const [pattern, values] of Object.entries(options.paths)) {
    const marker = pattern.indexOf("*");
    const match = marker < 0 ? (specifier === pattern ? "" : undefined) : (specifier.startsWith(pattern.slice(0, marker)) && specifier.endsWith(pattern.slice(marker + 1)) ? specifier.slice(marker, specifier.length - pattern.slice(marker + 1).length) : undefined);
    if (match === undefined || !Array.isArray(values)) continue;
    for (const value of values) if (typeof value === "string") targets.push(`${base}/${value.replace("*", match)}`.replace(/^\.\//, ""));
  }
  return targets;
}

function stripComments(value: string): string { return value.replace(/\/\*[\s\S]*?\*\/|(^|[^:])\/\/.*$/gm, "$1"); }
