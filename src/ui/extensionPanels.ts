import * as vscode from "vscode";
import { AccountService } from "../account/accountService";
import { Logger } from "../logging/logger";

let accountPanel: vscode.WebviewPanel | undefined;
let welcomePanel: vscode.WebviewPanel | undefined;

export function showAccountCenter(account: AccountService, refreshStatus: () => Promise<void>, logger: Logger): void {
  if (accountPanel !== undefined) { accountPanel.reveal(); void loadAccount(accountPanel, account); return; }
  const panel = vscode.window.createWebviewPanel("deinscomplete.accountCenter", "DeinsComplete Account", vscode.ViewColumn.Active, { enableScripts: true });
  accountPanel = panel;
  panel.webview.html = accountHTML(panel.webview);
  panel.onDidDispose(() => { accountPanel = undefined; });
  panel.webview.onDidReceiveMessage(async (message: unknown) => {
    if (!isAction(message)) return;
    try {
      if (message.action === "signIn") await vscode.commands.executeCommand("deinscomplete.signIn");
      if (message.action === "signOut") await vscode.commands.executeCommand("deinscomplete.signOut");
      if (message.action === "diagnostics") await vscode.commands.executeCommand("deinscomplete.diagnostics");
      if (message.action === "settings") await vscode.commands.executeCommand("workbench.action.openSettings", "@ext:deinscomplete.deinscomplete");
      if (message.action === "welcome") showWelcome();
      if (message.action === "revoke" && typeof message.id === "string") {
        const confirmed = await vscode.window.showWarningMessage("Revoke this DeinsComplete installation? It will stop receiving completions.", { modal: true }, "Revoke");
        if (confirmed === "Revoke") await account.revokeInstallation(message.id);
      }
      await refreshStatus();
      await loadAccount(panel, account);
    } catch (error) {
      logger.error("Account Center action failed", error);
      void panel.webview.postMessage({ type: "error", message: "The action could not be completed. Try again." });
    }
  });
  void loadAccount(panel, account);
}

export function showWelcome(): void {
  if (welcomePanel !== undefined) { welcomePanel.reveal(); return; }
  const panel = vscode.window.createWebviewPanel("deinscomplete.welcome", "Welcome to DeinsComplete", vscode.ViewColumn.Active, { enableScripts: true });
  welcomePanel = panel;
  panel.webview.html = welcomeHTML(panel.webview);
  panel.onDidDispose(() => { welcomePanel = undefined; });
  panel.webview.onDidReceiveMessage(async (message: unknown) => {
    if (!isAction(message)) return;
    if (message.action === "signIn") await vscode.commands.executeCommand("deinscomplete.signIn");
    if (message.action === "account") await vscode.commands.executeCommand("deinscomplete.accountCenter");
    if (message.action === "trigger") await vscode.commands.executeCommand("deinscomplete.triggerCompletion");
    if (message.action === "settings") await vscode.commands.executeCommand("workbench.action.openSettings", "@ext:deinscomplete.deinscomplete");
  });
}

async function loadAccount(panel: vscode.WebviewPanel, account: AccountService): Promise<void> {
  try {
    const status = await account.getStatus();
    void panel.webview.postMessage({ type: "account", status });
  } catch {
    void panel.webview.postMessage({ type: "error", message: "Account information is temporarily unavailable." });
  }
}

function isAction(value: unknown): value is { action: string; id?: string } {
  return typeof value === "object" && value !== null && "action" in value && typeof (value as { action?: unknown }).action === "string";
}

