import json, os

ROOT = r'C:\AI\wubigrok'
M = os.path.join(ROOT, 'docs', 'code-audit-2026-10-02', 'machine')
s1 = json.load(open(os.path.join(M, 'scan.json'), encoding='utf-8'))
s2 = json.load(open(os.path.join(M, 'scan2.json'), encoding='utf-8'))

L = []
A = L.append
A('# 附录 A · 机械扫描原始数据（2026-10-02）')
A('')
A('> 本附录由 `docs/code-audit-2026-10-02/machine/scan.py` + `scan2.py` 生成，全部数据可由脚本复跑（工作目录 C:/AI/wubigrok）。')
A('> 口径：只统计 `internal/` 与 `frontend/src/` 下的源码（排除 node_modules / clones / backups / dist / build / releases / whisper_data / .tmp / novels / wailsjs）。')
A('')
A('## A0. 规模与门禁基线')
A('')
t = s1['totals']
A('| 项 | 文件数 | 非测试 LOC | 测试文件 | 测试 LOC |')
A('|---|---|---|---|---|')
A('| Go（internal/） | %d | %d | %d | %d |' % (t['go_files'], t['go_src_loc'], t['go_test_files'], t['go_test_loc']))
A('| TS/TSX（frontend/src） | %d | %d | %d | %d |' % (t['ts_files'], t['ts_src_loc'], t['ts_test_files'], t['ts_test_loc']))
A('| **合计** | **%d** | **%d** | **%d** | **%d** |' % (t['go_files'] + t['ts_files'], t['go_src_loc'] + t['ts_src_loc'], t['go_test_files'] + t['ts_test_files'], t['go_test_loc'] + t['ts_test_loc']))
A('')
A('- `go vet ./...`：exit 0，0 issues（2026-10-02 本机实测）')
A('- `golangci-lint run ./...`（v2.14.0，errcheck/govet/ineffassign/staticcheck/unused/misspell 全开）：`0 issues.`')
A('- 结论：**静态检查层已经清零，剩下的全是结构层问题**（下面每一项都是 lint 抓不到的）。')
A('')

A('## A1. 上帝文件')
A('')
A('### A1.1 Go 非测试文件 LOC TOP 40（>600 行即为高发区）')
A('')
A('| # | LOC | 文件 |')
A('|---|---|---|')
for i, (n, p) in enumerate(s1['god_files_go_top40'], 1):
    A('| %d | %d | `%s` |' % (i, n, p))
A('')
A('### A1.2 前端非测试文件 LOC TOP 40')
A('')
A('| # | LOC | 文件 |')
A('|---|---|---|')
for i, (n, p) in enumerate(s1['god_files_ts_top40'], 1):
    A('| %d | %d | `%s` |' % (i, n, p))
A('')

gf = json.load(open(os.path.join(M, 'gofunc.json'), encoding='utf-8'))
longf = gf['long_funcs_ge100']
A('## A2. 上帝函数（Go，AST 精确统计，单函数 ≥100 行，共 %d 个；其中 ≥200 行 %d 个）'
  % (len(longf), len([f for f in longf if f['loc'] >= 200])))
A('')
A('> 口径：`go/parser` + `go/ast` 精确函数跨度（不是花括号计数），工具 `machine/godfunc`（源码 `.tmp/godfunc/main.go`），只算 `internal/` 非测试文件。')
A('')
A('| # | 行数 | 位置 | 方法/函数 |')
A('|---|---|---|---|')
for i, f in enumerate(longf[:60], 1):
    recv = (f['recv'] + '.') if f['recv'] else ''
    A('| %d | %d | `%s:%d` | `%s%s` |' % (i, f['loc'], f['file'], f['line'], recv, f['name']))
A('')
A('### A2.1 单文件最大函数 TOP 25（文件级上帝化程度）')
A('')
A('| # | 最大函数行数 | 文件 | 该文件函数数 | 位置 |')
A('|---|---|---|---|---|')
for i, f in enumerate(gf['files'][:25], 1):
    A('| %d | %d | `%s` | %d | `%s` |' % (i, f['max_func'], f['file'], f['funcs'], f['max_func_at']))
A('')

A('## A3. 坏味道标记词频（非测试源码）')
A('')
A('| 模式 | 命中数 |')
A('|---|---|')
for k, v in s1['marker_totals'].items():
    A('| %s | %d |' % (k, v))
A('')
for k, v in s1['marker_samples'].items():
    A('### 样例：%s' % k)
    A('')
    for f, ln, txt in v[:8]:
        A('- `%s:%d` — `%s`' % (f, ln, txt.replace('|', '\\|')))
    A('')

