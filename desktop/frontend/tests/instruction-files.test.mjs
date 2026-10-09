import test from 'node:test';
import assert from 'node:assert/strict';
import {groupInstructionFiles,preferredInstructionFile} from '../src/instruction-files.js';
test('Gemini and Antigravity share a physical file without duplicate choices',()=>{
 const docs=[{Agent:'gemini-cli',Path:'/home/.gemini/GEMINI.md'},{Agent:'antigravity',Path:'/home/.gemini/GEMINI.md'}];
 const groups=groupInstructionFiles(docs,[{path:docs[0].Path,status:'saved',size:100}]);
 assert.equal(groups.length,1);assert.deepEqual(groups[0].agents,['antigravity','gemini-cli']);assert.equal(preferredInstructionFile(groups,'gemini-cli'),groups[0]);
});
test('Copilot variants group only when their paths match',()=>{
 const groups=groupInstructionFiles([{Agent:'copilot',Path:'/home/.copilot/copilot-instructions.md'},{Agent:'copilot-cli',Path:'/home/.copilot/copilot-instructions.md'},{Agent:'copilot-cli',Path:'/custom/copilot-instructions.md'}]);
 assert.equal(groups.length,2);assert.equal(groups[0].agents.length,2);assert.equal(groups[1].agents.length,1);
});
test('existing modular guidance is preferred over an uncreated main file',()=>{
 const groups=groupInstructionFiles([{Agent:'copilot',Path:'/main',Note:'(not created)'},{Agent:'copilot',Path:'/modular'}],[{path:'/main',status:'not created',size:0},{path:'/modular',status:'saved',size:50}]);
 assert.equal(preferredInstructionFile(groups,'copilot').path,'/modular');assert.equal(preferredInstructionFile(groups,'manual-agent'),undefined);
 assert.deepEqual(groupInstructionFiles(null,null),[]);
});
