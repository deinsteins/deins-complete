export interface TextWindow {
  text: string;
  truncated: boolean;
}

export function takePrefixWindow(text: string, maximumCharacters: number): TextWindow {
  if (text.length <= maximumCharacters) {
    return { text, truncated: false };
  }

  const window = text.slice(-maximumCharacters);
  const firstLineBreak = window.indexOf("\n");
  return { text: firstLineBreak >= 0 ? window.slice(firstLineBreak + 1) : window, truncated: true };
}

export function takeSuffixWindow(text: string, maximumCharacters: number): TextWindow {
  if (text.length <= maximumCharacters) {
    return { text, truncated: false };
  }

  const window = text.slice(0, maximumCharacters);
  const lastLineBreak = window.lastIndexOf("\n");
  return { text: lastLineBreak > 0 ? window.slice(0, lastLineBreak + 1) : window, truncated: true };
}
