import { CompletionContext } from "../context/contextTypes";

export type CompletionRequestMode = "fast" | "full";
export type CompletionIntent =
  | "component-props"
  | "member-access"
  | "function-arguments"
  | "tailwind-class"
  | "object-fields"
  | "import"
  | "function-body"
  | "condition-expression"
  | "type-definition"
  | "general";
// Kept as an alias because feedback and repository ranking already use this name.
export type CompletionFocus = CompletionIntent;

// Full context is reserved for cross-file/framework signals; syntax completion stays fast.
export function completionRequestMode(context: CompletionContext): CompletionRequestMode {
  const focus = completionFocus(context);
  if (focus === "tailwind-class" || focus === "import" || focus === "function-arguments") return "full";
  const jsx = jsxOpeningTag(context.textBeforeCursorOnLine);
  if (jsx !== undefined && !jsx.hasAttributes) return "fast";
  const imports = context.imports ?? context.prefix.slice(0, 10000);
  if (imports === "") return "fast";
  const names = [...imports.matchAll(/(?:import\s+(?:type\s+)?(?:\{\s*)?([A-Za-z_$][\w$]*)|\bfrom\s+[^\n]+\s+import\s+([A-Za-z_$][\w$]*))/g)]
    .flatMap((match) => [match[1], match[2]].filter((name): name is string => name !== undefined));
  const line = context.textBeforeCursorOnLine;
  return names.some((name) => new RegExp(`(?:\\b${escapeRegExp(name)}(?:\\.|\\s*\\()|<${escapeRegExp(name)}\\b|:\\s*${escapeRegExp(name)}\\b)`).test(line)) ? "full" : "fast";
}

export function completionFocus(context: CompletionContext): CompletionFocus {
  const line = context.textBeforeCursorOnLine;
  const nearby = context.prefix.slice(-600);
  if (/\b(?:className|class|@apply)\b/.test(line)) return "tailwind-class";
  if (jsxOpeningTag(line)?.hasAttributes === true) return "component-props";
  if (/\.[A-Za-z_$]*$/.test(line)) return "member-access";
  if (/^\s*(?:import\b|from\s+\S*\s+import\b|(?:const|let|var)\s+.*=\s*require\([^)]*$)/.test(line)) return "import";
  if (/\b(?:interface|type|class|enum|struct)\s+[A-Za-z_$][\w$]*(?:\s+(?:extends|implements)\s+[\w$., ]*)?\s*\{?\s*$/.test(line)) return "type-definition";
  if (/(?:\b(?:if|while|for|switch)\s*\([^)]*|&&\s*|\|\|\s*|\?\s*)$/.test(line)) return "condition-expression";
  if (/[A-Za-z_$][\w$]*\s*\([^)]*$/.test(line)) return "function-arguments";
  if (/(?:\b(?:function|func)\b|=>)[\s\S]{0,300}\{\s*$/.test(nearby)) return "function-body";
  if (/[{,]\s*[A-Za-z_$]*$/.test(line)) return "object-fields";
  return "general";
}

function jsxOpeningTag(line: string): { hasAttributes: boolean } | undefined {
  const match = /<([A-Za-z][\w.:-]*)([^<>]*)$/.exec(line);
  return match === null ? undefined : { hasAttributes: /^\s/.test(match[2]) };
}

function escapeRegExp(value: string): string { return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"); }
