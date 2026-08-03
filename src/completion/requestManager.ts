import { createHash } from "node:crypto";
import { CompletionEngine } from "./completionEngine";
import { CompletionRequest, CompletionResult } from "./completionTypes";

export interface RequestSettings { debounceMs(): number; cacheEnabled(): boolean; cacheTtlMs(): number; cacheMaxEntries(): number; }
export interface CompletionStats { requested:number; backendRequests:number; cacheHits:number; negativeCacheHits:number; deduplicated:number; cancelled:number; succeeded:number; totalLatencyMs:number; totalDebounceMs:number; totalBackendMs:number; }
export type CompletionActivity = "thinking" | "cached" | "ready";
type Active = { id:number; key:string; controller:AbortController; promise:Promise<CompletionResult|null>; request:CompletionRequest; backendStarted:boolean; consumer:number; abortTimer?:ReturnType<typeof setTimeout> };
type Entry = { value:CompletionResult; expiresAt:number; documentKey:string; prefix:string; suffix:string; fingerprint:string };

export class RequestManager {
  private readonly active = new Map<string, Active>();
  private readonly cache = new Map<string, Entry>();
  private readonly quick = new Map<string, Entry>();
  private readonly negative = new Map<string, number>();
  private id = 0;
  private readonly stats: CompletionStats = { requested:0, backendRequests:0, cacheHits:0, negativeCacheHits:0, deduplicated:0, cancelled:0, succeeded:0, totalLatencyMs:0, totalDebounceMs:0, totalBackendMs:0 };

  constructor(private readonly engine:CompletionEngine, private readonly settings:RequestSettings, private readonly activity?:(activity:CompletionActivity)=>void, private readonly debounceAdjustment?:(request:CompletionRequest)=>number) {}

  complete(documentKey:string, request:CompletionRequest, source:AbortSignal):Promise<CompletionResult|null> {
    this.stats.requested++;
    const key = this.key(request);
    const cached = request.repositoryContextTask === undefined ? (this.get(key) ?? this.refine(documentKey, request)) : this.getQuick(this.baseKey(request));
    if (cached) { this.stats.cacheHits++; this.activity?.("cached"); return Promise.resolve(cached); }
    const negativeUntil = this.negative.get(key);
    if (request.repositoryContextTask === undefined && negativeUntil !== undefined && negativeUntil > Date.now()) { this.stats.negativeCacheHits++; return Promise.resolve(null); }

    const current = this.active.get(documentKey);
    if (current?.key === key) return this.reuse(current, request, source, false);
    if (current?.backendStarted && compatibleContinuation(current.request, request)) return this.reuse(current, request, source, true);
    if (current !== undefined) this.abortNow(current);

    const controller = new AbortController();
    const active: Active = { id:++this.id, key, controller, promise:Promise.resolve(null), request, backendStarted:false, consumer:0 };
    const cleanup = this.bindCancellation(active, source, active.consumer);
    active.promise = this.run(request, active, documentKey, key);
    this.active.set(documentKey, active);
    return active.promise.finally(cleanup);
  }

  clearCache():void { this.cache.clear(); this.quick.clear(); this.negative.clear(); }
  cancel(documentKey:string):void { const active = this.active.get(documentKey); if (active !== undefined) this.abortNow(active); }
  cancelAll():void { for (const active of this.active.values()) this.abortNow(active); }
  dispose():void { this.cancelAll(); this.active.clear(); this.clearCache(); }
  getStats():CompletionStats { return { ...this.stats }; }

  private reuse(active:Active, request:CompletionRequest, source:AbortSignal, continuation:boolean):Promise<CompletionResult|null> {
    this.stats.deduplicated++;
    active.consumer++;
    this.clearAbortTimer(active);
    const cleanup = this.bindCancellation(active, source, active.consumer);
    const result = continuation ? active.promise.then((value) => refineActiveCompletion(active.request, request, value)) : active.promise;
    return result.finally(cleanup);
  }

  private bindCancellation(active:Active, source:AbortSignal, consumer:number):()=>void {
    const abort = () => {
      if (active.consumer !== consumer) return;
      if (!active.backendStarted) { this.abortNow(active); return; }
      this.clearAbortTimer(active);
      active.abortTimer = setTimeout(() => this.abortNow(active), 60);
    };
    if (source.aborted) abort(); else source.addEventListener("abort", abort, { once:true });
    return () => source.removeEventListener("abort", abort);
  }

  private abortNow(active:Active):void { this.clearAbortTimer(active); active.controller.abort(); }
  private clearAbortTimer(active:Active):void { if (active.abortTimer !== undefined) { clearTimeout(active.abortTimer); active.abortTimer = undefined; } }

