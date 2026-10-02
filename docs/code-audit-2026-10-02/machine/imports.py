import os, re, json, collections

ROOT = r'C:\AI\wubigrok'
OUT = os.path.join(ROOT, 'docs', 'code-audit-2026-10-02', 'machine')

imp_re = re.compile(r'^\s*(?:[\w.]+\s+)?"(github\.com/gaea/gaea/[^"]+)"', re.M)


def pkg_of(rel):
    return os.path.dirname(rel).replace('\\', '/') or '.'


edges = collections.defaultdict(set)   # package -> set(imported package)
files = collections.defaultdict(list)
for dp, dn, fn in os.walk(os.path.join(ROOT, 'internal')):
    dn[:] = [d for d in dn if d not in ('testdata',)]
    for f in fn:
        if not f.endswith('.go') or f.endswith('_test.go'):
            continue
        p = os.path.join(dp, f)
        rel = os.path.relpath(p, ROOT).replace('\\', '/')
        pkg = pkg_of(rel)
        files[pkg].append(rel)
        text = open(p, encoding='utf-8-sig', errors='replace').read()
        for m in imp_re.finditer(text):
            tgt = m.group(1).replace('github.com/gaea/gaea/', '')
            edges[pkg].add(tgt)

fanin = collections.Counter()
for pkg, deps in edges.items():
    for d in deps:
        fanin[d] += 1

# 层级倒挂候选：域包/gaea 子包 反向 import internal/app
inversions = []
for pkg, deps in edges.items():
    if pkg.startswith('internal/app'):
        continue
    for d in deps:
        if d == 'internal/app':
            inversions.append([pkg, d])

# 同名双包：internal/X 与 internal/gaea/X 或 internal/app 与 internal/gaea/app 之类
names = collections.defaultdict(list)
for pkg in files:
    names[os.path.basename(pkg)].append(pkg)
twin = {k: sorted(v) for k, v in names.items() if len(v) >= 2 and not k.startswith('builtin')}

# app 的热依赖（域包是否都挂 app）
app_deps = sorted([(len(files[p]), p) for p in edges if p.startswith('internal/app')], reverse=True)

result = {
    'packages': len(files),
    'fanin_top30': fanin.most_common(30),
    'imports_into_internal_app': inversions,
    'twin_package_names': twin,
    'top_importers': sorted(((len(v), k) for k, v in edges.items()), reverse=True)[:20],
    'gaea_subpkg_count': len([p for p in files if p.startswith('internal/gaea/')]),
}
json.dump(result, open(os.path.join(OUT, 'imports.json'), 'w', encoding='utf-8'), ensure_ascii=False, indent=1)
print('packages', len(files))
print('fan-in top15:')
for p, c in fanin.most_common(15):
    print('  %3d  %s' % (c, p))
print('imports into internal/app from elsewhere:', len(inversions))
for a, b in inversions[:20]:
    print('  ', a, '->', b)
print('twin package basenames:', len(twin))
for k, v in list(sorted(twin.items()))[:20]:
    print('  ', k, v)
print('top importers:')
for c, p in sorted(((len(v), k) for k, v in edges.items()), reverse=True)[:10]:
    print('  %3d  %s' % (c, p))
