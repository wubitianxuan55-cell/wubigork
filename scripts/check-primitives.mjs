// 原语平行实现守卫（2026-10-02 全仓审计「06 横切面分册 · 刀1」）——warn 档，可复跑，暂不接 CI。
// 理由（06 分册第 16/221 行）：截断原语从审计初记的 8 处涨到 41 个定义，「收敛」从没配过门禁，
// 于是每一次收敛都在下一版被复制回去。本脚本先把「现存量」钉成基线，只拦「新增项」。
//
// 四项检测（正则即口径，注释里的正则就是代码里用的正则）：
//   ① 截断原语定义   Go internal/**/*.go
//      顶层函数：/^func[ \t]+(truncate|clip)[A-Za-z0-9_]*(?:[ \t]*\[[^\]]*\])?[ \t]*\(/i
//        grep 等价：grep -rIniE '^func[[:space:]]+(truncate|clip)' internal --include='*.go' | wc -l
//        注意必须带 -i：大小写敏感只有 37，含大写 Truncate/TruncateRef/TruncateToolOutput(With) 4 处才是审计的 41。
//      方法形式：/^func[ \t]*\([^)]*\)[ \t]*(truncate|clip)[A-Za-z0-9_]*(?:[ \t]*\[[^\]]*\])?[ \t]*\(/（大小写敏感）
//        只收小写开头的未导出原语；大写 Truncate 方法（context.Store.Truncate 等）语义是「截断存储条目」，
//        不是字符串截断，故不收（避免污染收敛目标）。
//   ② 原子写旁路     internal/app/**/*.go 里的 /(?:os\.CreateTemp|ioutil\.TempFile)[ \t]*\(/
//       口径只看 internal/app（刀1 指定）；全仓数字由输出里的参考行给出，不入基线。
//   ③ 前端旁路       frontend/src/**/*.{ts,tsx} 里的 /\ba\.download[ \t]*=/ 与 /\batob[ \t]*\(/
//       豁免两个统一出口本体：frontend/src/gaea/lib/saveFile.ts、frontend/src/gaea/lib/bytes.ts。
//   ④ heredoc 伪写   .go 源文件内容里出现 heredoc 形态（审计 P0#23/源 X1-01）：
//       重定向形态 /<<-?[ \t]*['"]?(EOF|PYEOF|GOEOF|JSONEOF|PY|PYTHON|EOS)['"]?/
//       泄漏定界行 /^(EOF|PYEOF|GOEOF|JSONEOF)[ \t]*$/
//       在册坑五犯（CHANGELOG 2026-10-02 五处）：Bash 工具 heredoc 把 `\n` 降级真实换行进 Go 字符串、
//       python 源码同样被降级——agent 用 shell 伪写源文件时定界符/形态会泄漏进文件本身。基线=0：任何
//       命中都按「新增」报（warn 档不 gate，strict 档 exit 1）。
//
// 认键规则（关键）：基线里的每一项用「文件 + 形式/API + 该文件内序号」认键，**行号只作参考不入键**——
// 无关编辑导致的行号漂移不该被报成「新增」。
//
// 用法：
//   node scripts/check-primitives.mjs                  默认 warn 档：打印表格 + 基线对比 + 新增清单，exit 0
//   node scripts/check-primitives.mjs --strict         有超出基线的新增项则 exit 1（接 CI 时用）
//   node scripts/check-primitives.mjs --json           机读 JSON（可叠加 --strict）
//   node scripts/check-primitives.mjs --write-baseline 用本次真实扫描结果覆写基线文件（人工评审后再用）
// 退出码：0=通过 / 1=--strict 且有新增 / 2=用法错、基线缺失或写入失败。
import fs from 'node:fs'
import path from 'node:path'

const BASELINE_PATH = 'docs/code-audit-2026-10-02/machine/primitive-baseline.json'
const AUDIT_DOC = 'docs/code-audit-2026-10-02/06-横切面-重复实现与死代码分册.md'

// 与 scripts/check-docs.mjs 同口径的跳过目录（多一个 wailsjs：Wails 生成物）
const SKIP_DIRS = /(^|[\\/])(node_modules|clones|backups|dist|build|wailsjs|\.tmp)([\\/]|$)/
const SKIP_LABEL = 'node_modules/clones/backups/dist/build/wailsjs/.tmp'