function accountHTML(webview: vscode.Webview): string {
  const nonce = randomNonce();
  return `<!doctype html><html lang="en"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src ${webview.cspSource} 'unsafe-inline'; script-src 'nonce-${nonce}'"><style>${styles}</style></head>
  <body><main><header><div><p class="eyebrow">DEINSCOMPLETE</p><h1>Account Center</h1><p class="muted">Your plan, usage, and linked installations.</p></div><span id="plan" class="badge">Loading</span></header>
  <div id="notice" class="notice" role="status" aria-live="polite">Loading account…</div>
  <section id="content" hidden><div class="grid"><article><span class="label">Monthly usage</span><strong id="usage">—</strong><div class="progress" aria-label="Monthly quota"><i id="bar"></i></div><small id="remaining"></small></article><article><span class="label">Account</span><strong id="email">—</strong><small id="features"></small></article></div>
  <section class="section"><div class="section-title"><div><h2>Installations</h2><p class="muted">Revoke devices you no longer use.</p></div><button class="secondary" id="refresh">Refresh</button></div><div id="installations" class="devices"></div></section></section>
  <div class="actions"><button id="signin">Sign in</button><button class="secondary" id="signout">Sign out</button><button class="secondary" id="diagnostics">Diagnostics</button><button class="secondary" id="settings">Settings</button><button class="ghost" id="welcome">How it works</button></div></main>
  <script nonce="${nonce}">const vscode=acquireVsCodeApi(),$=id=>document.getElementById(id),send=(action,id)=>vscode.postMessage({action,id});
  for(const id of ['signin','signout','diagnostics','settings','welcome','refresh'])$(id).onclick=()=>send(id==='refresh'?'refresh':id);
  window.addEventListener('message',event=>{const m=event.data;if(m.type==='error'){ $('notice').textContent=m.message;$('notice').className='notice error';return }if(m.type!=='account')return;const s=m.status;$('notice').className='notice';if(!s){$('notice').textContent='Sign in to link this installation and share your plan across devices.';$('content').hidden=true;$('plan').textContent='Guest';$('signin').hidden=false;$('signout').hidden=true;return}$('notice').textContent='Account connected';$('content').hidden=false;$('signin').hidden=true;$('signout').hidden=false;const l=s.entitlements.limits,total=Math.max(0,l.monthlyCompletions),used=Math.max(0,l.used),remaining=Math.max(0,l.remaining),percent=total?Math.min(100,Math.round(used/total*100)):0;$('plan').textContent=s.account.plan.code.toUpperCase();$('usage').textContent=used.toLocaleString()+' / '+total.toLocaleString();$('remaining').textContent=remaining.toLocaleString()+' remaining · '+(100-percent)+'% available';$('bar').style.transform='scaleX('+(percent/100)+')';$('email').textContent=s.account.user.email;$('features').textContent=[s.entitlements.features.streaming?'Streaming':'Standard',s.entitlements.features.repositoryContext?'Repository context':'Current-file context'].join(' · ');$('installations').replaceChildren(...s.installations.map(i=>{const row=document.createElement('div');row.className='device';const info=document.createElement('div'),title=document.createElement('strong'),meta=document.createElement('small'),button=document.createElement('button');title.textContent='Installation '+i.id.slice(0,8);meta.textContent=i.status+' · Last used '+(i.lastSeenAt?new Date(i.lastSeenAt).toLocaleString():'never');info.append(title,meta);button.textContent='Revoke';button.className='danger';button.onclick=()=>send('revoke',i.id);row.append(info,button);return row}))});</script></body></html>`;
}

function welcomeHTML(webview: vscode.Webview): string {
  const nonce = randomNonce();
  return `<!doctype html><html lang="en"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src ${webview.cspSource} 'unsafe-inline'; script-src 'nonce-${nonce}'"><style>${styles}.demo{font-family:var(--vscode-editor-font-family);font-size:16px;line-height:1.8}.ghost{color:var(--vscode-editorGhostText-foreground);opacity:.75}.steps{counter-reset:step;display:grid;gap:12px}.steps article{display:grid;grid-template-columns:32px 1fr;gap:12px}.steps article:before{counter-increment:step;content:counter(step);display:grid;place-items:center;width:28px;height:28px;border-radius:50%;background:var(--vscode-button-background);color:var(--vscode-button-foreground);font-weight:700}</style></head><body><main><header><div><p class="eyebrow">AI COMPLETION, IN YOUR FLOW</p><h1>Write the next line faster.</h1><p class="muted">DeinsComplete reads a bounded context around your cursor and shows a suggestion as native ghost text.</p></div><span class="badge">PRIVATE BETA</span></header><div class="grid"><article class="demo"><span>const product = await productService.</span><span class="ghost">getById(productId);</span><br><span class="muted">Press Tab to accept</span></article><article><span class="label">Privacy boundary</span><strong>Small, relevant context</strong><small>No repository indexing. Related workspace snippets can be disabled in Settings.</small></article></div><section class="section"><h2>Get started</h2><div class="steps"><article><div><strong>Sign in</strong><p class="muted">Connect your private-beta account and installation.</p></div></article><article><div><strong>Start typing</strong><p class="muted">Pause briefly after a member access, function call, or assignment.</p></div></article><article><div><strong>Accept with Tab</strong><p class="muted">Use Esc to dismiss. Trigger manually whenever you need a suggestion.</p></div></article></div></section><div class="actions"><button id="signin">Sign in</button><button class="secondary" id="account">Account Center</button><button class="secondary" id="trigger">Trigger completion</button><button class="ghost" id="settings">Privacy settings</button></div></main><script nonce="${nonce}">const vscode=acquireVsCodeApi();for(const id of ['signin','account','trigger','settings'])document.getElementById(id).onclick=()=>vscode.postMessage({action:id});</script></body></html>`;
}

