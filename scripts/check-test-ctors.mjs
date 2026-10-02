// 测试裸构造守卫（`&App{}` / nil 嵌入 panic 家族）——warn 档，可复跑，暂不接 CI。
//
// 立项理由（审计 P0#25 / 源 X1-03，仍在册）：
//   internal/app/app.go 的 App 是**五重嵌入指针**（*core / *writingState / *mediaState /
//   *whisperState / *officeState，且四个 state 各自再嵌一份 *core），唯一装配点是
//   app.go 的 New()；App 本身仍是「可裸构造的导出零值」。测试里 `&App{}` 这类裸字面量
//   不经过任何装配，方法一旦碰到未设置的嵌入指针就是 nil 解引用 panic——同一家族已
//   三次进 CHANGELOG（v4.406「App 字面量测试缺 mediaState 初始化——嵌入指针 nil…panic」、
//   v4.407「&mediaState{} 裸初始化嵌入 core 为 nil…」等），每次都是先炸再补。
//
// 本脚本（第一版，**不武断判错**）把命中拆成**两个正交维度**分别标注，不做"是/否"结论：
//   ① 分档（按字面量里显式设置了几个嵌入域状态）：
//        bare（一个都没设，最高危）> core-only（只设 core）> partial（设了 core+部分域）> full。
//   ② 是否在"构造助手"函数内（判据：所在顶层函数返回值里含 `*App`，如
//      taskInboxFixture/newTestAppWithProject 这类 fixture）。
//   关键：**两者不可互相抵消**——仓内实测 `internal/app/gaea_dag_risk_test.go:45`
//   `func a0() *App { return &App{} }` 就是"助手内 + bare"：它自己危险，且把所有
//   调用点一起拖下水（风险放大器），单看"在助手内"会把它漏掉。故本脚本把
//   「助手内 + bare」单列一行，优先级最高。warn 档只报不判错，分档只用于排序复核。
//   基线 = 首跑真实存量（逐项豁免）；--strict 只拦**新增**命中位置（新写的裸构造）。
//   基线认键 = `文件::函数名#该函数内序号`，**行号只作展示**——无关编辑造成的行号漂移不误报。
//
// 检测口径（简易 AST：注释/字符串剥离 + 花括号配平，不是完整 Go parser）：
//   逐行剥离 // 与 /* */ 注释、`"..."`/'...' 字面量（Go struct tag 是反引号，本脚本
//   不含 Go 源码解析，故一并中和）后，匹配 /&App[ \t]*\{/；
//   从 `{` 起按 `{}()[]` 配平收整个字面量，取 depth==1 的 `key:` 键名。
//   已知漏报面（见 README「口径与已知漏报」）：`app := App{}`（值构造，非指针）不收；
//   `&app.App{`（跨包）不收；同一行两个键只取能识别的；函数签名跨多行/参数里带
//   匿名函数类型的"是否助手"判定可能漏（本次仓内实测 0 例）；`//go:build` 变体正常扫。
//
// 用法：
//   node scripts/check-test-ctors.mjs                   默认 warn 档：计数表 + 新增项，exit 0
//   node scripts/check-test-ctors.mjs --list-all        连存量命中一起逐行列出（人工复核用）
//   node scripts/check-test-ctors.mjs --strict          有超出基线的新增命中则 exit 1（接 CI 时用）
//   node scripts/check-test-ctors.mjs --json            机读 JSON（可叠加 --strict / --list-all）
//   node scripts/check-test-ctors.mjs --write-baseline  用本次真实扫描结果覆写基线（人工评审后用）
//   node scripts/check-test-ctors.mjs --root <dir>      换扫描根（缺省 internal/app；隔离树负例用）
//   node scripts/check-test-ctors.mjs --app-path <file> 换 App 结构体定义文件（缺省 internal/app/app.go）
// 退出码：0=通过 / 1=--strict 且有新增命中 / 2=用法错、根缺失、基线缺失或损坏。
import fs from 'node:fs'
import path from 'node:path'

const BASELINE_PATH = 'docs/code-audit-2026-10-02/machine/testctors-baseline.json'
const DEF_ROOT = 'internal/app'
const DEF_APP = 'internal/app/app.go'
const AUDIT_DOC = 'docs/code-audit-2026-10-02/round-1-p0-status.md'

