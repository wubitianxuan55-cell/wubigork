import json, os, re, hashlib, collections, sys

ROOT = r'C:\AI\wubigrok'
OUT = os.path.join(ROOT, 'docs', 'code-audit-2026-10-02', 'machine')
os.makedirs(OUT, exist_ok=True)

SKIP_DIRS = {'node_modules', 'clones', 'backups', 'dist', 'build', 'releases', 'whisper_data',
             '.git', '.tmp', '.tmp-pdf', '.gaea', 'novels', 'design-system', '.agent-teams',
             '.dsh-vision-router', 'wailsjs', 'progress'}
SCAN_ROOTS = [os.path.join(ROOT, 'internal'), os.path.join(ROOT, 'frontend', 'src'),
              os.path.join(ROOT, 'shared'), os.path.join(ROOT, 'scripts'), ROOT]


def rel(p):
    return os.path.relpath(p, ROOT).replace('\\', '/')


def walk(exts, include_tests=True, roots=None):
    out = []
    seen = set()
    for r in (roots or [os.path.join(ROOT, 'internal'), os.path.join(ROOT, 'frontend', 'src')]):
        for dirpath, dirnames, filenames in os.walk(r):
            dirnames[:] = [d for d in dirnames if d not in SKIP_DIRS]
            for fn in filenames:
                if not fn.endswith(exts):
                    continue
                p = os.path.join(dirpath, fn)
                if p in seen:
                    continue
                seen.add(p)
                is_test = fn.endswith('.test.ts') or fn.endswith('.test.tsx') or fn.endswith('_test.go')
                if is_test and not include_tests:
                    continue
                out.append(p)
    return sorted(out)


def read(p):
    try:
        with open(p, 'r', encoding='utf-8', errors='replace') as f:
            return f.read().split('\n')
    except Exception:
        return []


# ---------- A. god files ----------
go_all = walk(('.go',))
ts_all = walk(('.ts', '.tsx'))
go_src = [p for p in go_all if not p.endswith('_test.go')]
go_test = [p for p in go_all if p.endswith('_test.go')]
ts_src = [p for p in ts_all if not (p.endswith('.test.ts') or p.endswith('.test.tsx'))]
ts_test = [p for p in ts_all if (p.endswith('.test.ts') or p.endswith('.test.tsx'))]

sizes = {}
for p in go_all + ts_all:
    sizes[p] = len(read(p))

god_go = sorted([(sizes[p], rel(p)) for p in go_src], reverse=True)
god_ts = sorted([(sizes[p], rel(p)) for p in ts_src], reverse=True)


# ---------- B. Go function lengths ----------
func_re = re.compile(r'^func\s+(?:\(([^)]*)\)\s*)?([A-Za-z_][A-Za-z0-9_]*)\s*\(')
long_funcs = []
for p in go_src:
    lines = read(p)
    i = 0
    while i < len(lines):
        m = func_re.match(lines[i])
        if not m:
            i += 1
            continue
        depth = 0
        started = False
        j = i
        while j < len(lines):
            depth += lines[j].count('{') - lines[j].count('}')
            if '{' in lines[j]:
                started = True
            if started and depth <= 0:
                break
            j += 1
        n = j - i + 1
        if n >= 120:
            long_funcs.append({'file': rel(p), 'line': i + 1, 'name': m.group(2), 'recv': (m.group(1) or '').strip(), 'loc': n})
        i = j + 1 if j > i else i + 1
long_funcs.sort(key=lambda x: -x['loc'])


# ---------- C. smell markers ----------
markers = {
    'TODO/FIXME/HACK/XXX': re.compile(r'\b(TODO|FIXME|HACK|XXX)\b'),
    '临时/兜底/兼容/废弃/占位(中文注释)': re.compile(r'(临时|兜底|兼容|废弃|占位|不再使用|历史遗留)'),
    'nolint': re.compile(r'nolint'),
    'panic(': re.compile(r'\bpanic\('),
    'recover()': re.compile(r'\brecover\(\)'),
    'time.Sleep': re.compile(r'time\.Sleep\('),
    '_, _ = / 丢错': re.compile(r'_,\s*_\s*[:=]|_ = \w+\(|,\s*_ :='),
    'interface{}/any 泛滥(go)': re.compile(r'interface\{\}|\bany\b'),
    'goroutine 起协程': re.compile(r'\bgo func\(|\bgo [a-zA-Z_][\w.]*\('),
}
marker_stats = collections.defaultdict(lambda: collections.Counter())
marker_locs = collections.defaultdict(list)
for label, rx in markers.items():
    for p in go_src + ts_src:
        for idx, line in enumerate(read(p), 1):
            if rx.search(line):
                marker_stats[label][rel(p).split('/')[0] + '/' + (rel(p).split('/')[1] if '/' in rel(p) else '')] += 1
                marker_locs[label].append((rel(p), idx, line.strip()[:120]))


# ---------- D. dead export heuristic (Go) ----------
decl_go = []
for p in go_src:
    for idx, line in enumerate(read(p), 1):
        m = func_re.match(line)
        if m and m.group(2)[:1].isupper():
            decl_go.append({'name': m.group(2), 'file': rel(p), 'line': idx, 'recv': (m.group(1) or '').strip(),
                            'iscode': rel(p).startswith('internal/gaea/')})
token_re = re.compile(r'[A-Za-z_][A-Za-z0-9_]*')
file_tokens = {}
for p in go_all:
    toks = set()
    for ln in read(p):
        toks.update(token_re.findall(ln))
    file_tokens[rel(p)] = toks
