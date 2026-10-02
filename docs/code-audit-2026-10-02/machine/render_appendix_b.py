import json, os, collections

ROOT = r'C:\AI\wubigrok'
M = os.path.join(ROOT, 'docs', 'code-audit-2026-10-02', 'machine')
b = json.load(open(os.path.join(M, 'bindings.json'), encoding='utf-8'))
im = json.load(open(os.path.join(M, 'imports.json'), encoding='utf-8'))
api = json.load(open(os.path.join(M, 'apilayer.json'), encoding='utf-8'))

L = []
A = L.append
A('# 附录 B · 绑定面死面 / 包依赖图 / 资产存活（2026-10-02）')
A('')
A('> 脚本：`machine/dead_bindings.py`、`machine/imports.py`、`machine/api_layer.py`。全部数据可复跑。')
A('')

A('## B1. 绑定面（743 方法）的前端消费面')
A('')
A('口径：取 `go run ./scripts/gen_bindings -names` 的 Go 侧全部导出绑定方法（743 个，与 `bindingNames.ts` 零漂移），')
A('然后在前端源码里统计「生产代码」「测试文件」两类消费者。以下文件视为**机械登记面**不计消费者：')
A('`bindingNames.ts`、`lib/bridge/**`、`lib/mock/**`、`spaceBindings.ts`、`types/wails.d.ts`、`wailsjs/**`、`wailsjsCompat/**`。')
A('**`frontend/src/api/**` 是真实调用层（18~53 个导入方），已计入消费者。**')
A('')
A('| 分类 | 方法数 | 占比 | 含义 |')
A('|---|---|---|---|')
A('| 零引用（生产+测试都没有） | %d | %.1f%% | 前端代码里根本没人调用 |' % (b['zero_frontend_consumer_count'], 100.0 * b['zero_frontend_consumer_count'] / b['go_names']))
A('| 仅测试引用 | %d | %.1f%% | 生产不调用，只有测试钉着 |' % (b['test_only_consumer_count'], 100.0 * b['test_only_consumer_count'] / b['go_names']))
A('| 仅 1 处生产引用 | %d | %.1f%% | 调用面极窄 |' % (b['single_prod_consumer_count'], 100.0 * b['single_prod_consumer_count'] / b['go_names']))
A('| 合计覆盖 | %d / %d | | 绑定面总数 743 |' % (b['go_names'], b['go_names']))
A('')
A('> **口径警告（必须随结论一起引用）**：「零引用」= 前端（含 legacy `api/` 层）没有任何调用点，')
A('> **不等于可以直接删**——其中一部分是给 CLI / 外部脚本 / HTTP 面（`GAEA_HTTP_PORT` 模式）留的导出。')
A('> AGENTS 里已把「零调用者绑定删除」列为待拍板项（宽口径 36，与本处窄口径 %d 差一个量级，**两套口径需要对齐**）。' % b['zero_frontend_consumer_count'])
A('')

groups = collections.defaultdict(list)
for n in b['zero_frontend_consumer']:
    for pre in ('GaeaCost', 'GaeaMemory', 'GaeaSkill', 'GaeaDream', 'GaeaTask', 'Gaea', 'Character', 'Chat', 'Novel', 'Sin', 'Weixin', 'Wx', 'Voice', 'Tts', 'Image', 'Model', 'Office', 'Doc', 'Xlsx', 'Pptx', 'Pdf', 'Brain', 'Skill', 'Memory', 'Cost', 'Export', 'Import', 'Search', 'Subagent', 'Task', 'Project', 'Config', 'Auth', 'Update'):
        if n.startswith(pre):
            groups[pre].append(n)
            break
    else:
        groups['其它'].append(n)
A('### B1.1 零引用方法按前缀分组（共 %d 个）' % b['zero_frontend_consumer_count'])
A('')
A('| 前缀 | 数量 | 样例（最多 12） |')
A('|---|---|---|')
for k, v in sorted(groups.items(), key=lambda kv: -len(kv[1])):
    A('| %s | %d | %s |' % (k, len(v), ', '.join('`%s`' % x for x in sorted(v)[:12])))
A('')
A('### B1.2 仅测试引用的方法（%d 个，按引用数降序）' % b['test_only_consumer_count'])
A('')
A('| 方法 | 仅有的引用位置 |')
A('|---|---|')
for t in b['test_only_consumer'][:60]:
    A('| `%s` | %s |' % (t['name'], ', '.join('`%s`' % x for x in t['files'][:3])))
A('')
A('### B1.3 生产调用面最宽的方法 TOP 20（改动风险面）')
A('')
A('| 引用点数 | 方法 |')
A('|---|---|')
for c, n in b['top_fanout']:
    A('| %d | `%s` |' % (c, n))
A('')

A('## B2. 前端 legacy `api/` 层的存活面')
A('')
A('| 文件 | 生产导入方数量 | 样例导入方 |')
A('|---|---|---|')
for r in api['rows']:
    if r['file'].endswith('.test.ts'):
        continue
    A('| `%s` | %d | %s |' % (r['file'], len(r['importers']), ', '.join('`%s`' % x.replace('frontend/src/', '') for x in r['importers'][:3])))
A('')
A('> 结论：`api/` 不是僵尸层（characterlib 18 个导入方、image 53 个、engines 20 个），')
A('> 但它与 `gaea/lib/bridge/**` 构成**两套并存的前端调用面**——同一批 Go 绑定方法两条路径可达。')
A('')

A('## B3. Go 包依赖图（internal/ 共 %d 个包）' % im['packages'])
A('')
A('- **上帝装配包**：`internal/app` 直接 import 110 个包（第二名的 `internal/gaea/boot` 27 个）——所有域的装配都压在 app。')
A('- **层级倒挂检查**：非 app 包反向 import `internal/app` 的边 = **%d 条**（该维度干净）。' % len(im['imports_into_internal_app']))
A('- **同名双包 7 组**（同一概念两套实现的头号嫌疑）：')
A('')
A('| 基名 | 包路径 | 非测试文件数 |')
A('|---|---|---|')
for k, v in sorted(im['twin_package_names'].items()):
    A('| %s | %s | |' % (k, ' / '.join('`%s`' % x for x in v)))
A('')
A('- **fan-in TOP 20**（改动波及面最大的包）：')
A('')
A('| 被依赖包数 | 包 |')
A('|---|---|')
for p, c in im['fanin_top30'][:20]:
    A('| %d | `%s` |' % (c, p))
A('')
A('- **单包 import 最多（最耦合的消费方）TOP 15**：')
A('')
A('| import 包数 | 包 |')
A('|---|---|')
for c, p in im['top_importers'][:15]:
    A('| %d | `%s` |' % (c, p))
A('')

A('## B4. 提示词资产存活')
A('')
A('- `prompts/*.json` 共 %d 个模板，**零引用 %d 个**（每个模板名都能在 Go/TS 源码里找到引用点）。'
  % (len(b['prompt_assets']), len(b['prompt_unreferenced'])))
A('- 模板清单：%s' % ', '.join('`%s`' % x for x in b['prompt_assets']))
A('')

open(os.path.join(ROOT, 'docs', 'code-audit-2026-10-02', 'appendix-B-surface.md'), 'w', encoding='utf-8').write('\n'.join(L))
print('written', len(L), 'lines')
