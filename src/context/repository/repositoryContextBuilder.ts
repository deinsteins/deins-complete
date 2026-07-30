import { createHash } from "node:crypto";
import * as path from "node:path";
import * as vscode from "vscode";
import { CompletionContext, RepositoryContext, RepositoryContextFile, RepositoryContextReason, RepositorySymbol } from "../contextTypes";
import { RepositoryContextSettings } from "./repositoryContextConfig";
import { relevantDependencies } from "./dependencyContext";
import { tsconfigAliasTargets } from "./projectConfig";
import { CompletionFocus, completionFocus } from "../../completion/contextComplexity";

const maxFileBytes = 1024 * 1024;
const perFileCharacters = 3000;
const ignoredSegments = new Set(["node_modules", ".git", "dist", "build", ".next", "coverage", "vendor", "target"]);
const sensitiveNames = /^(\.env(?:\..*)?|id_rsa|id_ed25519|credentials(?:\..*)?|secrets(?:\..*)?)$/i;
const sensitiveExtensions = /\.(pem|key|p12|pfx)$/i;

type ImportReference = { specifier: string; names: string[] };
type Candidate = { uri: vscode.Uri; reason: RepositoryContextReason; names: string[]; score: number };
type CachedText = { text: string; version?: number };

/** Lightweight workspace signals only: imports and already-open/recent documents. */
export class RepositoryContextBuilder {
  private readonly recent = new Map<string, vscode.Uri>();
  private readonly cache = new Map<string, CachedText>();
  private readonly dependencyCache = new Map<string, CachedText>();
  private readonly tsconfigCache = new Map<string, CachedText>();
  private readonly stats = { requests: 0, success: 0, partial: 0, timedOut: 0, filesIncluded: 0, totalDurationMs: 0, lastFiles: 0, lastCharacters: 0, lastDurationMs: 0 };

  constructor(private readonly settings: RepositoryContextSettings) {}

  record(document: vscode.TextDocument): void {
    if (!this.isWorkspaceFile(document.uri)) return;
    this.recent.delete(document.uri.toString());
    this.recent.set(document.uri.toString(), document.uri);
    while (this.recent.size > 20) this.recent.delete(this.recent.keys().next().value as string);
  }

  invalidate(uri: vscode.Uri): void {
    this.cache.delete(uri.toString());
    if (path.posix.basename(uri.path) === "package.json") this.dependencyCache.delete(uri.toString());
    if (path.posix.basename(uri.path) === "tsconfig.json") this.tsconfigCache.delete(uri.toString());
  }
  getStats() { return { ...this.stats }; }

