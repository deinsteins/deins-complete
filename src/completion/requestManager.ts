import { createHash } from "node:crypto";
import { CompletionEngine } from "./completionEngine";
import { CompletionRequest, CompletionResult } from "./completionTypes";

export interface RequestSettings { debounceMs(): number; cacheEnabled(): boolean; cacheTtlMs(): number; cacheMaxEntries(): number; }
export interface CompletionStats { requested:number; backendRequests:number; cacheHits:number; cancelled:number; succeeded:number; totalLatencyMs:number; }
type CacheEntry={value:CompletionResult; expiresAt:number};

export class RequestManager {
  private readonly active=new Map<string,{id:number; controller:AbortController}>(); private readonly cache=new Map<string,CacheEntry>(); private sequence=0;
  private readonly stats:CompletionStats={requested:0,backendRequests:0,cacheHits:0,cancelled:0,succeeded:0,totalLatencyMs:0};
  constructor(private readonly engine:CompletionEngine,private readonly settings:RequestSettings){}
  async complete(documentKey:string,request:CompletionRequest,source:AbortSignal):Promise<CompletionResult|null>{
    const started=performance.now();this.stats.requested++;const key=this.key(request);const cached=this.get(key);if(cached){this.stats.cacheHits++;return cached}
    const id=++this.sequence;this.active.get(documentKey)?.controller.abort();const controller=new AbortController();const abort=()=>controller.abort();source.addEventListener("abort",abort,{once:true});this.active.set(documentKey,{id,controller});
    try{await wait(this.settings.debounceMs(),controller.signal);if(controller.signal.aborted)return null;this.stats.backendRequests++;const result=await this.engine.complete(request,controller.signal);if(controller.signal.aborted||this.active.get(documentKey)?.id!==id)return null;if(result&&this.settings.cacheEnabled())this.set(key,result);if(result)this.stats.succeeded++;return result
    }catch{return null}finally{source.removeEventListener("abort",abort);if(controller.signal.aborted)this.stats.cancelled++;if(this.active.get(documentKey)?.id===id)this.active.delete(documentKey);this.stats.totalLatencyMs+=performance.now()-started}
  }
  clearCache(){this.cache.clear()} cancel(documentKey:string){this.active.get(documentKey)?.controller.abort()} dispose(){for(const a of this.active.values())a.controller.abort();this.active.clear();this.cache.clear()}
  getStats():CompletionStats{return {...this.stats}}
  private key(r:CompletionRequest){return createHash("sha256").update(r.language).update("\0").update(r.prefix.slice(-2000)).update("\0").update(r.suffix.slice(0,1000)).digest("hex")}
  private get(key:string){const e=this.cache.get(key);if(!e||e.expiresAt<Date.now()){this.cache.delete(key);return null}this.cache.delete(key);this.cache.set(key,e);return e.value}
  private set(key:string,value:CompletionResult){this.cache.delete(key);this.cache.set(key,{value,expiresAt:Date.now()+this.settings.cacheTtlMs()});while(this.cache.size>this.settings.cacheMaxEntries())this.cache.delete(this.cache.keys().next().value!)}
}
function wait(ms:number,s:AbortSignal){return new Promise<void>((resolve)=>{const t=setTimeout(resolve,ms);s.addEventListener("abort",()=>{clearTimeout(t);resolve()},{once:true})})}
