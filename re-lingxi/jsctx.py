import re,sys
p=sys.argv[1]; pat=sys.argv[2]; n=int(sys.argv[3]) if len(sys.argv)>3 else 300
d=open(p,encoding='utf-8',errors='ignore').read()
for i,m in enumerate(re.finditer(pat,d)):
    if i>=4: break
    print("=== hit",i,"at",m.start())
    print(d[max(0,m.start()-n):m.start()+n])
    print()