  async build(document: vscode.TextDocument, current: CompletionContext, signal: AbortSignal): Promise<RepositoryContext | undefined> {
    const limits = this.settings.getRepositoryContextLimits();
    if (!limits.enabled || signal.aborted || !this.isWorkspaceFile(document.uri)) return undefined;
    this.stats.requests++;
    const started = performance.now();
    const deadline = started + limits.timeoutMs;
    const expired = () => signal.aborted || performance.now() >= deadline;
    this.record(document);
    const candidates = new Map<string, Candidate>();
    const dependencies = await this.readDependencies(document);
    const add = (candidate: Candidate) => {
      if (candidate.uri.toString() === document.uri.toString() || !this.isEligible(candidate.uri)) return;
      const previous = candidates.get(candidate.uri.toString());
      if (previous === undefined || candidate.score > previous.score) candidates.set(candidate.uri.toString(), candidate);
    };

    for (const reference of extractLocalImports(document.getText())) {
      if (expired()) break;
      const uri = await this.resolveImport(document.uri, reference.specifier);
      if (uri !== undefined) add({ uri, reason: "import", names: reference.names, score: 110 });
    }
    if (dependencies.includes("tailwindcss")) {
      const config = await this.findWorkspaceFile(document.uri, ["tailwind.config.ts", "tailwind.config.js", "tailwind.config.cjs", "tailwind.config.mjs"]);
      if (config !== undefined) add({ uri: config, reason: "framework-config", names: ["theme", "colors", "extend"], score: 100 });
    }
    for (const uri of await this.siblingTests(document.uri)) add({ uri, reason: "symbol-reference", names: identifiersNearCursor(current), score: 80 });
    for (const uri of [...this.recent.values()].reverse()) {
      if (expired()) break;
      const open = vscode.workspace.textDocuments.find((item) => item.uri.toString() === uri.toString());
      if (open !== undefined && open.uri.toString() !== document.uri.toString()) {
        add({ uri, reason: "open-file", names: identifiersNearCursor(current), score: 40 + (open.languageId === document.languageId ? 10 : 0) });
      }
    }

    const files: RepositoryContextFile[] = [];
    const symbols: RepositorySymbol[] = [];
    let characters = 0;
    for (const candidate of [...candidates.values()].sort((a, b) => b.score - a.score)) {
      if (expired() || files.length >= limits.maxFiles || characters >= limits.maxCharacters) break;
      const text = await this.readText(candidate.uri);
      if (text === undefined || containsSensitiveContent(text)) continue;
      const remaining = Math.min(perFileCharacters, limits.maxCharacters - characters);
      const content = selectSnippet(text, candidate.names, remaining);
      if (content === "") continue;
      const filePath = safeRelativePath(candidate.uri);
      files.push({ path: filePath, language: languageFor(candidate.uri), content, reason: candidate.reason });
      characters += content.length;
      symbols.push(...extractSymbols(text, filePath, candidate.names));
    }
    const focus = completionFocus(current);
    const diagnostics = vscode.languages.getDiagnostics(document.uri).slice(0, 3).map((item) => item.message.replace(/\s+/g, " ").slice(0, 300));
    if (!expired()) symbols.push(...await this.librarySymbols(document, current, focus, deadline));
    if (signal.aborted) return undefined;
    const boundedSymbols = dedupeSymbols(symbols).slice(0, 20);
    const fingerprint = createHash("sha256").update(files.map((file) => `${file.path}\0${file.content}`).join("\0")).update("\0").update(dependencies.join("\0")).update("\0").update(boundedSymbols.map((symbol) => `${symbol.filePath}\0${symbol.name}\0${symbol.signature ?? ""}`).join("\0")).digest("hex");
    const durationMs = Math.round(performance.now() - started);
    const timedOut = durationMs >= limits.timeoutMs;
    this.stats.success++;
    this.stats.filesIncluded += files.length;
    this.stats.totalDurationMs += durationMs;
    this.stats.lastFiles = files.length;
    this.stats.lastCharacters = characters;
    this.stats.lastDurationMs = durationMs;
    if (timedOut) this.stats.timedOut++;
    if (timedOut || files.length < candidates.size) this.stats.partial++;
    return { files, symbols: boundedSymbols, dependencies, focus, diagnostics, fingerprint, durationMs, timedOut };
  }

  private async readDependencies(document: vscode.TextDocument): Promise<string[]> {
    const folder = vscode.workspace.getWorkspaceFolder(document.uri);
    if (folder === undefined) return [];
    const manifest = vscode.Uri.joinPath(folder.uri, "package.json");
    let text: string | undefined;
    const open = vscode.workspace.textDocuments.find((item) => item.uri.toString() === manifest.toString());
    if (open !== undefined) {
      const cached = this.dependencyCache.get(manifest.toString());
      text = cached?.version === open.version ? cached.text : open.getText();
      this.dependencyCache.set(manifest.toString(), { text, version: open.version });
    } else {
      const cached = this.dependencyCache.get(manifest.toString());
      if (cached !== undefined) text = cached.text;
      else try { text = new TextDecoder().decode(await vscode.workspace.fs.readFile(manifest)); this.dependencyCache.set(manifest.toString(), { text }); } catch { return []; }
    }
    return relevantDependencies(text, document.getText());
  }

  private async resolveImport(from: vscode.Uri, specifier: string): Promise<vscode.Uri | undefined> {
    const roots = [from.with({ path: path.posix.join(path.posix.dirname(from.path), specifier) })];
    const folder = vscode.workspace.getWorkspaceFolder(from);
    if (folder !== undefined) for (const target of await this.aliasTargets(folder.uri, specifier)) roots.push(folder.uri.with({ path: path.posix.join(folder.uri.path, target) }));
    const extensions = ["", ".ts", ".tsx", ".js", ".jsx", ".mts", ".cts", ".mjs", ".cjs", "/index.ts", "/index.tsx", "/index.js", "/index.jsx"];
    for (const root of roots) for (const extension of extensions) {
      const uri = root.with({ path: root.path + extension });
      if (!this.isEligible(uri)) continue;
      try { const stat = await vscode.workspace.fs.stat(uri); if (stat.size <= maxFileBytes && (stat.type & vscode.FileType.File) !== 0) return uri; } catch { /* unresolved imports are normal */ }
    }
    return undefined;
  }

