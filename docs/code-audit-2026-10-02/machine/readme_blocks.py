import json, os, glob, re, collections

ROOT = r'C:\AI\wubigrok'
AD = os.path.join(ROOT, 'docs', 'code-audit-2026-10-02')
M = os.path.join(AD, 'machine')
units = {}
for p in sorted(glob.glob(os.path.join(AD, 'units', '*.json'))):
    d = json.load(open(p, encoding='utf-8-sig'))
    units[d.get('id')] = d

GROUP = {
    '01 后端 gaea 内核': ['GA1', 'GA2', 'GA3', 'GA4', 'GA5', 'GA6'],
    '02 后端应用层·小说与轻语': ['AP1', 'AP2'],
    '03 后端应用层·办公/造价/进度/基础': ['AP3', 'AP4', 'AP5', 'AP6', 'AP7', 'AP8', 'AP9'],
    '04 内部领域包与 whisper': ['IN1', 'IN2', 'IN3', 'IN4'],
    '05 前端 React/TS': ['FE1', 'FE2', 'FE3', 'FE4', 'FE5', 'FE6', 'FE7'],
    '06 横切面': ['X1'],
}
ORDER = ['P0', 'P1', 'P2', 'P3']

L = []
A = L.append

tot = collections.Counter()
cat = collections.defaultdict(collections.Counter)
scores = []
for g, ids in GROUP.items():
    for i in ids:
        u = units.get(i)
        if not u:
            continue
        sc = u.get('score')
        scores.append(sc if isinstance(sc, (int, float)) else 0)
        for f in u.get('findings', []):
            s = f.get('severity', '?')
            c = f.get('category', '?')
            tot[s] += 1
            cat[c][s] += 1

A('### 评分总表（单元级，低分=屎山重）')
A('')
A('| 分册 | 单元 | 板块 | 评分 | findings | P0 | P1 |')
A('|---|---|---|---|---|---|---|')
for i, u in sorted(units.items(), key=lambda kv: (kv[1].get('score') if isinstance(kv[1].get('score'), (int, float)) else 99)):
    g = [k for k, v in GROUP.items() if i in v]
    fs = u.get('findings', [])
    c = collections.Counter(f.get('severity') for f in fs)
    A('| %s | %s | %s | %s | %d | %d | %d |' % ((g[0] if g else '?')[:2], i, (u.get('module') or '')[:34], u.get('score'), len(fs), c['P0'], c['P1']))
A('')
A('平均分 **%.2f / 10**（27 单元，等权）' % (sum(scores) / len(scores)))
A('')
A('### 严重度 × 类别矩阵')
A('')
A('| 类别 | P0 | P1 | P2 | P3 | 合计 |')
A('|---|---|---|---|---|---|')
for c, cc in sorted(cat.items(), key=lambda kv: -sum(kv[1].values())):
    A('| %s | %d | %d | %d | %d | %d |' % (c, cc['P0'], cc['P1'], cc['P2'], cc['P3'], sum(cc.values())))
A('| **合计** | **%d** | **%d** | **%d** | **%d** | **%d** |' % (tot['P0'], tot['P1'], tot['P2'], tot['P3'], sum(tot.values())))
A('')

A('### P0 全表（%d 条）' % tot['P0'])
A('')
A('| # | 单元 | 标题 | 位置 | 工作量 | 影响（摘） |')
A('|---|---|---|---|---|---|')
n = 0
for i, u in units.items():
    for f in u.get('findings', []):
        if f.get('severity') != 'P0':
            continue
        n += 1
        A('| %d | %s | %s | `%s:%s` | %s | %s |' % (n, i, (f.get('title') or '').replace('|', '/'), f.get('file'), f.get('line'), f.get('effort'), (f.get('impact') or '')[:110].replace('|', '/').replace('\n', ' ')))
A('')

A('### 分册 roll-up（单元级合计；去重后的 分册 内部计数见各分册 §六）')
A('')
A('| 分册 | 单元数 | 评分均值 | findings | P0 | P1 | P2 | P3 | 分册内拆刀数 |')
A('|---|---|---|---|---|---|---|---|---|')
FILES = {
    '01 后端 gaea 内核': '01-后端-gaea内核分册.md',
    '02 后端应用层·小说与轻语': '02-后端-应用层-小说与轻语分册.md',
    '03 后端应用层·办公/造价/进度/基础': '03-后端-应用层-办公造价进度与基础设施分册.md',
    '04 内部领域包与 whisper': '04-内部领域包与whisper分册.md',
    '05 前端 React/TS': '05-前端分册.md',
    '06 横切面': '06-横切面-重复实现与死代码分册.md',
}
for g, ids in GROUP.items():
    cc = collections.Counter()
    sc = []
    for i in ids:
        u = units.get(i)
        if not u:
            continue
        if isinstance(u.get('score'), (int, float)):
            sc.append(u['score'])
        for f in u.get('findings', []):
            cc[f.get('severity')] += 1
    path = os.path.join(AD, FILES[g])
    cut = 0
    if os.path.exists(path):
        txt = open(path, encoding='utf-8', errors='replace').read()
        m = re.search(r'## 四、拆刀建议(.*?)(?=\n## )', txt, re.S)
        if m:
            cut = len(re.findall(r'^\|\s*\d+\s*\|', m.group(1), re.M)) or len(re.findall(r'^\|\s*刀', m.group(1), re.M))
    A('| %s | %d | %.2f | %d | %d | %d | %d | %d | %d |' % (g, len([i for i in ids if i in units]), (sum(sc) / len(sc)) if sc else 0, sum(cc.values()), cc['P0'], cc['P1'], cc['P2'], cc['P3'], cut))
A('')
open(os.path.join(M, 'readme_blocks.md'), 'w', encoding='utf-8').write('\n'.join(L))
print('\n'.join(L[:12]))
print('...')
print('P0 total', tot['P0'], 'findings total', sum(tot.values()), 'avg', round(sum(scores) / len(scores), 3))
