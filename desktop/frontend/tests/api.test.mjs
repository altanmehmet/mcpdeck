import test from "node:test";
import assert from "node:assert/strict";
import { call, friendly, isNative } from "../src/api.js";

test("native actions use the desktop binding and preserve arguments", async () => {
  const edit={text:"local draft",expected:"saved"};
  globalThis.window={go:{client:{App:{ReviewInstructions:async (value)=>{assert.equal(value,edit);return {token:"review"};}}}}};
  assert.equal(isNative(),true);
  assert.deepEqual(await call("ReviewInstructions",edit),{token:"review"});
});
test("missing native action fails without falling back to HTTP", async () => {
  globalThis.window={go:{client:{App:{}}}};
  const before=globalThis.fetch;
  globalThis.fetch=()=>{throw new Error("HTTP fallback must not run")};
  try{await assert.rejects(call("Missing"),/does not support this action/);}finally{globalThis.fetch=before;}
});
test("preview surfaces backend errors without exposing its internal response", async () => {
  globalThis.window={};
  const before=globalThis.fetch;
  globalThis.fetch=async()=>({ok:true,json:async()=>({error:"Review expired"})});
  try{await assert.rejects(call("ApplyReview","old"),/Review expired/);}finally{globalThis.fetch=before;}
});
test("agent labels use own properties only", () => {
  assert.equal(friendly("codex"),"Codex");
  assert.equal(friendly("constructor"),"constructor");
});
