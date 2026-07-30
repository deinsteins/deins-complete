import { CompletionContext } from "../context/contextTypes";

export type CompletionRequestMode = "fast" | "full";
export type CompletionFocus = "component-props" | "member-access" | "function-arguments" | "tailwind-class" | "object-fields" | "general";

// Full context is reserved for cross-file/framework signals; syntax completion stays fast.
export function completionRequestMode(context: CompletionContext): CompletionRequestMode {
  const imports = context.imports ?? context.prefix.slice(0, 10000);
  if (imports === "") return "fast";
  const names = [...imports.matchAll(/(?:import\s+(?:type\s+)?(?:\{\s*)?([A-Za-z_$][\w$]*)|\bfrom\s+[^\n]+\s+import\s+([A-Za-z_$][\w$]*))/g)]
    .flatMap((match) => [match[1], match[2]].filter((name): name is string => name !== undefined));
  const line = context.textBeforeCursorOnLine;
  if (completionFocus(context) === "tailwind-class") return "full";
  return names.some((name) => new RegExp(`(?:\\b${escapeRegExp(name)}(?:\\.|\\s*\\()|<${escapeRegExp(name)}\\b)`).test(line)) ? "full" : "fast";
}

export function completionFocus(context: CompletionContext): CompletionFocus {
  const line = context.textBeforeCursorOnLine;
  if (/\b(?:className|class|@apply)\b/.test(line)) return "tailwind-class";
  if (/<[A-Z][\w.:-]*\s+[^>]*$/.test(line)) return "component-props";
  if (/\.[A-Za-z_$]*$/.test(line)) return "member-access";
  if (/[A-Za-z_$][\w$]*\s*\([^)]*$/.test(line)) return "function-arguments";
  if (/[{,]\s*[A-Za-z_$]*$/.test(line)) return "object-fields";
  return "general";
}

function escapeRegExp(value: string): string { return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"); }
