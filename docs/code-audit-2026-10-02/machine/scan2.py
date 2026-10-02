import json, os, re, hashlib, collections, itertools

ROOT = r'C:\AI\wubigrok'
OUT = os.path.join(ROOT, 'docs', 'code-audit-2026-10-02', 'machine')
SKIP_DIRS = {'node_modules', 'clones', 'backups', 'dist', 'build', 'releases', 'whisper_data',
             '.git', '.tmp', '.tmp-pdf', '.gaea', 'novels', 'design-system', '.agent-teams',
             '.dsh-vision-router', 'wailsjs'}


def rel(p):
    return os.path.relpath(p, ROOT).replace('\\', '/')


def walk(exts, roots, include_tests=False):
    out = []
    for r in roots:
        for dp, dn, fn in os.walk(r):
            dn[:] = [d for d in dn if d not in SKIP_DIRS]
            for f in fn:
                if not f.endswith(exts):
                    continue
                if not include_tests and (f.endswith('_test.go') or f.endswith('.test.ts') or f.endswith('.test.tsx')):
                    continue
                out.append(os.path.join(dp, f))
    return sorted(out)


def read(p):
    try:
        with open(p, 'r', encoding='utf-8', errors='replace') as f:
            return f.read().split('\n')
    except Exception:
        return []


roots = [os.path.join(ROOT, 'internal'), os.path.join(ROOT, 'frontend', 'src')]
go = walk('.go', roots)
ts = walk(('.ts', '.tsx'), roots)


def norm_lines(p):
    out = []
    for i, ln in enumerate(read(p), 1):
        s = ln.strip()
        if len(s) < 6:
            continue
        if s.startswith('//') or s.startswith('*') or s.startswith('/*') or s.startswith('#'):
            continue
        if s in ('{', '}', '});', ');', '})', '};', 'import', 'return', '} else {', '})', '},'):
            continue
        out.append((re.sub(r'\s+', ' ', s), i))
    return out


# ---- 1. long identical lines across files (copy-paste of fat lines) ----
longline = collections.defaultdict(list)
for p in go + ts:
    for s, i in norm_lines(p):
        if len(s) >= 80:
            longline[hashlib.md5(s.encode('utf-8')).hexdigest()].append((rel(p), i, s[:110]))
copies = [{'files': sorted({c[0] for c in v}), 'n': len(v), 'sample': v[0][2], 'locs': [[c[0], c[1]] for c in v[:6]]}
          for k, v in longline.items() if len({c[0] for c in v}) >= 2]
copies.sort(key=lambda x: (-len(x['files']), -x['n']))

# ---- 2. block duplicates: window 10 over normalized lines ----
def blocks(paths, window=10):
    table = collections.defaultdict(list)
    for p in paths:
        nl = norm_lines(p)
        for i in range(len(nl) - window + 1):
            text = '\n'.join(x[0] for x in nl[i:i + window])
            table[hashlib.md5(text.encode('utf-8')).hexdigest()].append((rel(p), nl[i][1]))
    groups = []
    for k, v in table.items():
        files = sorted({x[0] for x in v})
        if len(files) >= 2:
            groups.append({'files': files, 'n': len(v), 'locs': [[x[0], x[1]] for x in v[:6]]})
    groups.sort(key=lambda g: (-len(g['files']), -g['n']))
    return groups


bgo = blocks(go)
bts = blocks(ts)


# ---- 3. same-area file similarity (parallel implementation candidates) ----
def shingles(path):
    return {hashlib.md5(s.encode('utf-8')).hexdigest() for s, _ in norm_lines(path)}


groups = collections.defaultdict(list)
for p in go + ts:
    parts = rel(p).split('/')
    area = '/'.join(parts[:2]) if parts[0] == 'internal' else '/'.join(parts[:3])
    if len(read(p)) >= 150:
        groups[area].append(p)

similar = []
for area, files in groups.items():
    data = {p: shingles(p) for p in files}
    for a, b in itertools.combinations(files, 2):
        A, B = data[a], data[b]
        if not A or not B:
            continue
        j = len(A & B) / len(A | B)
        if j >= 0.30:
            similar.append({'jaccard': round(j, 3), 'a': rel(a), 'b': rel(b),
                            'a_loc': len(read(a)), 'b_loc': len(read(b)),
                            'shared': len(A & B)})
similar.sort(key=lambda x: -x['jaccard'])

# ---- 4. same-basename modules in different dirs ----
byname = collections.defaultdict(list)
for p in go + ts:
    byname[os.path.splitext(os.path.basename(p))[0]].append(rel(p))
dupe_names = {k: v for k, v in byname.items() if len(v) >= 2}

# ---- 5. suspicious single-file "god packages" ----
pkg_loc = collections.defaultdict(lambda: [0, 0])
for p in go:
    parts = rel(p).split('/')
    key = '/'.join(parts[:3]) if len(parts) > 2 else '/'.join(parts[:2])
    pkg_loc[key][0] += 1
    pkg_loc[key][1] += len(read(p))
god_pkgs = sorted([(v[1], v[0], k) for k, v in pkg_loc.items()], reverse=True)[:30]

result = {
    'copy_long_lines_top40': copies[:40],
    'copy_long_lines_count': len(copies),
    'dup_blocks_go_top30': bgo[:30],
    'dup_blocks_go_count': len(bgo),
    'dup_blocks_ts_top30': bts[:30],
    'dup_blocks_ts_count': len(bts),
    'similar_file_pairs_top40': similar[:40],
    'similar_file_pairs_count': len(similar),
    'dupe_basenames_count': len(dupe_names),
    'dupe_basenames_sample': dict(list(sorted(dupe_names.items(), key=lambda kv: -len(kv[1])))[:40]),
    'god_packages_top30': god_pkgs,
}
with open(os.path.join(OUT, 'scan2.json'), 'w', encoding='utf-8') as f:
    json.dump(result, f, ensure_ascii=False, indent=1)

print('copy_long_lines', len(copies), 'dup_blocks_go', len(bgo), 'dup_blocks_ts', len(bts),
      'similar_pairs', len(similar), 'dupe_basenames', len(dupe_names))
for c in copies[:12]:
    print('COPY', len(c['files']), c['files'][:3], '|', c['sample'][:80])
for s in similar[:12]:
    print('SIM', s['jaccard'], s['a'], s['b'], s['a_loc'], s['b_loc'])
for b in bgo[:6]:
    print('DUPGO', len(b['files']), b['files'][:4])
for b in bts[:6]:
    print('DUPTS', len(b['files']), b['files'][:4])
