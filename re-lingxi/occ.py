import re,sys
p=r"C:\Users\wubi\AppData\Roaming\WPS 灵犀\app_server_5.24.0\service\lingxi.exe"
d=open(p,'rb').read().decode('utf-8','ignore')
pat=sys.argv[1]; lim=int(sys.argv[2]) if len(sys.argv)>2 else 6
pre=int(sys.argv[3]) if len(sys.argv)>3 else 300; post=int(sys.argv[4]) if len(sys.argv)>4 else 900
cnt=0
for m in re.finditer(pat,d):
    cnt+=1
    if cnt>lim: break
    print("=== hit",cnt,"at",m.start())
    print(d[max(0,m.start()-pre):m.start()+post])
