import { CodeStyle } from "./contextTypes";

export function inferCodeStyle(text: string): CodeStyle {
  const lines = text.slice(0, 20000).split(/\r?\n/);
  const tabLines = lines.filter((line) => /^\t+\S/.test(line)).length;
  const widths = lines.map((line) => /^( +)\S/.exec(line)?.[1].length ?? 0).filter((width) => width > 0 && width <= 16);
  const indentation = tabLines > widths.length ? "tabs" : "spaces";
  const strings = text.slice(0, 20000).match(/"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'/g) ?? [];
  const single = strings.filter((value) => value.startsWith("'")).length;
  const double = strings.length - single;
  const codeLines = lines.map((line) => line.trim()).filter((line) => line !== "" && !/^(?:\/\/|#|\*|\{|\})/.test(line));
  const semicolons = codeLines.filter((line) => line.endsWith(";")).length;
  return {
    indentation,
    ...(indentation === "spaces" && widths.length > 0 ? { indentSize: Math.min(8, Math.min(...widths)) } : {}),
    ...(single === double ? {} : { quote: single > double ? "single" as const : "double" as const }),
    ...(codeLines.length >= 2 && semicolons * 2 !== codeLines.length ? { semicolons: semicolons * 2 > codeLines.length ? "always" as const : "never" as const } : {}),
  };
}
