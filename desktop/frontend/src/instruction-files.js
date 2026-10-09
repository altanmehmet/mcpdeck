// Group physical files rather than agent brands. Custom Copilot homes remain separate.
export function groupInstructionFiles(documents = [], metadata = []) {
 documents=documents || [];metadata=metadata || [];
 const status=new Map(metadata.map(file=>[file.path,file]));
 const groups=new Map();
 for(const doc of documents){
  if(!doc.Path) continue;
  let group=groups.get(doc.Path);
  if(!group){const file=status.get(doc.Path);group={path:doc.Path,agents:[],documents:[],status:file?.status || (doc.Note?.includes("(not created)")?"not created":"saved"),size:file?.size};groups.set(doc.Path,group);}
  if(!group.agents.includes(doc.Agent))group.agents.push(doc.Agent);
  group.documents.push(doc);
 }
 return [...groups.values()].map(group=>({...group,agents:group.agents.sort()}));
}
export function preferredInstructionFile(groups, agent) {
 const candidates=groups.filter(group=>group.agents.includes(agent));
 return candidates.find(group=>group.status==="saved"&&group.size!==0) || candidates.find(group=>group.status==="empty") || candidates[0];
}
