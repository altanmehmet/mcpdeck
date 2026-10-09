import { test } from 'node:test';
import assert from 'node:assert/strict';
import { lineDiff } from '../src/diff.js';
test('only inserted and deleted lines are highlighted',()=>{
 const result=lineDiff('first\nremoved\nlast','first\nadded\nlast');
 assert.deepEqual(result.current.map(x=>x.changed),[false,true,false]);
 assert.deepEqual(result.proposed.map(x=>x.changed),[false,true,false]);
});
test('duplicate lines use distinct matches',()=>{
 const result=lineDiff('same\nsame','same\nsame\nsame');
 assert.deepEqual(result.proposed.map(x=>x.changed),[false,false,true]);
});
test('large repeated-line native documents do not rescan matches',()=>{
 const source=Array(30000).fill('same').join('\n');
 const result=lineDiff(source,source+'\nnew');
 assert.equal(result.current.filter(x=>x.changed).length,0);
 assert.equal(result.proposed.filter(x=>x.changed).length,1);
});
