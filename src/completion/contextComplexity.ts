import { CompletionContext } from "../context/contextTypes";

export type CompletionRequestMode = "fast" | "full";

// The only full-mode signal is an imported name at the cursor. Everything
// else stays fast; repository context is opportunistic, never a requirement.
export function completionRequestMode(context: CompletionContext): CompletionRequestMode {
  const imports = context.imports ?? context.prefix.slice(0, 10000);
  if (imports === "") return "fast";
  const names = [...imports.matchAll(/(?:import\s+(?:type\s+)?(?:\{\s*)?([A-Za-z_$][\w$]*)|\bfrom\s+[^\n]+\s+import\s+([A-Za-z_$][\w$]*))/g)]
    .flatMap((match) => [match[1], match[2]].filter((name): name is string => name !== undefined));
  const line = context.textBeforeCursorOnLine;
  return names.some((name) => new RegExp(`\\b${escapeRegExp(name)}(?:\\.|\\s*\\()`).test(line)) ? "full" : "fast";
}

function escapeRegExp(value: string): string { return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"); }
