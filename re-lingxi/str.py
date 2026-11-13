import re,sys
p=sys.argv[1]; pat=sys.argv[2] if len(sys.argv)>2 else None
d=open(p,'rb').read()
s=re.findall(rb'[\x20-\x7e]{6,200}',d)
seen=set(); out=[]
for b in s:
    t=b.decode('ascii','ignore')
    if pat and not re.search(pat,t,re.I): continue
    if t in seen: continue
    seen.add(t); out.append(t)
print('\n'.join(out[:int(sys.argv[3]) if len(sys.argv)>3 else 200]))
