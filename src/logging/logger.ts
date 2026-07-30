import * as vscode from "vscode";

export class Logger implements vscode.Disposable {
  private readonly channel = vscode.window.createOutputChannel("DeinsComplete");

  debug(message: string): void { this.write("DEBUG", message); }
  info(message: string): void { this.write("INFO", message); }
  warn(message: string): void { this.write("WARN", message); }

  error(message: string, error?: unknown): void {
    const detail = error instanceof Error ? error.message : error === undefined ? "" : String(error);
    this.write("ERROR", detail ? `${message}: ${detail}` : message);
  }

  show(): void { this.channel.show(true); }
  dispose(): void { this.channel.dispose(); }

  private write(level: string, message: string): void {
    this.channel.appendLine(`[${level}] ${this.sanitize(message)}`);
  }

  private sanitize(message: string): string {
    return message
      .replace(/(authorization\s*[:=]\s*)([^\s,;]+)/gi, "$1[REDACTED]")
      .replace(/((?:api[_ -]?key|access[_ -]?token|token)\s*[:=]\s*)([^\s,;]+)/gi, "$1[REDACTED]");
  }
}