// app.go 读不到时的兜底嵌入字段名（隔离树负例里没有 app.go——此时仍能分档）
const FALLBACK_EMBEDDED = ['core', 'writingState', 'mediaState', 'whisperState', 'officeState']
const SKIP_DIRS = /(^|[\\/])(node_modules|clones|backups|dist|build|wailsjs|\.tmp|\.git)([\\/]|$)/

// ── 参数 ───────────────────────────────────────────────────────────────────
const argv = process.argv.slice(2)
const flags = new Set()
const opts = { root: DEF_ROOT, app: DEF_APP, baseline: BASELINE_PATH }
for (let i = 0; i < argv.length; i++) {
  const a = argv[i]
  if (a === '--root' || a === '--app-path' || a === '--baseline') {
    const v = argv[++i]
    if (v === undefined) { console.error(`参数 ${a} 缺值`); process.exit(2) }
    if (a === '--root') opts.root = v
    else if (a === '--app-path') opts.app = v
    else opts.baseline = v
    continue
  }
  if (!['--strict', '--json', '--write-baseline', '--list-all', '--help', '-h'].includes(a)) {
    console.error(`未知参数：${a}`)
    console.error('用法：node scripts/check-test-ctors.mjs [--strict] [--json] [--list-all] [--write-baseline] [--root dir] [--app-path file] [--baseline p]')
    process.exit(2)
  }
  flags.add(a)
}
if (flags.has('--help') || flags.has('-h')) {
  console.log('用法：node scripts/check-test-ctors.mjs [--strict] [--json] [--list-all] [--write-baseline] [--root dir] [--app-path file] [--baseline p]')
  process.exit(0)
}
const STRICT = flags.has('--strict')
const JSON_OUT = flags.has('--json')
const WRITE_BASELINE = flags.has('--write-baseline')
const LIST_ALL = flags.has('--list-all')

// ── 工具 ───────────────────────────────────────────────────────────────────
const posix = (p) => p.split(path.sep).join('/')
const readLines = (file) => fs.readFileSync(file, 'utf8').split(/\r?\n/)
const clip = (s, n = 120) => (s.length > n ? s.slice(0, n) + '…' : s)

const WIDE = /[\u1100-\u115F\u2E80-\uA4CF\uAC00-\uD7A3\uF900-\uFAFF\uFE30-\uFE4F\uFF00-\uFF60\uFFE0-\uFFE6]/
const dispWidth = (s) => { let w = 0; for (const ch of s) w += WIDE.test(ch) ? 2 : 1; return w }
const pad = (s, w) => s + ' '.repeat(Math.max(0, w - dispWidth(s)))
function renderTable(rows) {
  const widths = []
  for (const row of rows) row.forEach((c, i) => { widths[i] = Math.max(widths[i] ?? 0, dispWidth(String(c))) })
  return rows.map((row) => row.map((c, i) => pad(String(c), widths[i])).join('  ').replace(/\s+$/, '')).join('\n')
}

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
    return raw.slice(0, 12)
  } catch { return null }
}

function walk(dir, out = []) {
  if (!fs.existsSync(dir)) return out
  for (const e of fs.readdirSync(dir, { withFileTypes: true })) {
    const p = posix(path.join(dir, e.name))
    if (SKIP_DIRS.test(p)) continue
    if (e.isDirectory()) walk(p, out)
    else if (e.name.endsWith('_test.go')) out.push(p)
  }
  return out
}

// 键的显示路径：仓内文件用相对 cwd 的路径（可读），隔离树文件用相对扫描根的路径（稳定）
function displayPath(file, root) {
  const rel = path.relative(process.cwd(), path.resolve(file))
  if (rel && !rel.startsWith('..')) return posix(rel)
  return posix(path.relative(path.resolve(root), path.resolve(file)))
}