// ── 口径正则（改这里必须同步改文件头注释与 README） ────────────────────────
const RE_TRUNC_FUNC = /^func[ \t]+(?<name>(?:truncate|clip)[A-Za-z0-9_]*)(?:[ \t]*\[[^\]]*\])?[ \t]*\(/i
const RE_TRUNC_METHOD = /^func[ \t]*\([^)]*\)[ \t]*(?<name>(?:truncate|clip)[A-Za-z0-9_]*)(?:[ \t]*\[[^\]]*\])?[ \t]*\(/
const RE_ATOMIC_G = /\b(?:os\.CreateTemp|ioutil\.TempFile)[ \t]*\(/g
const RE_DOWNLOAD = /\ba\.download[ \t]*=/
const RE_ATOB = /\batob[ \t]*\(/
const RE_HEREDOC_OPENER = /<<-?[ \t]*['"]?(EOF|PYEOF|GOEOF|JSONEOF|PY|PYTHON|EOS)['"]?/
const RE_HEREDOC_DELIM = /^(EOF|PYEOF|GOEOF|JSONEOF)[ \t]*$/
const EXEMPT_FRONTEND = new Set([
  'frontend/src/gaea/lib/saveFile.ts',
  'frontend/src/gaea/lib/bytes.ts',
])

// ── 参数 ───────────────────────────────────────────────────────────────────
const argv = process.argv.slice(2)
const flags = new Set(argv)
for (const a of argv) {
  if (!['--strict', '--json', '--write-baseline', '--help', '-h'].includes(a)) {
    console.error(`未知参数：${a}\n用法：node scripts/check-primitives.mjs [--strict] [--json] [--write-baseline]`)
    process.exit(2)
  }
}
if (flags.has('--help') || flags.has('-h')) {
  console.log('用法：node scripts/check-primitives.mjs [--strict] [--json] [--write-baseline]')
  process.exit(0)
}
const STRICT = flags.has('--strict')
const JSON_OUT = flags.has('--json')
const WRITE_BASELINE = flags.has('--write-baseline')

// ── 工具 ───────────────────────────────────────────────────────────────────
const posix = (p) => p.split(path.sep).join('/')
const readLines = (file) => fs.readFileSync(file, 'utf8').split(/\r?\n/)
const clip = (s, n = 120) => (s.length > n ? s.slice(0, n) + '…' : s)

function walk(dir, exts, out = []) {
  if (!fs.existsSync(dir)) return out
  for (const e of fs.readdirSync(dir, { withFileTypes: true })) {
    const p = posix(path.join(dir, e.name))
    if (SKIP_DIRS.test(p)) continue
    if (e.isDirectory()) walk(p, exts, out)
    else if (exts.some((x) => e.name.endsWith(x))) out.push(p)
  }
  return out
}

// 显示宽度对齐（CJK 记 2 列），让表格在中文标签下仍然整齐
const WIDE = /[\u1100-\u115F\u2E80-\uA4CF\uAC00-\uD7A3\uF900-\uFAFF\uFE30-\uFE4F\uFF00-\uFF60\uFFE0-\uFFE6]/
const dispWidth = (s) => {
  let w = 0
  for (const ch of s) w += WIDE.test(ch) ? 2 : 1
  return w
}
const pad = (s, w) => s + ' '.repeat(Math.max(0, w - dispWidth(s)))
function renderTable(rows) {
  const widths = []
  for (const row of rows) row.forEach((c, i) => { widths[i] = Math.max(widths[i] ?? 0, dispWidth(String(c))) })
  return rows.map((row) => row.map((c, i) => pad(String(c), widths[i])).join('  ').replace(/\s+$/, '')).join('\n')
}
const isTestScope = (f) => /\.(test|spec)\.(ts|tsx)$/.test(f) || /(^|\/)(__tests__|__mocks__)\//.test(f)

function gitHead() {
  try {
    const raw = fs.readFileSync('.git/HEAD', 'utf8').trim()
    if (raw.startsWith('ref: ')) {
      const ref = raw.slice(5).trim()
      const p = path.join('.git', ref)
      if (fs.existsSync(p)) return fs.readFileSync(p, 'utf8').trim().slice(0, 12)
      if (fs.existsSync(path.join('.git', 'packed-refs'))) {
        const m = fs.readFileSync(path.join('.git', 'packed-refs'), 'utf8').match(new RegExp(`refs/[^\\s]*${ref.replace('refs/', '')}\\s+([0-9a-f]{40})`))
        if (m) return m[1].slice(0, 12)
      }
      return ref
    }
    return raw.slice(0, 12) // detached HEAD
  } catch {
    return null
  }
}

// ── 扫描 ───────────────────────────────────────────────────────────────────
function scanTruncate(files) {
  const items = []
  for (const file of files) {
    const lines = readLines(file)
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i]
      if (!line.startsWith('func')) continue // 定义行顶格；`^func` 保证注释/字符串不误伤
      const free = RE_TRUNC_FUNC.exec(line)
      if (free) {
        items.push({ file, line: i + 1, name: free.groups.name, form: 'func', text: clip(line.trim()) })
        continue
      }
      const meth = RE_TRUNC_METHOD.exec(line)
      if (meth) items.push({ file, line: i + 1, name: meth.groups.name, form: 'method', text: clip(line.trim()) })
    }
  }
  for (const it of items) it.key = `${it.file}::${it.form}::${it.name}`
  return items.sort(byPos)
}

function scanAtomicWrite(files) {
  const items = []
  for (const file of files) {
    if (!file.startsWith('internal/app/')) continue
    const seq = new Map() // API → 该文件内出现序号（认键用，与行号解耦）
    const lines = readLines(file)
    for (let i = 0; i < lines.length; i++) {
      for (const m of lines[i].matchAll(RE_ATOMIC_G)) {
        const api = m[0].replace(/[ \t]*\($/, '')
        const n = (seq.get(api) ?? 0) + 1
        seq.set(api, n)
        items.push({ file, line: i + 1, api, name: api, form: api, key: `${file}::${api}#${n}`, text: clip(lines[i].trim()) })
      }
    }
  }
  return items.sort(byPos)
}

function scanFrontend(files) {
  const items = []
  for (const file of files) {
    if (EXEMPT_FRONTEND.has(file)) continue
    const seq = new Map()
    const lines = readLines(file)
    for (let i = 0; i < lines.length; i++) {
      for (const [kind, re] of [['download-assign', RE_DOWNLOAD], ['atob', RE_ATOB]]) {
        if (!re.test(lines[i])) continue
        const n = (seq.get(kind) ?? 0) + 1
        seq.set(kind, n)
        items.push({
          file, line: i + 1, kind, name: kind, form: kind, test: isTestScope(file),
          key: `${kind}:${file}#${n}`, text: clip(lines[i].trim()),
        })
      }
    }
  }
  return items.sort(byPos)
}

function scanHeredoc(files) {
  const items = []
  for (const file of files) {
    if (!file.endsWith('.go')) continue
    const seq = new Map()
    const lines = readLines(file)
    for (let i = 0; i < lines.length; i++) {
      const kind = RE_HEREDOC_OPENER.test(lines[i]) ? 'heredoc-opener' : RE_HEREDOC_DELIM.test(lines[i]) ? 'heredoc-delim' : null
      if (!kind) continue
      const n = (seq.get(kind) ?? 0) + 1
      seq.set(kind, n)
      items.push({ file, line: i + 1, kind, name: kind, form: kind, key: `${kind}:${file}#${n}`, text: clip(lines[i].trim()) })
    }
  }
  return items.sort(byPos)
}

function byPos(a, b) {
  if (a.file !== b.file) return a.file < b.file ? -1 : 1
  if (a.line !== b.line) return a.line - b.line
  return String(a.form).localeCompare(String(b.form))
}

// ── 汇合 ───────────────────────────────────────────────────────────────────
const goFiles = walk('internal', ['.go']).sort()
const tsFiles = walk('frontend/src', ['.ts', '.tsx']).sort()
const truncItems = scanTruncate(goFiles)
const atomicItems = scanAtomicWrite(goFiles)
const frontendItems = scanFrontend(tsFiles)
const heredocItems = scanHeredoc(goFiles)

const categories = {
  truncate: {
    label: '① 截断原语定义',
    items: truncItems,
    count: truncItems.length,
    freeCount: truncItems.filter((i) => i.form === 'func').length,
    methodCount: truncItems.filter((i) => i.form === 'method').length,
  },
  atomicWrite: {
    label: '② 原子写旁路',
    items: atomicItems,
    count: atomicItems.length,
  },
  frontendBypass: {
    label: '③ 前端保存/解码旁路',
    items: frontendItems,
    count: frontendItems.length,
    downloadCount: frontendItems.filter((i) => i.kind === 'download-assign').length,
    atobCount: frontendItems.filter((i) => i.kind === 'atob').length,
    testScopeCount: frontendItems.filter((i) => i.test).length,
  },
  heredoc: {
    label: '④ heredoc 伪写形态',
    items: heredocItems,
    count: heredocItems.length,
  },
}

const scanMeta = { goFiles: goFiles.length, tsFiles: tsFiles.length, skipDirs: SKIP_LABEL }
const regexMeta = {
  truncateFunc: String(RE_TRUNC_FUNC),
  truncateMethod: String(RE_TRUNC_METHOD),
  atomicWrite: String(RE_ATOMIC_G),
  downloadAssign: String(RE_DOWNLOAD),
  atob: String(RE_ATOB),
  heredocOpener: String(RE_HEREDOC_OPENER),
  heredocDelim: String(RE_HEREDOC_DELIM),
  exemptFrontend: [...EXEMPT_FRONTEND],
}
const auditMeta = {
  source: AUDIT_DOC,
  truncateCountAsAudited: 41,
  truncateCountCaseSensitiveGrep: 37,
  truncateConvergeTarget: '≤5（审计附录 A / 刀4 验收线）',
  atomicWriteInternalAppAsEnumerated: 13,
  frontendDownloadAssignProduction: 2,
}

// ── --write-baseline ───────────────────────────────────────────────────────
if (WRITE_BASELINE) {
  const baseline = {
    schema: 'gaea-primitive-baseline/1',
    generatedAt: new Date().toISOString(),
    generatedBy: 'node scripts/check-primitives.mjs --write-baseline',
    head: gitHead(),
    audit: auditMeta,
    scan: scanMeta,
    regexes: regexMeta,
    categories: {
      truncate: {
        count: categories.truncate.count,
        freeCount: categories.truncate.freeCount,
        methodCount: categories.truncate.methodCount,
        items: truncItems,
      },
      atomicWrite: { count: categories.atomicWrite.count, items: atomicItems },
      frontendBypass: {
        count: categories.frontendBypass.count,
        downloadCount: categories.frontendBypass.downloadCount,
        atobCount: categories.frontendBypass.atobCount,
        testScopeCount: categories.frontendBypass.testScopeCount,
        items: frontendItems,
      },
      heredoc: { count: categories.heredoc.count, items: heredocItems },
    },
  }
  try {
    fs.mkdirSync(path.dirname(BASELINE_PATH), { recursive: true })
    fs.writeFileSync(BASELINE_PATH, JSON.stringify(baseline, null, 2) + '\n', 'utf8')
  } catch (e) {
    console.error(`基线写入失败：${BASELINE_PATH} — ${e.message}`)
    process.exit(2)
  }
  console.log(`基线已写入 ${BASELINE_PATH}`)
  console.log(`  截断原语 ${categories.truncate.count}（顶层 ${categories.truncate.freeCount} + 方法 ${categories.truncate.methodCount}）`)
  console.log(`  原子写旁路 internal/app ${categories.atomicWrite.count}`)
  console.log(`  前端旁路 ${categories.frontendBypass.count}（a.download= ${categories.frontendBypass.downloadCount} / atob( ${categories.frontendBypass.atobCount}）`)
  console.log(`  heredoc 伪写形态 ${categories.heredoc.count}`)
  process.exit(0)
}

// ── 与基线对比 ─────────────────────────────────────────────────────────────
let baseline = null
if (fs.existsSync(BASELINE_PATH)) {
  try {
    baseline = JSON.parse(fs.readFileSync(BASELINE_PATH, 'utf8'))
    if (baseline.schema !== 'gaea-primitive-baseline/1') throw new Error(`schema 不匹配：${baseline.schema}`)
  } catch (e) {
    console.error(`基线无法解析：${BASELINE_PATH} — ${e.message}`)
    process.exit(2)
  }
} else {
  console.error(`基线缺失：${BASELINE_PATH}`)
  console.error('先跑一次 node scripts/check-primitives.mjs --write-baseline（人工评审基线后再入库）')
  process.exit(2)
}

const compare = (kind, items) => {
  const base = new Set((baseline.categories[kind]?.items ?? []).map((i) => i.key))
  const keys = new Set(items.map((i) => i.key))
  return {
    baselineCount: base.size,
    added: items.filter((i) => !base.has(i.key)),
    removed: (baseline.categories[kind]?.items ?? []).filter((i) => !keys.has(i.key)),
  }
}
const cmp = {
  truncate: compare('truncate', truncItems),
  atomicWrite: compare('atomicWrite', atomicItems),
  frontendBypass: compare('frontendBypass', frontendItems),
  heredoc: compare('heredoc', heredocItems),
}
// 基线里的分项计数（表格要按「顶层函数 / 方法形式 / 两种前端模式」分列对比）
const baseItems = {
  truncate: baseline.categories.truncate?.items ?? [],
  atomicWrite: baseline.categories.atomicWrite?.items ?? [],
  frontendBypass: baseline.categories.frontendBypass?.items ?? [],
  heredoc: baseline.categories.heredoc?.items ?? [],
}
const baseFree = baseItems.truncate.filter((i) => i.form === 'func').length
const baseMethod = baseItems.truncate.filter((i) => i.form === 'method').length
const baseDownload = baseItems.frontendBypass.filter((i) => i.kind === 'download-assign').length
const baseAtob = baseItems.frontendBypass.filter((i) => i.kind === 'atob').length
const addedTotal = cmp.truncate.added.length + cmp.atomicWrite.added.length + cmp.frontendBypass.added.length + cmp.heredoc.added.length
const removedTotal = cmp.truncate.removed.length + cmp.atomicWrite.removed.length + cmp.frontendBypass.removed.length + cmp.heredoc.removed.length
const exitCode = STRICT && addedTotal > 0 ? 1 : 0

// ── --json ─────────────────────────────────────────────────────────────────
if (JSON_OUT) {
  const payload = {
    tool: 'check-primitives',
    schema: 'gaea-primitive-guard/1',
    mode: STRICT ? 'strict' : 'warn',
    ranAt: new Date().toISOString(),
    head: gitHead(),
    auditDoc: AUDIT_DOC,
    scan: scanMeta,
    regexes: regexMeta,
    baseline: { path: BASELINE_PATH, generatedAt: baseline.generatedAt ?? null, head: baseline.head ?? null },
    counts: {
      truncate: { free: categories.truncate.freeCount, method: categories.truncate.methodCount, total: categories.truncate.count, baseline: cmp.truncate.baselineCount },
      atomicWrite: { total: categories.atomicWrite.count, baseline: cmp.atomicWrite.baselineCount },
      frontendBypass: {
        downloadAssign: categories.frontendBypass.downloadCount,
        atob: categories.frontendBypass.atobCount,
        total: categories.frontendBypass.count,
        testScope: categories.frontendBypass.testScopeCount,
        baseline: cmp.frontendBypass.baselineCount,
      },
      heredoc: { total: categories.heredoc.count, baseline: cmp.heredoc.baselineCount },
    },
    added: { total: addedTotal, truncate: cmp.truncate.added, atomicWrite: cmp.atomicWrite.added, frontendBypass: cmp.frontendBypass.added, heredoc: cmp.heredoc.added },
    removed: { total: removedTotal, truncate: cmp.truncate.removed, atomicWrite: cmp.atomicWrite.removed, frontendBypass: cmp.frontendBypass.removed, heredoc: cmp.heredoc.removed },
    hits: { truncate: truncItems, atomicWrite: atomicItems, frontendBypass: frontendItems, heredoc: heredocItems },
    exitCode,
  }
  console.log(JSON.stringify(payload, null, 2))
  process.exit(exitCode)
}

// ── 人读输出 ───────────────────────────────────────────────────────────────
const dash = (n) => (n === 0 ? '0' : `+${n}`)
console.log(`原语平行实现守卫（${STRICT ? 'strict' : 'warn'} 档）· 2026-10-02 审计 06 分册 · 刀1`)
console.log(`扫描：Go internal/**/*.go ${scanMeta.goFiles} 个文件 · TS frontend/src/**/*.{ts,tsx} ${scanMeta.tsFiles} 个文件 · 跳过 ${SKIP_LABEL}`)
console.log(`基线：${BASELINE_PATH}（${baseline.generatedAt ?? '无日期'}${baseline.head ? ` @ ${baseline.head}` : ''}）`)
console.log('')
console.log(renderTable([
  ['检测项', '当前', '基线', '新增', '审计口径'],
  ['① 截断原语 · 顶层函数 truncate*/clip*', categories.truncate.freeCount, baseFree, dash(cmp.truncate.added.filter((i) => i.form === 'func').length), '41（grep -i 口径）'],
  ['① 截断原语 · 方法形式（附加口径）', categories.truncate.methodCount, baseMethod, dash(cmp.truncate.added.filter((i) => i.form === 'method').length), '不单独计数'],
  ['① 合计', categories.truncate.count, cmp.truncate.baselineCount, dash(cmp.truncate.added.length), '41'],
  ['② 原子写旁路 · internal/app', categories.atomicWrite.count, cmp.atomicWrite.baselineCount, dash(cmp.atomicWrite.added.length), '13（分册列举）'],
  ['③ 前端旁路 · 裸 a.download =', categories.frontendBypass.downloadCount, baseDownload, dash(cmp.frontendBypass.added.filter((i) => i.kind === 'download-assign').length), '2（生产码）'],
  ['③ 前端旁路 · 裸 atob(', categories.frontendBypass.atobCount, baseAtob, dash(cmp.frontendBypass.added.filter((i) => i.kind === 'atob').length), '审计未逐个列举'],
  ['③ 合计（含测试文件命中）', categories.frontendBypass.count, cmp.frontendBypass.baselineCount, dash(cmp.frontendBypass.added.length), '—'],
  ['④ heredoc 伪写形态（.go）', categories.heredoc.count, cmp.heredoc.baselineCount, dash(cmp.heredoc.added.length), '基线 0（P0#23）'],
]))
console.log('')
console.log(`口径：① ${regexMeta.truncateFunc}`)
console.log(`      （方法形式：${regexMeta.truncateMethod}）`)
console.log(`      ② ${regexMeta.atomicWrite}`)
console.log(`      ③ ${regexMeta.downloadAssign} · ${regexMeta.atob}  豁免 ${[...EXEMPT_FRONTEND].join(' / ')}`)
console.log(`      ④ ${regexMeta.heredocOpener} · ${regexMeta.heredocDelim}`)

for (const [kind, c] of Object.entries(cmp)) {
  const label = categories[kind].label
  if (c.added.length) {
    console.log('')
    console.log(`  NEW   ${label} 新增 ${c.added.length} 处（未在基线内）：`)
    for (const i of c.added) console.log(`        ${i.file}:${i.line}  [${i.form}]  ${i.text}`)
  } else {
    console.log(`  OK    ${label} 无新增（存量 ${c.baselineCount} 处全在基线内，不逐一列出）`)
  }
  if (c.removed.length) {
    console.log(`  NOTE  ${label} 基线内 ${c.removed.length} 处已消失（收敛有进展，确认后跑 --write-baseline 更新基线）：`)
    for (const i of c.removed) console.log(`        ${i.file}:${i.line ?? '?'}  [${i.form}]`)
  }
}

console.log('')
if (STRICT) {
  if (addedTotal > 0) {
    console.log(`原语守卫 STRICT FAIL：新增 ${addedTotal} 处超出基线——接 CI 后此处 exit 1`)
    process.exit(1)
  }
  console.log(`原语守卫 STRICT OK：无新增（截断 ${categories.truncate.count} / 原子写 ${categories.atomicWrite.count} / 前端 ${categories.frontendBypass.count} / heredoc ${categories.heredoc.count}）`)
  process.exit(0)
}
if (addedTotal > 0) {
  console.log(`原语守卫 WARN：新增 ${addedTotal} 处（见上）——warn 档不 gate，exit 0；接 CI 的 --strict 会 exit 1`)
} else {
  console.log(`原语守卫 WARN：无新增 · 存量 截断 ${categories.truncate.count}（目标 ≤5）/ 原子写 ${categories.atomicWrite.count} / 前端旁路 ${categories.frontendBypass.count} / heredoc ${categories.heredoc.count}（基线 0）——warn 档 exit 0（未接 CI，接门禁前先拍板验收线）`)
}
process.exit(0)
