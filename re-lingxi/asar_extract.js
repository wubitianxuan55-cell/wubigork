const fs=require('fs'),path=require('path');
const asar=process.argv[2], prefix=process.argv[3], destRoot=process.argv[4];
const json=JSON.parse(fs.readFileSync('out/asar-header.json','utf8'));
const base=8+Number(fs.readFileSync("out/base.txt","utf8").trim());
const fd=fs.openSync(asar,'r');
let n=0;
(function walk(node,p){
  for(const [name,ent] of Object.entries(node.files||{})){
    const cur=p?p+'/'+name:name;
    if(ent.files){ if((prefix&&!prefix.startsWith(cur)&&!cur.startsWith(prefix))===false) walk(ent,cur); else if(!prefix) walk(ent,cur); else if(cur.startsWith(prefix)||prefix.startsWith(cur)) walk(ent,cur); }
    else {
      if(prefix && !cur.startsWith(prefix)) continue;
      const dst=path.join(destRoot,cur);
      fs.mkdirSync(path.dirname(dst),{recursive:true});
      const out=Buffer.alloc(ent.size);
      fs.readSync(fd,out,0,ent.size,base+Number(ent.offset));
      fs.writeFileSync(dst,out); n++;
    }
  }
})(json,'');
console.log('extracted',n,'files to',destRoot);