// 逐行剥离注释与字符串（字符串内容整体中和：`"&App{}"` 这类字面量不该算命中）
function stripGo(lines) {
  const out = []
  let inBlock = false
  for (const raw of lines) {
    let s = ''
    let i = 0
    while (i < raw.length) {
      if (inBlock) {
        const e = raw.indexOf('*/', i)
        if (e === -1) { i = raw.length; break }
        inBlock = false; i = e + 2; continue
      }
      const two = raw.slice(i, i + 2)
      if (two === '//') break
      if (two === '/*') { inBlock = true; i += 2; continue }
      const ch = raw[i]
      if (ch === '"' || ch === "'" || ch === '`') {
        let j = i + 1
        while (j < raw.length) {
          if (raw[j] === '\\' && ch !== '`') { j += 2; continue }
          if (raw[j] === ch) break
          j++
        }
        s += '""'
        i = j + 1
        continue
      }
      s += ch
      i++
    }
    out.push(s)
  }
  return out
}

// ── App 结构体的嵌入指针字段名（改 App 定义自动跟随，不写死）────────────────
function parseEmbedded(appPath) {
  if (!fs.existsSync(appPath)) return { names: [...FALLBACK_EMBEDDED], source: 'fallback（app-path 不存在）' }
  const lines = stripGo(readLines(appPath))
  let cur = false
  let depth = 0
  const names = []
  for (const text of lines) {
    if (!cur) {
      if (/^type\s+App\s+struct\s*\{/.test(text)) { cur = true; depth = 0 }
      else continue
    } else if (depth === 1) {
      const m = /^\s*\*([A-Za-z_]\w*)\s*$/.exec(text)
      if (m) names.push(m[1])
    }
    for (const ch of text) { if (ch === '{') depth++; else if (ch === '}') depth-- }
    if (cur && depth <= 0) break
  }
  // 只保留确实是嵌入指针的（*core 等小写类型）
  const filtered = names.filter((n) => /^[a-z]/.test(n))
  if (!filtered.length) return { names: [...FALLBACK_EMBEDDED], source: 'fallback（App 结构体解析为空）' }
  return { names: filtered, source: `${posix(appPath)} 解析` }
}

// ── 函数签名解析（是否"构造助手"：返回值含 *App）───────────────────────────
function skipParens(s, i) { // s[i] === '(' → 返回配对 ')' 之后的下标
  let d = 0
  for (; i < s.length; i++) {
    if (s[i] === '(') d++
    else if (s[i] === ')') { d--; if (d === 0) return i + 1 }
  }
  return -1
}
// 从第 i 行起收集函数头：直到"圆/方括号已配平且见到 '{'"——不能在第一个 '{' 就停，
// 因为参数里可能有 `struct{}`（仓内实测 newCreateChapterSuspendingApp 就栽在这）。
function collectHeader(lines, i) {
  let h = ''
  for (let k = i; k < Math.min(lines.length, i + 10); k++) {
    h += (k === i ? '' : ' ') + lines[k]
    let d = 0
    for (const ch of h) { if (ch === '(' || ch === '[') d++; else if (ch === ')' || ch === ']') d-- }
    if (h.includes('{') && d <= 0) break
  }
  return h
}
// 找函数体开括号：参数表之后的第一个"括号深度 0"的 '{'（参数里的 struct{} 不算）
function cutBody(s, from) {
  let d = 0
  for (let i = from; i < s.length; i++) {
    const c = s[i]
    if (c === '(' || c === '[') d++
    else if (c === ')' || c === ']') d--
    else if (c === '{' && d === 0) return s.slice(0, i)
  }
  return s
}
function parseFuncSig(header) {
  const m = /^func[ \t]+/.exec(header)
  if (!m) return null
  let i = m[0].length
  if (header[i] === '(') { i = skipParens(header, i); if (i < 0) return null }
  const nm = /^[ \t]*([A-Za-z_]\w*)/.exec(header.slice(i))
  if (!nm) return null
  i += nm[0].length
  if (header[i] !== '(') return null
  const close = skipParens(header, i)
  if (close < 0) return null
  const rest = cutBody(header, close + 1).slice(close + 1).trim()
  return { name: nm[1], results: rest, returnsApp: /\*App\b/.test(rest) }
}

// ── 字面量键名解析（depth==1 的 `key:`）───────────────────────────────────
function literalKeys(lines, startLine, startCol) {
  const keys = []
  let depth = 0
  for (let li = startLine; li < lines.length; li++) {
    const text = lines[li]
    for (let ci = li === startLine ? startCol : 0; ci < text.length; ci++) {
      const ch = text[ci]
      if (ch === '{' || ch === '(' || ch === '[') depth++
      else if (ch === '}' || ch === ')' || ch === ']') {
        depth--
        if (depth <= 0) return { keys, endLine: li }
      } else if (depth === 1) {
        const m = /^([A-Za-z_]\w*)[ \t]*:/.exec(text.slice(ci))
        if (m && (ci === 0 || /[^A-Za-z0-9_]/.test(text[ci - 1]))) { keys.push(m[1]); ci += m[0].length - 1 }
      }
    }
  }
  return { keys, endLine: lines.length - 1 }
}

function classify(keys, embedded) {
  const set = embedded.filter((n) => keys.includes(n))
  if (set.length === 0) return 'bare'
  if (set.length === embedded.length) return 'full'
  if (set.length === 1 && set[0] === 'core') return 'core-only'
  return 'partial'
}
const CLASS_LABEL = {
  bare: '高危 · 未设任何域状态',
  'core-only': '中危 · 只设 core，其余域状态 nil',
  partial: '中危 · 域状态不齐',
  full: '低危 · 域状态齐',
}
const CLASS_ORDER = ['bare', 'core-only', 'partial', 'full']

// ── 扫描 ───────────────────────────────────────────────────────────────────
if (!fs.existsSync(opts.root)) { console.error(`扫描根缺失：${opts.root}`); process.exit(2) }
const emb = parseEmbedded(opts.app)
const files = walk(opts.root).sort()
const hits = []
for (const file of files) {
  const raw = readLines(file)
  const lines = stripGo(raw)
  const funcAt = new Array(lines.length).fill(null)
  let cur = null
  for (let i = 0; i < lines.length; i++) {
    if (/^func[ \t]/.test(lines[i])) {
      const sig = parseFuncSig(collectHeader(lines, i))
      cur = sig ? { ...sig, line: i + 1 } : null
    }
    funcAt[i] = cur
  }
  const seqInFunc = new Map()
  for (let i = 0; i < lines.length; i++) {
    const re = /&App[ \t]*\{/g
    let m
    while ((m = re.exec(lines[i])) !== null) {
      const openCol = m.index + m[0].length - 1
      const { keys, endLine } = literalKeys(lines, i, openCol)
      const fn = funcAt[i]
      const fnName = fn ? fn.name : '(顶层/非函数内)'
      const inHelper = Boolean(fn && fn.returnsApp)
      const n = (seqInFunc.get(fnName) ?? 0) + 1
      seqInFunc.set(fnName, n)
      // 字面量之后的补充装配（`a := &App{}` 紧接 `a.writingState = ...`）必须看见，
      // 否则会把"分两步装配"的用例误报成 bare——本仓有实例（newCreateChapterSuspendingApp）。
      const varName = (/([A-Za-z_]\w*)\s*:?=[^=]*&App[ \t]*\{[^]*$/.exec(lines[i]) ?? [])[1] ?? null
      const postAssigned = []
      if (varName) {
        for (let k = endLine + 1; k < lines.length && funcAt[k] === fn; k++) {
          const am = new RegExp(`\\b${varName}\\.([A-Za-z_]\\w*)\\s*=`).exec(lines[k])
          if (am && emb.names.includes(am[1]) && !postAssigned.includes(am[1])) postAssigned.push(am[1])
        }
      }
      const allKeys = [...keys, ...postAssigned]
      const cls = classify(allKeys, emb.names)
      const literalOnly = classify(keys, emb.names)
      hits.push({
        file: displayPath(file, opts.root),
        line: i + 1,
        col: openCol + 1,
        func: fnName,
        seq: n,
        key: `${displayPath(file, opts.root)}::${fnName}#${n}`,
        inHelper,
        helperAmplifier: inHelper && cls === 'bare', // 助手内 + bare = 风险放大器
        class: cls,
        classLiteralOnly: literalOnly,
        postAssigned,
        rescuedByPostAssign: literalOnly !== cls, // 只看字面量会误报，补装配后降档
        embeddedSet: embeddedSet(allKeys, emb.names),
        embeddedMissing: emb.names.filter((x) => !allKeys.includes(x)),
        text: clip(raw[i].trim()),
      })
    }
  }
}
function embeddedSet(keys, embedded) { return embedded.filter((n) => keys.includes(n)) }

const byClass = Object.fromEntries(CLASS_ORDER.map((c) => [c, hits.filter((h) => h.class === c).length]))
const byClassLiteralOnly = Object.fromEntries(CLASS_ORDER.map((c) => [c, hits.filter((h) => h.classLiteralOnly === c).length]))
const inHelperCount = hits.filter((h) => h.inHelper).length
const amplifierCount = hits.filter((h) => h.helperAmplifier).length
const rescuedCount = hits.filter((h) => h.rescuedByPostAssign).length

// 参考行：非测试文件里的 `&App{`（含唯一装配点 app.go 的 a := &App{core: c}）不入基线
let prodRef = 0
for (const e of fs.existsSync(opts.root) ? fs.readdirSync(opts.root, { withFileTypes: true }) : []) {
  if (!e.isFile() || !e.name.endsWith('.go') || e.name.endsWith('_test.go')) continue
  const lines = stripGo(readLines(posix(path.join(opts.root, e.name))))
  for (const l of lines) prodRef += (l.match(/&App[ \t]*\{/g) ?? []).length
}

const scanMeta = {
  root: posix(opts.root),
  testFiles: files.length,
  appPath: posix(opts.app),
  embedded: emb.names,
  embeddedSource: emb.source,
  totalHits: hits.length,
  prodRefHits: prodRef,
}
const helperNames = [...new Set(hits.filter((h) => h.inHelper).map((h) => h.func))].sort()

// ── --write-baseline ───────────────────────────────────────────────────────
if (WRITE_BASELINE) {
  const baseline = {
    schema: 'gaea-test-ctors-baseline/1',
    generatedAt: new Date().toISOString(),
    generatedBy: 'node scripts/check-test-ctors.mjs --write-baseline',
    head: gitHead(),
    audit: { doc: AUDIT_DOC, finding: 'P0#25 / 源 X1-03（三重以上 *core 嵌入 → nil 嵌入 panic 三犯）' },
    inputs: scanMeta,
    classCounts: byClass,
    classCountsLiteralOnly: byClassLiteralOnly,
    inHelperCount,
    amplifierCount,
    rescuedByPostAssign: rescuedCount,
    constructorHelpers: helperNames,
    // 首跑真实存量：逐项豁免。**豁免 ≠ 已确认安全**，只是"当前存量"；
    // bare/core-only 与 amplifier（助手内+bare）是人工复核队列，收口后逐条删掉再看 --strict。
    exempt: hits.map((h) => ({ key: h.key, file: h.file, line: h.line, func: h.func, class: h.class, classLiteralOnly: h.classLiteralOnly, postAssigned: h.postAssigned, inHelper: h.inHelper, helperAmplifier: h.helperAmplifier, embeddedSet: h.embeddedSet, embeddedMissing: h.embeddedMissing, text: h.text })),
  }
  try {
    fs.mkdirSync(path.dirname(opts.baseline), { recursive: true })
    fs.writeFileSync(opts.baseline, JSON.stringify(baseline, null, 2) + '\n', 'utf8')
  } catch (e) {
    console.error(`基线写入失败：${opts.baseline} — ${e.message}`)
    process.exit(2)
  }
  console.log(`基线已写入 ${opts.baseline}`)
  console.log(`  扫描 ${scanMeta.root} 下 ${files.length} 个 *_test.go，裸 &App{ 命中 ${hits.length} 处`)
  console.log(`  分档（含字面量后补装配）：${CLASS_ORDER.map((c) => `${c} ${byClass[c]}`).join(' / ')}`)
  console.log(`  只看字面量的口径：${CLASS_ORDER.map((c) => `${c} ${byClassLiteralOnly[c]}`).join(' / ')}（补装配降档 ${rescuedCount} 处）`)
  console.log(`  助手内 ${inHelperCount} 处（其中 助手内+bare 风险放大器 ${amplifierCount} 处）· 构造助手 ${helperNames.length} 个`)
  console.log(`  参考：非测试 .go 里 &App{ ${prodRef} 处（含唯一装配点，不入基线）`)
  process.exit(0)
}

// ── 与基线对比 ─────────────────────────────────────────────────────────────
let baseline = null
if (fs.existsSync(opts.baseline)) {
  try {
    baseline = JSON.parse(fs.readFileSync(opts.baseline, 'utf8'))
    if (baseline.schema !== 'gaea-test-ctors-baseline/1') throw new Error(`schema 不匹配：${baseline.schema}`)
  } catch (e) {
    console.error(`基线无法解析：${opts.baseline} — ${e.message}`)
    process.exit(2)
  }
} else {
  console.error(`基线缺失：${opts.baseline}`)
  console.error('先跑一次 node scripts/check-test-ctors.mjs --write-baseline（人工评审基线后再入库）')
  process.exit(2)
}

const baseKeys = new Set((baseline.exempt ?? []).map((i) => i.key))
const added = hits.filter((h) => !baseKeys.has(h.key))
// 扫描根与基线不一致（--root 隔离树负例）时不做"存量消失"对比，否则会把整套存量报成 gone
const sameRoot = posix(opts.root) === posix(baseline.inputs?.root ?? DEF_ROOT)
const gone = sameRoot ? (baseline.exempt ?? []).filter((i) => !hits.some((h) => h.key === i.key)) : []
const exitCode = STRICT && added.length > 0 ? 1 : 0

// ── --json ─────────────────────────────────────────────────────────────────
if (JSON_OUT) {
  const payload = {
    tool: 'check-test-ctors',
    schema: 'gaea-test-ctors-guard/1',
    mode: STRICT ? 'strict' : 'warn',
    ranAt: new Date().toISOString(),
    head: gitHead(),
    auditDoc: AUDIT_DOC,
    scan: scanMeta,
    baseline: { path: posix(opts.baseline), generatedAt: baseline.generatedAt ?? null, head: baseline.head ?? null, exempt: baseKeys.size },
    classLabels: CLASS_LABEL,
    counts: { total: hits.length, byClass, byClassLiteralOnly, inHelper: inHelperCount, helperAmplifier: amplifierCount, rescuedByPostAssign: rescuedCount, baseline: baseKeys.size },
    constructorHelpers: helperNames,
    added: { total: added.length, items: added },
    gone: { total: gone.length, items: gone },
    hits,
    exitCode,
  }
  console.log(JSON.stringify(payload, null, 2))
  process.exit(exitCode)
}

// ── 人读输出 ───────────────────────────────────────────────────────────────
console.log(`测试裸构造守卫（&App{} / nil 嵌入 panic 家族 · ${STRICT ? 'strict' : 'warn'} 档）· 审计 P0#25`)
console.log(`扫描：${scanMeta.root}/**/*_test.go ${files.length} 个文件 · 命中裸 &App{ ${hits.length} 处 · 参考：非测试 .go ${prodRef} 处（含唯一装配点，不入基线）`)
console.log(`嵌入字段口径：${emb.names.map((n) => '*' + n).join(' ')}（来源：${emb.source}）`)
console.log(`基线：${posix(opts.baseline)}（${baseline.generatedAt ?? '无日期'}${baseline.head ? ` @ ${baseline.head}` : ''}）· 豁免存量 ${baseKeys.size} 处`)
console.log('')
console.log(renderTable([
  ['分档（字面量 + 紧随其后的 var.field 补装配）', '当前', '基线', '新增', '含义'],
  ...CLASS_ORDER.map((c) => [c, byClass[c], (baseline.classCounts ?? {})[c] ?? 0, `+${added.filter((h) => h.class === c).length}`, CLASS_LABEL[c]]),
  ['合计', hits.length, baseKeys.size, `+${added.length}`, '—'],
  ['【另一维度】助手内（函数返回 *App）', inHelperCount, baseline.inHelperCount ?? 0, `+${added.filter((h) => h.inHelper).length}`, '助手内=预期构造点，但**不自动安全**'],
  ['【另一维度】助手内 + bare（放大器）', amplifierCount, baseline.amplifierCount ?? 0, `+${added.filter((h) => h.helperAmplifier).length}`, '自身危险且拖全调用点下水，最先看'],
  ['【口径对照】只看字面量会多报', rescuedCount, baseline.rescuedByPostAssign ?? 0, `+${added.filter((h) => h.rescuedByPostAssign).length}`, '`a := &App{}` 后接 `a.writingState = …`，已计入'],
]))
console.log(`构造助手（返回值含 *App）：${helperNames.length} 个（${helperNames.slice(0, 6).join(', ')}${helperNames.length > 6 ? ' …' : ''}）`)
if (!sameRoot) console.log(`注：扫描根 ${scanMeta.root} 与基线根 ${baseline.inputs?.root ?? DEF_ROOT} 不同（隔离树模式），已跳过"存量消失"对比`)

if (added.length) {
  console.log('')
  console.log(`  NEW   新增裸构造 ${added.length} 处（未在基线内，须人工确认是否需要装配）：`)
  for (const h of [...added].sort((a, b) => CLASS_ORDER.indexOf(a.class) - CLASS_ORDER.indexOf(b.class) || b.helperAmplifier - a.helperAmplifier)) {
    console.log(`        ${h.file}:${h.line}  [${h.class}]${h.helperAmplifier ? '[助手内·放大器]' : h.inHelper ? '[助手内]' : ''}  函数 ${h.func}()  已设 ${h.embeddedSet.join(',') || '（无）'} / 缺 ${h.embeddedMissing.join(',') || '（无）'}${h.postAssigned.length ? `  （其中补装配：${h.postAssigned.join(',')}）` : ''}`)
    console.log(`            ${h.text}`)
  }
} else {
  console.log(`  OK    无新增裸构造（存量 ${hits.length} 处全在基线内${LIST_ALL ? '' : '，不逐一列出；加 --list-all 全列'}）`)
}
if (gone.length) {
  console.log(`  NOTE  基线内 ${gone.length} 处命中已消失（收敛有进展，确认后跑 --write-baseline 更新基线）：`)
  for (const i of gone) console.log(`        ${i.file}  [${i.class}]  函数 ${i.func}()`)
}

if (LIST_ALL) {
  console.log('')
  console.log(`  存量全部命中（${hits.length} 处，按分档排序、放大器优先——人工复核队列）：`)
  const sorted = [...hits].sort((a, b) => {
    const d = CLASS_ORDER.indexOf(a.class) - CLASS_ORDER.indexOf(b.class)
    if (d) return d
    if (a.helperAmplifier !== b.helperAmplifier) return a.helperAmplifier ? -1 : 1
    return a.file < b.file ? -1 : a.file > b.file ? 1 : a.line - b.line
  })
  for (const h of sorted) {
    console.log(`        ${h.file}:${h.line}  [${h.class}]${h.helperAmplifier ? '[助手内·放大器]' : h.inHelper ? '[助手内]' : ''}  ${h.func}()  已设 ${h.embeddedSet.join(',') || '（无）'} / 缺 ${h.embeddedMissing.join(',') || '（无）'}${h.postAssigned.length ? `  （其中补装配：${h.postAssigned.join(',')}）` : ''}`)
    console.log(`            ${h.text}`)
  }
}

console.log('')
if (STRICT) {
  if (added.length) {
    console.log(`测试裸构造守卫 STRICT FAIL：新增 ${added.length} 处裸 &App{（见上）——接 CI 后此处 exit 1`)
    process.exit(1)
  }
  console.log(`测试裸构造守卫 STRICT OK：无新增（存量 ${hits.length} 处全在基线内；待人工收口：bare ${byClass.bare} / core-only ${byClass['core-only']} / 放大器 ${amplifierCount}）`)
  process.exit(0)
}
if (added.length) {
  console.log(`测试裸构造守卫 WARN：新增 ${added.length} 处（见上）——warn 档不 gate，exit 0；接 CI 的 --strict 会 exit 1`)
} else {
  console.log(`测试裸构造守卫 WARN：无新增 · 存量 ${hits.length} 处（bare ${byClass.bare} / core-only ${byClass['core-only']} / partial ${byClass.partial} / full ${byClass.full}；助手内 ${inHelperCount}，其中放大器 ${amplifierCount}；补装配降档 ${rescuedCount}）——warn 档 exit 0（未接 CI，收口前先拍板验收线）`)
}
process.exit(0)