dead_candidates = []
NOISE = {'String', 'Error', 'ServeHTTP', 'Handle', 'Init', 'Start', 'Stop', 'Close', 'Run', 'New', 'Get', 'Set'}
for d in decl_go:
    if d['name'] in NOISE:
        continue
    hits = 0
    hit_files = []
    for f, toks in file_tokens.items():
        if f == d['file']:
            continue
        if d['name'] in toks:
            hits += 1
            if len(hit_files) < 3:
                hit_files.append(f)
    if hits == 0:
        dead_candidates.append({'name': d['name'], 'file': d['file'], 'line': d['line'], 'recv': d['recv']})
dead_candidates.sort(key=lambda x: x['file'])


# ---------- E. duplicate blocks ----------
def dup_blocks(paths, window=12, min_len=12):
    table = collections.defaultdict(list)
    for p in paths:
        lines = read(p)
        norm = []
        for idx, line in enumerate(lines, 1):
            s = line.strip()
            if len(s) < min_len:
                norm.append(None)
                continue
            if s.startswith('//') or s.startswith('*') or s.startswith('/*') or s.startswith('import'):
                norm.append(None)
                continue
            norm.append((s, idx))
        for i in range(len(norm) - window + 1):
            chunk = norm[i:i + window]
            if any(c is None for c in chunk):
                continue
            text = '\n'.join(c[0] for c in chunk)
            h = hashlib.md5(text.encode('utf-8')).hexdigest()
            table[h].append((rel(p), chunk[0][1]))
    groups = []
    for h, locs in table.items():
        files = sorted({l[0] for l in locs})
        if len(files) >= 2:
            groups.append({'files': files, 'count': len(locs), 'anchor': [list(l) for l in locs[:6]]})
    groups.sort(key=lambda g: (-len(g['files']), -g['count']))
    return groups


dup_go = dup_blocks(go_src)
dup_ts = dup_blocks(ts_src)


# ---------- F. frontend test smell ----------
skip_markers = []
for p in ts_test:
    for idx, line in enumerate(read(p), 1):
        if re.search(r'\b(it|test|describe)\.(skip|only|todo)\b|xit\(|xdescribe\(', line):
            skip_markers.append({'file': rel(p), 'line': idx, 'text': line.strip()[:120]})
go_skips = []
for p in go_test:
    for idx, line in enumerate(read(p), 1):
        if re.search(r't\.Skip\(', line):
            go_skips.append({'file': rel(p), 'line': idx, 'text': line.strip()[:120]})
big_tests = sorted([(sizes[p], rel(p)) for p in ts_test + go_test], reverse=True)[:25]


# ---------- G. repo hygiene ----------
def dsize(path):
    tot = 0
    for dp, dn, fn in os.walk(path):
        dn[:] = [d for d in dn if d not in ('node_modules',)]
        for f in fn:
            try:
                tot += os.path.getsize(os.path.join(dp, f))
            except Exception:
                pass
    return tot


hygiene = {}
for d in ['.tmp', 'backups', 'clones', 'dist', 'releases', 'whisper_data', 'docs', 'internal',
          'frontend/src', 'frontend/node_modules', 'node_modules']:
    p = os.path.join(ROOT, d)
    if os.path.isdir(p):
        hygiene[d] = round(dsize(p) / 1024 / 1024, 1)
changelog = os.path.getsize(os.path.join(ROOT, 'CHANGELOG.md'))

result = {
    'totals': {
        'go_files': len(go_all), 'go_src_files': len(go_src), 'go_test_files': len(go_test),
        'go_src_loc': sum(sizes[p] for p in go_src), 'go_test_loc': sum(sizes[p] for p in go_test),
        'ts_files': len(ts_all), 'ts_src_files': len(ts_src), 'ts_test_files': len(ts_test),
        'ts_src_loc': sum(sizes[p] for p in ts_src), 'ts_test_loc': sum(sizes[p] for p in ts_test),
    },
    'god_files_go_top40': god_go[:40],
    'god_files_ts_top40': god_ts[:40],
    'long_funcs_top60': long_funcs[:60],
    'long_funcs_count_ge120': len(long_funcs),
    'markers': {k: dict(v.most_common(12)) for k, v in marker_stats.items()},
    'marker_totals': {k: len(v) for k, v in marker_locs.items()},
    'marker_samples': {k: v[:12] for k, v in marker_locs.items()},
    'dead_export_candidates_count': len(dead_candidates),
    'dead_export_candidates': dead_candidates[:120],
    'dup_go_group_count': len(dup_go),
    'dup_go_top25': dup_go[:25],
    'dup_ts_group_count': len(dup_ts),
    'dup_ts_top25': dup_ts[:25],
    'ts_skip_markers': skip_markers[:40],
    'ts_skip_count': len(skip_markers),
    'go_skip_count': len(go_skips),
    'go_skips': go_skips[:20],
    'big_test_files': big_tests,
    'hygiene_mb': hygiene,
    'changelog_bytes': changelog,
}

with open(os.path.join(OUT, 'scan.json'), 'w', encoding='utf-8') as f:
    json.dump(result, f, ensure_ascii=False, indent=1)

print(json.dumps({k: result[k] for k in ['totals', 'long_funcs_count_ge120', 'marker_totals',
                                         'dead_export_candidates_count', 'dup_go_group_count',
                                         'dup_ts_group_count', 'ts_skip_count', 'go_skip_count',
                                         'hygiene_mb', 'changelog_bytes']}, ensure_ascii=False, indent=1))
