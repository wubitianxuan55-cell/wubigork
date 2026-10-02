import json, os, collections, glob

ROOT = r'C:\AI\wubigrok'
UD = os.path.join(ROOT, 'docs', 'code-audit-2026-10-02', 'units')
OUT = os.path.join(ROOT, 'docs', 'code-audit-2026-10-02', 'machine')

units = []
broken = []
for p in sorted(glob.glob(os.path.join(UD, '*.json'))):
    try:
        d = json.load(open(p, encoding='utf-8-sig'))
    except Exception as e:
        broken.append([os.path.basename(p), str(e)[:120]])
        continue
    d['_file'] = os.path.basename(p)
    units.append(d)

sev = collections.Counter()
cat = collections.Counter()
allf = []
for u in units:
    for f in u.get('findings', []):
        sev[f.get('severity', '?')] += 1
        cat[f.get('category', '?')] += 1
        f['_unit'] = u.get('id')
        f['_module'] = u.get('module')
        allf.append(f)

order = {'P0': 0, 'P1': 1, 'P2': 2, 'P3': 3}
allf.sort(key=lambda f: (order.get(f.get('severity'), 9), f.get('_unit', '')))

summary = {
    'units_total': len(units),
    'units_broken': broken,
    'findings_total': len(allf),
    'severity': dict(sev),
    'category': dict(cat),
    'scores': sorted([[u.get('id'), u.get('module'), u.get('score'), u.get('verdict', '')[:400],
                       len(u.get('findings', []))] for u in units], key=lambda x: (x[2] if isinstance(x[2], (int, float)) else 99)),
}
json.dump(summary, open(os.path.join(OUT, 'summary.json'), 'w', encoding='utf-8'), ensure_ascii=False, indent=1)
json.dump(allf, open(os.path.join(OUT, 'findings_all.json'), 'w', encoding='utf-8'), ensure_ascii=False, indent=1)

print('units', len(units), 'broken', broken)
print('findings', len(allf), dict(sev))
print('category', dict(cat.most_common()))
print()
print('=== 评分（低到高） ===')
for sid, mod, sc, verdict, n in summary['scores']:
    print('%-4s %-52s score=%-5s findings=%s' % (sid, (mod or '')[:50], sc, n))
print()
print('=== P0 清单 ===')
for f in allf:
    if f.get('severity') != 'P0':
        continue
    print('[%s] %s | %s:%s | %s' % (f['_unit'], f.get('title'), f.get('file'), f.get('line'), (f.get('impact') or '')[:90]))
