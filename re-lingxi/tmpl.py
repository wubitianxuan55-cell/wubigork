import re,sys
p=r"C:\Users\wubi\AppData\Roaming\WPS 灵犀\app_server_5.24.0\service\lingxi.exe"
d=open(p,'rb').read().decode('utf-8','ignore')
pat=sys.argv[1]; n=int(sys.argv[2]) if len(sys.argv)>2 else 3000
m=re.search(pat,d)
if not m: print("NO MATCH"); sys.exit()
print(d[max(0,m.start()-200):m.start()+n])