  private async aliasTargets(folder: vscode.Uri, specifier: string): Promise<string[]> {
    const config = vscode.Uri.joinPath(folder, "tsconfig.json");
    const open = vscode.workspace.textDocuments.find((item) => item.uri.toString() === config.toString());
    let text: string | undefined;
    if (open !== undefined) { const cached = this.tsconfigCache.get(config.toString()); text = cached?.version === open.version ? cached.text : open.getText(); this.tsconfigCache.set(config.toString(), { text, version: open.version }); }
    else { const cached = this.tsconfigCache.get(config.toString()); if (cached !== undefined) text = cached.text; else try { text = new TextDecoder().decode(await vscode.workspace.fs.readFile(config)); this.tsconfigCache.set(config.toString(), { text }); } catch { return []; } }
    return tsconfigAliasTargets(text, specifier);
  }

  private async findWorkspaceFile(from: vscode.Uri, names: string[]): Promise<vscode.Uri | undefined> {
    const folder = vscode.workspace.getWorkspaceFolder(from);
    if (folder === undefined) return undefined;
    for (const name of names) {
      const uri = vscode.Uri.joinPath(folder.uri, name);
      try { const stat = await vscode.workspace.fs.stat(uri); if (stat.size <= maxFileBytes && (stat.type & vscode.FileType.File) !== 0) return uri; } catch { /* optional config */ }
    }
    return undefined;
  }

  private async siblingTests(uri: vscode.Uri): Promise<vscode.Uri[]> {
    const extension = path.posix.extname(uri.path); const base = uri.path.slice(0, -extension.length);
    if (/\.(?:test|spec)$/.test(base)) return [];
    const tests = [".test", ".spec"].map((suffix) => uri.with({ path: base + suffix + extension }));
    const found: vscode.Uri[] = [];
    for (const test of tests) try { const stat = await vscode.workspace.fs.stat(test); if ((stat.type & vscode.FileType.File) !== 0 && stat.size <= maxFileBytes) found.push(test); } catch { /* optional test */ }
    return found;
  }

  private async librarySymbols(document: vscode.TextDocument, current: CompletionContext, focus: CompletionFocus, deadline: number): Promise<RepositorySymbol[]> {
    const names = new Set(identifiersNearCursor(current));
    const imports = extractExternalImports(document.getText()).filter((reference) => reference.names.some((name) => names.has(name))).slice(0, 3);
    const remaining = Math.min(20, Math.max(0, deadline - performance.now()));
    if (imports.length === 0 || remaining === 0) return [];
    const source = `package:${imports[0].specifier}`;
    const members = focus === "component-props" || focus === "member-access"
      ? await this.providerMembers(document, current, source, Math.min(remaining, 12)) : [];
    const hoverRemaining = Math.min(20, Math.max(0, deadline - performance.now()));
    if (hoverRemaining === 0) return members;
    const hover = await beforeDeadline(vscode.commands.executeCommand<vscode.Hover[]>("vscode.executeHoverProvider", document.uri, document.positionAt(current.cursorOffset)), hoverRemaining);
    const signature = hover === undefined ? "" : hover.map((item) => item.contents.map(hoverContent).join("\n")).join("\n").replace(/\s+/g, " ").slice(0, 500);
    return [...members, ...imports.flatMap((reference) => reference.names.filter((name) => names.has(name)).map((name) => ({ name, kind: "library-symbol", filePath: `package:${reference.specifier}`, signature: `${name} from ${reference.specifier}${signature === "" ? "" : ` — ${signature}`}` })))];
  }

  private async providerMembers(document: vscode.TextDocument, current: CompletionContext, source: string, milliseconds: number): Promise<RepositorySymbol[]> {
    const result = await beforeDeadline(vscode.commands.executeCommand<vscode.CompletionList>("vscode.executeCompletionItemProvider", document.uri, document.positionAt(current.cursorOffset)), milliseconds);
    return (result?.items ?? []).slice(0, 12).map((item) => ({ name: completionLabel(item), kind: completionKind(item.kind), filePath: source, signature: typeof item.detail === "string" ? item.detail.slice(0, 300) : undefined }));
  }

  private async readText(uri: vscode.Uri): Promise<string | undefined> {
    const open = vscode.workspace.textDocuments.find((document) => document.uri.toString() === uri.toString());
    if (open !== undefined) {
      const cached = this.cache.get(uri.toString());
      if (cached?.version === open.version) return cached.text;
      const text = open.getText();
      this.cache.set(uri.toString(), { text, version: open.version });
      return text;
    }
    const cached = this.cache.get(uri.toString());
    if (cached !== undefined) return cached.text;
    try {
      const bytes = await vscode.workspace.fs.readFile(uri);
      const text = new TextDecoder().decode(bytes);
      this.cache.set(uri.toString(), { text });
      while (this.cache.size > 40) this.cache.delete(this.cache.keys().next().value as string);
      return text;
    } catch { return undefined; }
  }