  private async run(request:CompletionRequest, active:Active, documentKey:string, key:string):Promise<CompletionResult|null> {
    const started = performance.now();
    try {
      const debounceStarted = performance.now();
      await delay(completionDebounceMs(request, this.settings.debounceMs(), this.debounceAdjustment?.(request) ?? 0), active.controller.signal);
      this.stats.totalDebounceMs += performance.now() - debounceStarted;
      if (active.controller.signal.aborted) return null;
      if (request.repositoryContextTask !== undefined) {
        const repository = await settleWithin(request.repositoryContextTask, 10, active.controller.signal);
        if (!active.controller.signal.aborted && repository !== undefined) request.repositoryContext = repository;
        key = this.key(request);
        const cached = this.get(key);
        if (cached) { this.stats.cacheHits++; this.activity?.("cached"); return cached; }
      }
      active.backendStarted = true;
      this.stats.backendRequests++;
      this.activity?.("thinking");
      const backendStarted = performance.now();
      const result = await this.engine.complete(request, active.controller.signal);
      this.stats.totalBackendMs += performance.now() - backendStarted;
      if (active.controller.signal.aborted || this.active.get(documentKey)?.id !== active.id) return null;
      if (result && this.settings.cacheEnabled()) this.set(key, result, documentKey, request);
      if (!result) this.negative.set(key, Date.now() + 2000);
      if (result) this.stats.succeeded++;
      return result;
    } catch { return null; }
    finally {
      if (active.controller.signal.aborted) this.stats.cancelled++;
      const isCurrent = this.active.get(documentKey)?.id === active.id;
      if (isCurrent) { this.clearAbortTimer(active); this.active.delete(documentKey); }
      this.stats.totalLatencyMs += performance.now() - started;
      if (isCurrent) this.activity?.("ready");
    }
  }

  private key(request:CompletionRequest):string { return createHash("sha256").update(this.baseKey(request)).update("\0").update(request.repositoryContext?.fingerprint ?? "").digest("hex"); }
  private baseKey(request:CompletionRequest):string { return createHash("sha256").update(request.language).update("\0").update(request.prefix.slice(-2000)).update("\0").update(request.suffix.slice(0,1000)).update("\0").update(JSON.stringify(request.style ?? null)).digest("hex"); }
  private get(key:string):CompletionResult|null { const entry=this.cache.get(key); if(!entry||entry.expiresAt<Date.now()){this.cache.delete(key);return null;} this.cache.delete(key);this.cache.set(key,entry);return{...entry.value,source:"cache",latencyMs:0}; }
  private getQuick(key:string):CompletionResult|null { const entry=this.quick.get(key);if(!entry||entry.expiresAt<Date.now()){this.quick.delete(key);return null;}return{...entry.value,source:"cache",latencyMs:0}; }
  private set(key:string,value:CompletionResult,documentKey:string,request:CompletionRequest):void { const entry={value,expiresAt:Date.now()+this.settings.cacheTtlMs(),documentKey,prefix:request.prefix,suffix:request.suffix,fingerprint:request.repositoryContext?.fingerprint??""};this.cache.delete(key);this.cache.set(key,entry);this.quick.set(this.baseKey(request),{...entry,expiresAt:Math.min(entry.expiresAt,Date.now()+2000)});while(this.cache.size>this.settings.cacheMaxEntries())this.cache.delete(this.cache.keys().next().value as string); }
  private refine(documentKey:string,request:CompletionRequest):CompletionResult|null { const fingerprint=request.repositoryContext?.fingerprint??"";for(const entry of [...this.cache.values()].reverse()){if(entry.expiresAt<Date.now()||entry.documentKey!==documentKey||entry.suffix!==request.suffix||entry.fingerprint!==fingerprint||!request.prefix.startsWith(entry.prefix))continue;const typed=request.prefix.slice(entry.prefix.length);if(typed!==""&&entry.value.text.startsWith(typed))return{text:entry.value.text.slice(typed.length),source:"cache",latencyMs:0};}return null; }
}

export function completionDebounceMs(request:CompletionRequest, configured:number, adjustment=0):number {
  const base = configured === 0 ? 0 : Math.max(40, Math.min(250, configured + adjustment));
  const line = request.textBeforeCursorOnLine;
  if (/react$/i.test(request.language) && (/[<>"'=:{]$/.test(line) || /<[A-Za-z][\w.:-]*(?:\s+[^<>]*)?$/.test(line))) return Math.min(base, /[<>"'=:{]$/.test(line) ? 60 : 80);
  return /[[.({=,:]$/.test(line) ? Math.min(base, 70) : base;
}

function compatibleContinuation(base:CompletionRequest, next:CompletionRequest):boolean {
  if (base.language!==next.language||base.safeFilePath!==next.safeFilePath||base.suffix!==next.suffix||base.intent!==next.intent||base.mode!==next.mode||!next.prefix.startsWith(base.prefix)) return false;
  const typed = next.prefix.slice(base.prefix.length);
  return typed.length > 0 && typed.length <= 64 && !typed.includes("\n") && next.cursorOffset === base.cursorOffset + typed.length;
}

function refineActiveCompletion(base:CompletionRequest, next:CompletionRequest, result:CompletionResult|null):CompletionResult|null {
  if (result === null) return null;
  const typed = next.prefix.slice(base.prefix.length);
  return result.text.startsWith(typed) ? { ...result, text:result.text.slice(typed.length) } : null;
}

function delay(milliseconds:number, signal:AbortSignal):Promise<void> { return new Promise((resolve)=>{const timer=setTimeout(resolve,milliseconds);signal.addEventListener("abort",()=>{clearTimeout(timer);resolve();},{once:true});}); }
async function settleWithin<T>(promise:Promise<T>,milliseconds:number,signal:AbortSignal):Promise<T|undefined> { return Promise.race([promise.catch(()=>undefined),delay(milliseconds,signal).then(()=>undefined)]); }
