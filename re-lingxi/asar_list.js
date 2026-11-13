const fs=require('fs');
const json=JSON.parse(fs.readFileSync('out/asar-header.json','utf8'));
const rows=[];
(function walk(node,prefix){
  for(const [name,ent] of Object.entries(node.files||{})){
    const p=prefix?prefix+'/'+name:name;
    if(ent.files) walk(ent, p);
    else rows.push({p,size:ent.size||0,off:ent.offset});
  }
})(json,'');
fs.writeFileSync('out/files.tsv',rows.map(r=>[r.p,r.size,r.off].join('\t')).join('\n'));
console.log('files',rows.length,'totalMB',(rows.reduce((a,b)=>a+b.size,0)/1048576).toFixed(1));
const dirAgg={};
for(const r of rows){const parts=r.p.split('/');const k=parts.slice(0,2).join('/');dirAgg[k]=(dirAgg[k]||0)+r.size;}
Object.entries(dirAgg).sort((a,b)=>b[1]-a[1]).slice(0,40).forEach(([k,v])=>console.log((v/1048576).toFixed(1)+'MB',k));
