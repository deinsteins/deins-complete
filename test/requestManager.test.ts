import assert from "node:assert/strict";
import test from "node:test";
import { completionDebounceMs, RequestManager } from "../src/completion/requestManager";
import { CompletionRequest } from "../src/completion/completionTypes";
const req=(prefix="x"):CompletionRequest=>({language:"ts",filePath:"x",safeFilePath:"x",prefix,suffix:"",cursorOffset:prefix.length,documentVersion:1,currentLine:prefix,textBeforeCursorOnLine:prefix,textAfterCursorOnLine:"",indentation:"",metadata:{totalDocumentCharacters:0,prefixCharacters:0,suffixCharacters:0,truncatedPrefix:false,truncatedSuffix:false,estimatedPrefixTokens:0,estimatedSuffixTokens:0,estimatedTotalTokens:0,buildDurationMilliseconds:0}});
const settings={debounceMs:()=>0,cacheEnabled:()=>true,cacheTtlMs:()=>60000,cacheMaxEntries:()=>2};
test("cache and in-flight deduplication avoid duplicate engine calls",async()=>{let calls=0;const manager=new RequestManager({complete:async()=>{calls++;return{text:"ok"}}},settings);const signal=new AbortController().signal;await Promise.all([manager.complete("a",req(),signal),manager.complete("a",req(),signal)]);await manager.complete("a",req(),signal);assert.equal(calls,1);assert.equal(manager.getStats().cacheHits,1)});
test("different documents can complete independently",async()=>{let calls=0;const manager=new RequestManager({complete:async()=>{calls++;return{text:"ok"}}},settings);const s=new AbortController().signal;await Promise.all([manager.complete("a",req("a"),s),manager.complete("b",req("b"),s)]);assert.equal(calls,2)});
test("repository fingerprint prevents stale completion cache reuse",async()=>{let calls=0;const manager=new RequestManager({complete:async()=>{calls++;return{text:"ok"}}},settings);const s=new AbortController().signal;const first={...req(),repositoryContext:{files:[],fingerprint:"before",durationMs:1,timedOut:false}},second={...req(),repositoryContext:{files:[],fingerprint:"after",durationMs:1,timedOut:false}};await manager.complete("a",first,s);await manager.complete("a",second,s);assert.equal(calls,2)});
test("local style changes prevent stale completion cache reuse",async()=>{let calls=0;const manager=new RequestManager({complete:async()=>{calls++;return{text:"ok"}}},settings);const s=new AbortController().signal;await manager.complete("a",{...req(),style:{indentation:"spaces",indentSize:2}},s);await manager.complete("a",{...req(),style:{indentation:"tabs"}},s);assert.equal(calls,2)});
test("cache continuation removes exactly what the user typed",async()=>{let calls=0;const manager=new RequestManager({complete:async()=>{calls++;return{text:"getUser()"}}},settings);const s=new AbortController().signal;const initial={...req("const user = "),textBeforeCursorOnLine:"const user = "};await manager.complete("a",initial,s);const continued={...req("const user = g"),textBeforeCursorOnLine:"const user = g"};assert.deepEqual(await manager.complete("a",continued,s),{text:"etUser()",source:"cache",latencyMs:0});assert.equal(calls,1)});
test("cache continuation rejects diverging typed text",async()=>{let calls=0;const manager=new RequestManager({complete:async()=>{calls++;return{text:"getUser()"}}},settings);const s=new AbortController().signal;await manager.complete("a",{...req("const user = "),textBeforeCursorOnLine:"const user = "},s);await manager.complete("a",{...req("const user = set"),textBeforeCursorOnLine:"const user = set"},s);assert.equal(calls,2)});
test("short negative cache avoids repeated empty requests",async()=>{let calls=0;const manager=new RequestManager({complete:async()=>{calls++;return null}},settings);const s=new AbortController().signal;await manager.complete("a",req("nothing"),s);await manager.complete("a",req("nothing"),s);assert.equal(calls,1);assert.equal(manager.getStats().negativeCacheHits,1)});
test("full-context request can use the short pre-debounce cache",async()=>{let calls=0;const manager=new RequestManager({complete:async()=>{calls++;return{text:"ok"}}},settings);const s=new AbortController().signal;await manager.complete("a",req("same"),s);const full={...req("same"),repositoryContextTask:Promise.resolve(undefined)};assert.deepEqual(await manager.complete("a",full,s),{text:"ok",source:"cache",latencyMs:0});assert.equal(calls,1)});
test("request manager reports activity and stage latency",async()=>{const activity:string[]=[];const manager=new RequestManager({complete:async()=>({text:"ok"})},settings,(value)=>activity.push(value));const s=new AbortController().signal;await manager.complete("a",req(),s);await manager.complete("a",req(),s);const stats=manager.getStats();assert.deepEqual(activity,["thinking","ready","cached"]);assert.equal(stats.backendRequests,1);assert.ok(stats.totalDebounceMs>=0);assert.ok(stats.totalBackendMs>=0)});

test("React JSX contexts use a shorter bounded debounce",()=>{
  assert.equal(completionDebounceMs({...req("<ProductCard"),language:"typescriptreact"},150),80);
  assert.equal(completionDebounceMs({...req("<ProductCard "),language:"typescriptreact"},150),80);
  assert.equal(completionDebounceMs({...req('value="'),language:"typescriptreact"},150),60);
  assert.equal(completionDebounceMs(req("plain"),150),150);
});

test("active backend result is reused for exact typed continuation",async()=>{
  let calls=0;let started!:()=>void;let finish!:(value:{text:string})=>void;
  const began=new Promise<void>((resolve)=>{started=resolve;});
  const result=new Promise<{text:string}>((resolve)=>{finish=resolve;});
  const manager=new RequestManager({complete:async()=>{calls++;started();return result;}},settings);
  const firstController=new AbortController();
  const base={...req("const user = "),intent:"general" as const,mode:"fast" as const};
  const first=manager.complete("a",base,firstController.signal);
  await began;
  firstController.abort();
  const continued=manager.complete("a",{...req("const user = g"),intent:"general",mode:"fast"},new AbortController().signal);
  finish({text:"getUser()"});
  assert.deepEqual(await continued,{text:"etUser()"});
  await first;
  assert.equal(calls,1);
});

test("active continuation rejects nonmatching provider output without fuzzy reuse",async()=>{
  let started!:()=>void;let finish!:(value:{text:string})=>void;
  const began=new Promise<void>((resolve)=>{started=resolve;});
  const result=new Promise<{text:string}>((resolve)=>{finish=resolve;});
  const manager=new RequestManager({complete:async()=>{started();return result;}},settings);
  const firstController=new AbortController();
  const first=manager.complete("a",{...req("const user = "),intent:"general",mode:"fast"},firstController.signal);
  await began;
  firstController.abort();
  const diverged=manager.complete("a",{...req("const user = s"),intent:"general",mode:"fast"},new AbortController().signal);
  finish({text:"getUser()"});
  assert.equal(await diverged,null);
  await first;
});
