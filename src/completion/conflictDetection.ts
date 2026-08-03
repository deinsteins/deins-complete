const inlineCompletionExtensions = new Map([
  ["github.copilot", "GitHub Copilot"],
  ["codeium.codeium", "Windsurf/Codeium"],
  ["tabnine.tabnine-vscode", "Tabnine"],
  ["visualstudioexptteam.vscodeintellicode", "IntelliCode"],
]);

export function conflictingInlineCompletionExtensions(installed: Iterable<string>): string[] {
  return [...installed].flatMap((id) => {
    const name = inlineCompletionExtensions.get(id.toLowerCase());
    return name === undefined ? [] : [name];
  });
}
