import re
p="extract_all.py"
s=open(p,encoding="utf-8").read()
s=s.replace('''route_re = re.compile(r"/(?:api|office|v[0-9]|kdocs|kdg|internal|open|auth)[a-zA-Z0-9_/{}\\.\\-]{3,90}")''',
'''route_re = re.compile(r"/(?:api|office|v[0-9]|kdocs|kdg|open)[a-zA-Z0-9_/{}\\.\\-]{3,90}")
ROUTE_NOISE = re.compile(r"(protoc-gen|etcdserverpb|genproto|googleapis|grpc|reflect|runtime|"
                         r"sync|crypto/|encoding/|net/|os/|path/|unicode|unsafe|vendor|pkg/mod|"
                         r"internal/|\\.go\\b|[A-Z])")''')
s=s.replace('''routes = Counter(m.group(0).rstrip(".,;)'\\"") for m in route_re.finditer(lx))''',
'''routes = Counter(r for r in (m.group(0).rstrip(".,;)'\\"") for m in route_re.finditer(lx))
                 if not ROUTE_NOISE.search(r) and r.count("/") <= 9)''')
s=s.replace('''for p, n in rows[:40]:''','''STD = {"runtime","internal","sync","reflect","syscall","unsafe","unicode","net","os","fmt",
       "strconv","strings","time","context","io","bytes","math","sort","path","crypto","encoding",
       "errors","flag","hash","html","index","log","mime","regexp","text","vendor","bufio","compress"}
for p, n in [r for r in rows if r[0] not in STD][:40]:''')
open(p,"w",encoding="utf-8").write(s)
print("patched")
