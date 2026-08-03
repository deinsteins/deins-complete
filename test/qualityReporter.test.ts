import assert from "node:assert/strict";
import test from "node:test";
import { BackendClient } from "../src/api/deinsCompleteClient";
import { QualityEvent } from "../src/api/apiTypes";
import { CompletionRequest } from "../src/completion/completionTypes";
import { QualityReporter } from "../src/feedback/qualityReporter";

const request: CompletionRequest = {
  language:"typescriptreact",filePath:"/private/ProductCard.tsx",safeFilePath:"src/ProductCard.tsx",prefix:"<Card ",suffix:"",cursorOffset:6,documentVersion:1,currentLine:"<Card ",textBeforeCursorOnLine:"<Card ",textAfterCursorOnLine:"",indentation:"",mode:"full",
  repositoryContext:{files:[],dependencies:["@mui/material"],focus:"component-props",fingerprint:"not-sent",durationMs:1,timedOut:false},
  metadata:{totalDocumentCharacters:6,prefixCharacters:6,suffixCharacters:0,truncatedPrefix:false,truncatedSuffix:false,estimatedPrefixTokens:2,estimatedSuffixTokens:0,estimatedTotalTokens:2,buildDurationMilliseconds:1},
};

test("quality reporter is opt-in and sends only bounded metadata", async () => {
  const events: QualityEvent[] = [];
  const client: BackendClient = { complete: async()=>({completion:{text:""}}), health:async()=>({healthy:true,latencyMs:1}), sendQualityEvent:async(event)=>{events.push(event);} };
  const disabled = new QualityReporter({qualityInsightsEnabled:()=>false},client);
  assert.equal(disabled.shown(request,{text:"value"}),undefined);
  const reporter = new QualityReporter({qualityInsightsEnabled:()=>true},client);
  const context = reporter.shown(request,{text:"private source code",requestId:"safe-id",source:"backend",latencyMs:123});
  reporter.accepted(context);
  await new Promise(resolve=>setImmediate(resolve));
  assert.equal(events.length,2);
  assert.equal(events[0].framework,"mui");
  assert.equal(events[0].language,"typescriptreact");
  assert.equal(events[0].type,"shown");
  assert.equal(events[1].type,"accepted");
  assert.equal(events[0].completionId,events[1].completionId);
  assert.equal(JSON.stringify(events).includes("private source code"),false);
  assert.equal(JSON.stringify(events).includes("ProductCard"),false);
  assert.equal(JSON.stringify(events).includes("not-sent"),false);
});