A('## A4. 死代码候选（Go 导出函数/方法，除定义文件外全仓零引用，共 %d 个）' % s1['dead_export_candidates_count'])
A('')
A('> 启发式：导出名（首字母大写）在其它任何 .go 文件（含测试）的词元集合中都不出现。绑定面经 Wails 反射/生成代码调用者会自动落此列，判定时需人工确认（尤其 `internal/app/gaea_*` 与 `bindings_*`）。')
A('')
A('| # | 位置 | 接收者 | 名称 |')
A('|---|---|---|---|')
for i, d in enumerate(s1['dead_export_candidates'][:80], 1):
    A('| %d | `%s:%d` | `%s` | `%s` |' % (i, d['file'], d['line'], d['recv'][:28], d['name']))
A('')
A('（仅列前 80 条，完整清单见 `machine/scan.json` 的 `dead_export_candidates`）')
A('')

A('## A5. 复制粘贴证据')
A('')
A('### A5.1 长行原样复制（≥80 字符的同一行出现在 ≥2 个文件，共 %d 组）' % s2['copy_long_lines_count'])
A('')
A('| # | 涉及文件数 | 文件（最多 5） | 复制的行（截断） |')
A('|---|---|---|---|')
for i, c in enumerate(s2['copy_long_lines_top40'][:30], 1):
    A('| %d | %d | %s | `%s` |' % (i, len(c['files']), ', '.join('`%s`' % x for x in c['files'][:5]), c['sample'][:100].replace('|', '\\|')))
A('')
A('### A5.2 逐字重复代码块（归一化后连续 10 行完全相同）')
A('')
A('- Go：%d 组；TS：%d 组' % (s2['dup_blocks_go_count'], s2['dup_blocks_ts_count']))
A('')
A('| # | 语言 | 涉及文件数 | 文件 |')
A('|---|---|---|---|')
for i, b in enumerate(s2['dup_blocks_go_top30'][:20], 1):
    A('| %d | Go | %d | %s |' % (i, len(b['files']), ', '.join('`%s`' % x for x in b['files'][:5])))
for i, b in enumerate(s2['dup_blocks_ts_top30'][:20], 1):
    A('| %d | TS | %d | %s |' % (i, len(b['files']), ', '.join('`%s`' % x for x in b['files'][:5])))
A('')
A('### A5.3 同区域高相似文件对（≥150 行文件的词元 Jaccard ≥0.30，平行实现候选）')
A('')
A('| # | Jaccard | A（LOC） | B（LOC） |')
A('|---|---|---|---|')
for i, p in enumerate(s2['similar_file_pairs_top40'], 1):
    A('| %d | %.3f | `%s`（%d） | `%s`（%d） |' % (i, p['jaccard'], p['a'], p['a_loc'], p['b'], p['b_loc']))
A('')
A('### A5.4 同名文件散布（同名基线出现在 ≥2 条路径，共 %d 组，取最多的 20 组）' % s2['dupe_basenames_count'])
A('')
for k, v in list(s2['dupe_basenames_sample'].items())[:20]:
    A('- `%s`（%d 处）：%s' % (k, len(v), ', '.join('`%s`' % x for x in v[:6])))
A('')

A('## A6. 测试卫生')
A('')
A('- Go `t.Skip(` 命中 %d 处（前 20）：' % s1['go_skip_count'])
for g in s1['go_skips'][:20]:
    A('  - `%s:%d` — `%s`' % (g['file'], g['line'], g['text'].replace('|', '\\|')))
A('- TS `it.skip/only/todo` 命中 %d 处' % s1['ts_skip_count'])
A('')
A('### 最大的测试文件 TOP 25')
A('')
A('| # | LOC | 文件 |')
A('|---|---|---|')
for i, (n, p) in enumerate(s1['big_test_files'], 1):
    A('| %d | %d | `%s` |' % (i, n, p))
A('')

A('## A7. 上帝包（LOC TOP 30 的 Go 包目录）')
A('')
A('| # | LOC | 文件数 | 包目录 |')
A('|---|---|---|---|')
for loc, cnt, k in s2['god_packages_top30']:
    A('| %d | %d | %d | `%s` |' % (1, loc, cnt, k))
A('')

A('## A8. 仓库体积（MB）与文档水位')
A('')
A('| 目录 | MB |')
A('|---|---|')
for k, v in sorted(s1['hygiene_mb'].items(), key=lambda kv: -kv[1]):
    A('| `%s` | %s |' % (k, v))
A('')
A('- `CHANGELOG.md` = %d 字节（%.2f MB，单文件）' % (s1['changelog_bytes'], s1['changelog_bytes'] / 1024 / 1024))
A('')

open(os.path.join(ROOT, 'docs', 'code-audit-2026-10-02', 'appendix-A-mechanical.md'), 'w', encoding='utf-8').write('\n'.join(L))
print('written', len(L), 'lines')