  private isWorkspaceFile(uri: vscode.Uri): boolean { return vscode.workspace.getWorkspaceFolder(uri) !== undefined; }
  private isEligible(uri: vscode.Uri): boolean {
    if (!this.isWorkspaceFile(uri)) return false;
    const parts = uri.path.split("/");
    const name = parts.at(-1) ?? "";
    return !parts.some((part) => ignoredSegments.has(part)) && !sensitiveNames.test(name) && !sensitiveExtensions.test(name);
  }
}

function extractLocalImports(text: string): ImportReference[] {
  const imports: ImportReference[] = [];
  const pattern = /(?:import\s+([^'";]+?)\s+from\s+|import\s*\(|require\s*\()\s*["'](\.{1,2}\/[\w./-]+)["']/g;
  for (const match of text.matchAll(pattern)) imports.push({ specifier: match[2], names: (match[1] ?? "").match(/[A-Za-z_$][\w$]*/g) ?? [] });
  return imports;
}

function extractExternalImports(text: string): ImportReference[] {
  const imports: ImportReference[] = [];
  const pattern = /import\s+([^'";]+?)\s+from\s+["']((?!\.)[^"']+)["']/g;
  for (const match of text.matchAll(pattern)) imports.push({ specifier: match[2], names: (match[1] ?? "").match(/[A-Za-z_$][\w$]*/g) ?? [] });
  return imports;
}

function identifiersNearCursor(context: CompletionContext): string[] {
  return `${context.currentLine}\n${context.prefix.slice(-500)}`.match(/[A-Za-z_$][\w$]*/g) ?? [];
}

function selectSnippet(text: string, names: string[], limit: number): string {
  if (text.length <= limit) return text;
  const lines = text.split(/\r?\n/);
  const index = names.map((name) => lines.findIndex((line) => new RegExp(`\\b${escapeRegExp(name)}\\b`).test(line))).find((value) => value !== undefined && value >= 0);
  const selected = index === undefined ? lines.filter((line) => /^\s*export\s+(?:function|class|interface|type|const)\b/.test(line)).slice(0, 20) : lines.slice(Math.max(0, index - 3), index + 12);
  return selected.join("\n").slice(0, limit);
}

function extractSymbols(text: string, filePath: string, names: string[]): RepositorySymbol[] {
  const wanted = new Set(names);
  const symbols: RepositorySymbol[] = [];
  for (const line of text.split(/\r?\n/)) {
    const match = line.match(/^\s*export\s+(?:default\s+)?(function|class|interface|type|const)\s+([A-Za-z_$][\w$]*)/);
    if (match !== null && (wanted.size === 0 || wanted.has(match[2]))) symbols.push({ name: match[2], kind: match[1], filePath, signature: line.trim().slice(0, 300) });
  }
  return symbols;
}

function dedupeSymbols(symbols: RepositorySymbol[]): RepositorySymbol[] {
  return [...new Map(symbols.map((symbol) => [`${symbol.filePath}\0${symbol.kind}\0${symbol.name}`, symbol])).values()];
}
function safeRelativePath(uri: vscode.Uri): string { return vscode.workspace.asRelativePath(uri, vscode.workspace.workspaceFolders?.length !== 1).replace(/\\/g, "/"); }
function languageFor(uri: vscode.Uri): string { return path.posix.extname(uri.path).slice(1) || "text"; }
function containsSensitiveContent(text: string): boolean { return /-----BEGIN (?:[A-Z ]+ )?PRIVATE KEY-----|(?:API_KEY|SECRET|PASSWORD)\s*=/i.test(text); }
function escapeRegExp(value: string): string { return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"); }
function hoverContent(value: vscode.MarkedString | vscode.MarkdownString): string { return typeof value === "string" ? value : "language" in value ? `${value.language} ${value.value}` : value.value; }
function completionLabel(item: vscode.CompletionItem): string { return typeof item.label === "string" ? item.label : item.label.label; }
function completionKind(kind: vscode.CompletionItemKind | undefined): string { return kind === vscode.CompletionItemKind.Method ? "method" : kind === vscode.CompletionItemKind.Property || kind === vscode.CompletionItemKind.Field ? "property" : "member"; }
async function beforeDeadline<T>(action: Thenable<T>, milliseconds: number): Promise<T | undefined> {
  let timer: ReturnType<typeof setTimeout> | undefined;
  try { return await Promise.race([Promise.resolve(action).catch(() => undefined), new Promise<undefined>((resolve) => { timer = setTimeout(() => resolve(undefined), milliseconds); })]); } finally { if (timer !== undefined) clearTimeout(timer); }
}
