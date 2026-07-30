const dependencyLimit = 24;

/** Returns only package names that can improve code-completion context. */
export function relevantDependencies(packageJSON: string, source: string): string[] {
  let manifest: { dependencies?: Record<string, unknown>; devDependencies?: Record<string, unknown> };
  try { manifest = JSON.parse(packageJSON) as typeof manifest; } catch { return []; }
  const declared = new Set([...Object.keys(manifest.dependencies ?? {}), ...Object.keys(manifest.devDependencies ?? {})]);
  const imported = [...source.matchAll(/(?:import(?:[\s\S]*?\s+from\s+)?|require\s*\()\s*["']([^"']+)["']/g)]
    .map((match) => packageName(match[1]))
    .filter((name): name is string => name !== undefined && declared.has(name));
  const hasClassContext = /\b(?:className|class|@apply)\b/.test(source);
  const tailwind = hasClassContext && declared.has("tailwindcss") ? ["tailwindcss"] : [];
  return [...new Set([...imported, ...tailwind])].sort().slice(0, dependencyLimit);
}

function packageName(specifier: string): string | undefined {
  if (specifier.startsWith(".") || specifier.startsWith("/")) return undefined;
  const parts = specifier.split("/");
  return specifier.startsWith("@") ? parts.slice(0, 2).join("/") : parts[0];
}
