import os, re, json

ROOT = r'C:\AI\wubigrok'
src = os.path.join(ROOT, 'frontend', 'src')
txt = {}
for dp, dn, fn in os.walk(src):
    dn[:] = [d for d in dn if d != 'node_modules']
    for f in fn:
        if f.endswith(('.ts', '.tsx')):
            p = os.path.join(dp, f)
            rel = os.path.relpath(p, ROOT).replace('\\', '/')
            txt[rel] = open(p, encoding='utf-8-sig', errors='replace').read()

api = sorted([r for r in txt if r.startswith('frontend/src/api/')])
rows = []
for a in api:
    base = os.path.basename(a)[:-3]
    pat = re.compile(r'''['"][^'"]*''' + re.escape(base) + r'''['"]''')
    importers = sorted({r for r in txt if r != a and pat.search(txt[r]) and not r.endswith(('.test.ts', '.test.tsx'))})
    rows.append({'file': a.replace('frontend/src/', ''), 'importers': importers})

out = {'api_files': len(api), 'rows': rows}
json.dump(out, open(os.path.join(ROOT, 'docs', 'code-audit-2026-10-02', 'machine', 'apilayer.json'), 'w', encoding='utf-8'), ensure_ascii=False, indent=1)
for r in rows:
    print('%-40s importers=%d %s' % (r['file'], len(r['importers']), r['importers'][:3]))
