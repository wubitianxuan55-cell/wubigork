import json, os, re, collections

ROOT = r'C:\AI\wubigrok'
OUT = os.path.join(ROOT, 'docs', 'code-audit-2026-10-02', 'machine')

raw = open(os.path.join(ROOT, '.tmp', 'binding-names.txt'), encoding='utf-8-sig').read().split('\n')
names = sorted({l.strip() for l in raw if re.fullmatch(r'[A-Za-z_][A-Za-z0-9_]*', l.strip())})

ts_raw = open(os.path.join(ROOT, 'frontend/src/gaea/lib/bindingNames.ts'), encoding='utf-8-sig').read()
ts_names = sorted(set(re.findall(r'^\s*"([^"]+)",?\s*$', ts_raw, re.M)))

# 机械登记文件（这些文件必然出现全部方法名，不算消费者）
MECH = {
    'frontend/src/gaea/lib/bindingNames.ts',
    'frontend/src/gaea/lib/bridge.ts',
    'frontend/src/gaea/lib/bridge/appBindings.ts',
    'frontend/src/gaea/lib/bridge/mappings.ts',
    'frontend/src/gaea/lib/bridge/proxy.ts',
    'frontend/src/gaea/lib/spaceBindings.ts',
    'frontend/src/types/wails.d.ts',
}
mech_prefix = ('frontend/src/gaea/lib/bridge/', 'frontend/src/gaea/lib/mock/',
               'frontend/src/wailsjs', 'frontend/src/wailsjsCompat', 'frontend/src/types/wails')


def is_test(rel):
    return rel.endswith('.test.ts') or rel.endswith('.test.tsx') or '/test/' in rel


files = []
for dp, dn, fn in os.walk(os.path.join(ROOT, 'frontend', 'src')):
    dn[:] = [d for d in dn if d not in ('node_modules',)]
    for f in fn:
        if f.endswith(('.ts', '.tsx')):
            p = os.path.join(dp, f)
            rel = os.path.relpath(p, ROOT).replace('\\', '/')
            if rel in MECH or rel.startswith(mech_prefix):
                continue
            files.append((rel, open(p, encoding='utf-8-sig', errors='replace').read()))

prod = collections.defaultdict(list)
test = collections.defaultdict(list)
for rel, text in files:
    for n in names:
        if re.search(r'\b' + re.escape(n) + r'\b', text):
            (test if is_test(rel) else prod)[n].append(rel)

zero = [n for n in names if not prod[n] and not test[n]]
test_only = [n for n in names if not prod[n] and bool(test[n])]
test_only_sorted = sorted(((len(test[n]), n, test[n][:2]) for n in test_only))
only_one = sorted(((len(prod[n]), n, prod[n][0]) for n in names if len(prod[n]) == 1))

result = {
    'go_names': len(names),
    'ts_names': len(ts_names),
    'drift_go_only': sorted(set(names) - set(ts_names)),
    'drift_ts_only': sorted(set(ts_names) - set(names)),
    'zero_frontend_consumer_count': len(zero),
    'zero_frontend_consumer': zero,
    'test_only_consumer_count': len(test_only),
    'test_only_consumer': [{'name': n, 'files': f} for _, n, f in test_only_sorted],
    'single_prod_consumer_count': len(only_one),
    'single_prod_consumer': [{'name': n, 'file': f} for _, n, f in sorted(only_one)[:80]],
    'top_fanout': sorted(((len(v), k) for k, v in prod.items()), reverse=True)[:20],
}

# 提示词资产引用检查
pdir = os.path.join(ROOT, 'prompts')
prompts = sorted(f[:-5] for f in os.listdir(pdir) if f.endswith('.json'))
go_ts = []
for root in (os.path.join(ROOT, 'internal'), os.path.join(ROOT, 'frontend', 'src'), os.path.join(ROOT, 'scripts')):
    for dp, dn, fn in os.walk(root):
        dn[:] = [d for d in dn if d not in ('node_modules', 'testdata')]
        for f in fn:
            if f.endswith(('.go', '.ts', '.tsx', '.mjs')):
                p = os.path.join(dp, f)
                go_ts.append((os.path.relpath(p, ROOT).replace('\\', '/'), open(p, encoding='utf-8-sig', errors='replace').read()))
unref = []
for pr in prompts:
    hits = [rel for rel, text in go_ts if pr in text]
    if not hits:
        unref.append(pr)
result['prompt_assets'] = prompts
result['prompt_unreferenced'] = unref
result['prompt_reference_sample'] = {pr: [rel for rel, text in go_ts if pr in text][:4] for pr in prompts[:6]}

json.dump(result, open(os.path.join(OUT, 'bindings.json'), 'w', encoding='utf-8'), ensure_ascii=False, indent=1)
print('go names', len(names), 'ts names', len(ts_names), 'drift', len(result['drift_go_only']), len(result['drift_ts_only']))
print('zero consumers (no prod, no test):', len(zero))
print('test-only consumers:', len(test_only))
print('single prod consumer:', len(only_one))
for c, n, f in sorted(only_one)[:20]:
    print('  ', n, '->', f)
print('prompts unreferenced:', unref)