const styles = `:root{color-scheme:light dark}*{box-sizing:border-box}body{margin:0;color:var(--vscode-foreground);background:var(--vscode-editor-background);font:var(--vscode-font-size)/1.5 var(--vscode-font-family)}main{max-width:880px;margin:0 auto;padding:48px 28px 72px}header,.section-title,.device,.actions{display:flex;align-items:center;justify-content:space-between;gap:16px}h1{font-size:clamp(30px,6vw,54px);line-height:1.05;letter-spacing:-.04em;margin:4px 0 12px}h2{margin:0 0 8px;font-size:20px}.eyebrow,.label{font-size:12px;letter-spacing:.12em;text-transform:uppercase;color:var(--vscode-descriptionForeground)}.muted,small{color:var(--vscode-descriptionForeground)}small{display:block;margin-top:8px}.badge{border:1px solid var(--vscode-widget-border);border-radius:999px;padding:6px 10px;font-weight:700}.notice{margin:28px 0;padding:12px 14px;border-left:3px solid var(--vscode-focusBorder);background:var(--vscode-textBlockQuote-background)}.notice.error{border-color:var(--vscode-errorForeground)}.grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:16px}.grid article,.section{border:1px solid var(--vscode-widget-border);border-radius:10px;padding:20px;background:var(--vscode-sideBar-background)}article strong{display:block;font-size:18px;margin-top:6px}.section{margin-top:16px}.devices{display:grid;gap:8px;margin-top:16px}.device{padding:12px;border:1px solid var(--vscode-widget-border);border-radius:8px}.progress{height:7px;margin-top:16px;overflow:hidden;border-radius:999px;background:var(--vscode-progressBar-background);opacity:.45}.progress i{display:block;width:100%;height:100%;transform-origin:left;background:var(--vscode-button-background);transition:transform .2s ease}.actions{justify-content:flex-start;flex-wrap:wrap;margin-top:24px}button{min-height:36px;padding:7px 14px;border:1px solid transparent;border-radius:4px;color:var(--vscode-button-foreground);background:var(--vscode-button-background);cursor:pointer}button:hover{background:var(--vscode-button-hoverBackground)}button:focus-visible{outline:2px solid var(--vscode-focusBorder);outline-offset:2px}.secondary,.ghost{color:var(--vscode-button-secondaryForeground);background:var(--vscode-button-secondaryBackground)}.secondary:hover,.ghost:hover{background:var(--vscode-button-secondaryHoverBackground)}.ghost{background:transparent}.danger{color:var(--vscode-errorForeground);background:transparent;border-color:var(--vscode-errorForeground)}@media(max-width:620px){main{padding:28px 18px}header{align-items:flex-start}.grid{grid-template-columns:1fr}.device{align-items:flex-start}}@media(prefers-reduced-motion:reduce){*{transition:none!important}}`;

function randomNonce(): string {
  const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
  return Array.from({ length: 24 }, () => alphabet[Math.floor(Math.random() * alphabet.length)]).join("");
}
