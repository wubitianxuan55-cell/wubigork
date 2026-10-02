// 跨语言契约漂移守卫（进度计划域）——warn 档，可复跑，暂不接 CI。
//
// 立项理由（第一轮修 internal/schedule 的教训，审计 FE7-01/FE7-02 同族）：
//   Go 侧持久化是「Unmarshal 进结构体 → MarshalIndent 整量覆盖写盘」，于是
//   「前端 types.ts 有字段、Go 结构体没有同名 json tag」= 每次保存静默丢掉该字段。
//   第一轮靠人肉写了三个往返用例才挖出三处：
//     ① SchedProject.baselines  → Project.Baselines（多基线槽位整表被保存链路擦掉）
//     ② SchedResource.calendar  → Resource.Calendar（资源级日历被擦掉）
//     ③ SchedAssignment 的 units/quantity/amount 数值三态（Go 侧必须是 *float64，
//        裸 float64+omitempty 会把「显式 0」与「未设置」合流）
//   三处都不能靠"下次注意"，必须机械闸。本脚本就是这道闸的第一版（warn 档）。
//
// 检测口径（两段都是"简易 AST"，即花括号/圆括号配平扫描，不是完整 TS/Go parser）：
//   TS 侧 frontend/src/schedule/types.ts（缺省；--types-path 可换，用于隔离树负例）
//     逐行剥离 // 行注释、/* */ 块注释、'..'/".."/`..` 字符串字面量后按行扫；
//     命中 `export interface NAME {`（也接受裸 `interface` 与 `export type NAME = {`）
//     即进入该接口体（花括号配平，depth=1 的成员才算本接口的字段），
//     取 depth==1 且形如 `name?: type` 的成员名（index signature `[k: string]` 不计）。
//     已知漏报面（见 README「口径与已知漏报」）：同一行写两个成员只取第一个；
//     `interface A extends B` 的继承字段不算 A 的字段（本文件无此写法）；
//     成员名跨行书写、泛型默认值里带 `:` 的表达式类型、`declare module` 内接口不收。
//   Go 侧 internal/schedule/types.go + internal/schedule/baseline.go（缺省；
//     --go-path 可换/可逗号分隔多个，用于隔离树负例）
//     命中 `^type NAME struct {` 后花括号配平收结构体体，
//     逐行取 `Field Type \`json:"tag"\`` 的 tag（`json:"-"` 忽略）；
//     已知漏报面：一行两个字段只取第一个；匿名嵌入字段（无 tag）不计；
//     字段名与类型跨行书写、`//go:build` 变体不收。
//
// 检查规则：
//   ① 包含性：TS 接口字段在本接口对应的 Go 结构体 json tag 集合里找不到同名 tag
//      → 报「前端有、Go 缺」（**gate**：这是静默丢失的那一类）。
//   ② 无对应结构体：TS 接口在 Go 侧找不到结构体（显式别名表 → 去 `Sched` 前缀
//      自动回落）→ 报 MISSING_STRUCT（**gate**，除非在基线 exemptInterfaces 里）。
//   ③ 反向（Go 有、TS 无）→ 只提示 HINT，**永不 gate**（可能是有意为之的服务端字段）。
//   ④ 数值三态提示：TS 侧可选数值字段（`f?: number`）在 Go 侧是裸数值 + omitempty
//      → HINT（对应缺陷③的残留面；**永不 gate**，防把「按需再改」变成挡门砖）。
//
// 认键规则：契约项以「TS 接口名 + 字段名」认键（`Interface.field`），行号只作展示——
// 无关编辑造成的行号漂移不会误报。基线只存豁免项，不存全量字段快照
// （契约是"必须一一对应"的，全量快照会让每次新增字段都要重写基线）。
//
// 用法：
//   node scripts/check-contract-drift.mjs                  默认 warn 档：打印计数/漂移/提示，exit 0
//   node scripts/check-contract-drift.mjs --strict         有新增漂移项则 exit 1（接 CI 时用）
//   node scripts/check-contract-drift.mjs --json           机读 JSON（可叠加 --strict）
//   node scripts/check-contract-drift.mjs --write-baseline 用本次真实扫描结果覆写基线（人工评审后用）
//   node scripts/check-contract-drift.mjs --types-path <p> --go-path <p[,p]>   换输入（隔离树负例用）
// 退出码：0=通过 / 1=--strict 且有新增漂移 / 2=用法错、输入缺失、基线缺失或损坏。
import fs from 'node:fs'
import path from 'node:path'

const BASELINE_PATH = 'docs/code-audit-2026-10-02/machine/contract-baseline.json'
const DEF_TYPES = 'frontend/src/schedule/types.ts'
// Go 侧口径含 baseline.go：Drift/DriftRow 不在 types.go 里，但同属该域的落盘/回传契约。
const DEF_GO = ['internal/schedule/types.go', 'internal/schedule/baseline.go']
const AUDIT_DOC = 'docs/code-audit-2026-10-02/round-1-p0-status.md'

// 契约配对别名：TS 接口名 → Go 结构体名。未列出的名字先按本表查，
// 查不到再尝试去掉 `Sched` 前缀（覆盖 SchedProject→Project 等 8 对）。
const ALIAS = {
  AoaPin: 'AoaPt', // 前端按"锚点"命名，Go 按坐标点命名
  BaselineDrift: 'Drift',
  BaselineDriftRow: 'DriftRow',
}
const AUTO_PREFIX = ['Sched', 'Gaea'] // 自动回落前缀（去前缀后命中 Go 结构体则配对）

// 历史三处缺陷（写进输出做"回归回执"：这三项必须在场，否则说明第一轮修复被回退）
const HISTORIC = [
  { iface: 'SchedProject', field: 'baselines', note: 'FE7-01 多基线槽位' },
  { iface: 'SchedResource', field: 'calendar', note: 'FE7-01 资源级日历' },
  { iface: 'SchedAssignment', field: 'units', note: 'FE7-02 数值三态' },
  { iface: 'SchedAssignment', field: 'quantity', note: 'FE7-02 数值三态' },
  { iface: 'SchedAssignment', field: 'amount', note: 'FE7-02 数值三态' },
]

// ── 参数 ───────────────────────────────────────────────────────────────────
const argv = process.argv.slice(2)
const flags = new Set()
const opts = { types: DEF_TYPES, go: [...DEF_GO], baseline: BASELINE_PATH }
for (let i = 0; i < argv.length; i++) {
  const a = argv[i]
  if (a === '--types-path' || a === '--go-path' || a === '--baseline') {
    const v = argv[++i]
    if (v === undefined) { console.error(`参数 ${a} 缺值`); process.exit(2) }
    if (a === '--types-path') opts.types = v
    else if (a === '--go-path') opts.go = v.split(',').map((s) => s.trim()).filter(Boolean)
    else opts.baseline = v
    continue
  }
  if (!['--strict', '--json', '--write-baseline', '--help', '-h'].includes(a)) {
    console.error(`未知参数：${a}`)
    console.error('用法：node scripts/check-contract-drift.mjs [--strict] [--json] [--write-baseline] [--types-path p] [--go-path p[,p]] [--baseline p]')
    process.exit(2)
  }
  flags.add(a)
}
if (flags.has('--help') || flags.has('-h')) {
  console.log('用法：node scripts/check-contract-drift.mjs [--strict] [--json] [--write-baseline] [--types-path p] [--go-path p[,p]] [--baseline p]')
  process.exit(0)
}
const STRICT = flags.has('--strict')
const JSON_OUT = flags.has('--json')
const WRITE_BASELINE = flags.has('--write-baseline')

// ── 工具 ───────────────────────────────────────────────────────────────────
const posix = (p) => p.split(path.sep).join('/')
const readLines = (file) => fs.readFileSync(file, 'utf8').split(/\r?\n/)
const clip = (s, n = 120) => (s.length > n ? s.slice(0, n) + '…' : s)

// 显示宽度对齐（CJK 记 2 列），风格对齐 scripts/check-primitives.mjs
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

// 逐行剥离注释与字符串字面量（跨行块注释/模板串状态保留在 state 里）。
// neutralize 掉的字符串内容不再参与括号配平与字段名匹配，避免 "{" / "a: b" 误伤。
function stripTs(lines) {
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
      if (two === '//') { break }
      if (two === '/*') { inBlock = true; i += 2; continue }
      const ch = raw[i]
      if (ch === '"' || ch === "'" || ch === '`') {
        let j = i + 1
        while (j < raw.length) {
          if (raw[j] === '\\') { j += 2; continue }
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

// Go 侧只剥注释：struct tag 是反引号原始字符串，必须保留（否则 tag 全丢）。
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
      if (raw[i] === '"') { // 普通字符串：整体保留但按字面量跳过转义
        let j = i + 1
        while (j < raw.length && raw[j] !== '"') { if (raw[j] === '\\') j++; j++ }
        s += raw.slice(i, j + 1); i = j + 1; continue
      }
      s += raw[i]; i++
    }
    out.push(s)
  }
  return out
}

// ── TS 解析：接口 → 字段表 ─────────────────────────────────────────────────
const RE_IFACE = /^\s*(?:export\s+)?(?:interface|type)\s+([A-Za-z_$][\w$]*)[^\n{]*\{/
const RE_MEMBER = /^\s*([A-Za-z_$][\w$]*)\s*(\??)\s*:\s*(.*)$/

function parseTsInterfaces(file) {
  const lines = stripTs(readLines(file))
  const ifaces = new Map()
  let cur = null
  let depth = 0
  for (let i = 0; i < lines.length; i++) {
    const text = lines[i]
    if (!cur) {
      const m = RE_IFACE.exec(text)
      if (!m) continue
      depth = 0
      for (const ch of text.slice(m[0].length - 1)) { if (ch === '{') depth++; else if (ch === '}') depth-- }
      cur = { name: m[1], line: i + 1, fields: [] }
      if (depth <= 0) { ifaces.set(cur.name, cur); cur = null }
      continue
    }
    if (depth === 1) {
      const m = RE_MEMBER.exec(text)
      // index signature `[k: string]: T` 与注释残渣不进字段表
      if (m && !/^\s*\[/.test(text)) cur.fields.push({ name: m[1], optional: m[2] === '?', type: clip(m[3].replace(/[;,]\s*$/, ''), 60), line: i + 1 })
    }
    for (const ch of text) { if (ch === '{') depth++; else if (ch === '}') depth-- }
    if (depth <= 0) { ifaces.set(cur.name, cur); cur = null }
  }
  return ifaces
}

// ── Go 解析：结构体 → json tag 集合 ────────────────────────────────────────
const RE_STRUCT = /^type\s+([A-Za-z_]\w*)\s+struct\s*\{/
const RE_GOFIELD = /^\s*([A-Z]\w*)\s+(\*?[\[\]\w.{}*]+)\s+`([^`]*)`/

function parseGoStructs(files) {
  const structs = new Map()
  for (const file of files) {
    const lines = stripGo(readLines(file))
    let cur = null
    let depth = 0
    for (let i = 0; i < lines.length; i++) {
      const text = lines[i]
      if (!cur) {
        const m = RE_STRUCT.exec(text)
        if (!m) continue
        cur = { name: m[1], file, line: i + 1, tags: new Map(), fields: new Map() }
        depth = 0
      } else if (depth === 1) {
        const m = RE_GOFIELD.exec(text)
        if (m) {
          const tag = /json:"([^"]*)"/.exec(m[3])
          if (tag) {
            const name = tag[1].split(',')[0]
            if (name && name !== '-') {
              cur.tags.set(name, { goField: m[1], goType: m[2], line: i + 1, opt: /,omitempty/.test(tag[1]) })
              cur.fields.set(m[1], { tag: name, goType: m[2], line: i + 1, opt: /,omitempty/.test(tag[1]) })
            }
          }
        }
      }
      for (const ch of text) { if (ch === '{') depth++; else if (ch === '}') depth-- }
      if (cur && depth <= 0) { structs.set(cur.name, cur); cur = null }
    }
  }
  return structs
}

// ── 汇合扫描 ───────────────────────────────────────────────────────────────
if (!fs.existsSync(opts.types)) { console.error(`TS 输入缺失：${opts.types}`); process.exit(2) }
for (const g of opts.go) if (!fs.existsSync(g)) { console.error(`Go 输入缺失：${g}`); process.exit(2) }

const ifaces = parseTsInterfaces(opts.types)
const structs = parseGoStructs(opts.go)

const resolveGo = (name) => {
  if (ALIAS[name]) return { go: ALIAS[name], how: 'alias' }
  if (structs.has(name)) return { go: name, how: 'same' }
  for (const pre of AUTO_PREFIX) {
    if (name.startsWith(pre) && name.length > pre.length) {
      const cand = name.slice(pre.length)
      if (structs.has(cand)) return { go: cand, how: `auto-${pre}` }
    }
  }
  return { go: null, how: null }
}

const missingStruct = [] // ② 无 Go 结构体
const missingField = []  // ① 前端有、Go 缺
const reverseOnly = []   // ③ Go 有、TS 无（提示）
const triState = []      // ④ 数值三态提示
const pairs = []
let fieldsTotal = 0

const sortedIfaces = [...ifaces.values()].sort((a, b) => (a.name < b.name ? -1 : 1))
for (const it of sortedIfaces) {
  fieldsTotal += it.fields.length
  const r = resolveGo(it.name)
  if (!r.go) {
    missingStruct.push({ key: it.name, iface: it.name, line: it.line, fields: it.fields.length, sample: it.fields.slice(0, 3).map((f) => f.name).join(', ') })
    continue
  }
  const st = structs.get(r.go)
  pairs.push({ iface: it.name, go: r.go, how: r.how, tsFields: it.fields.length, goTags: st.tags.size, line: it.line, goLine: st.line })
  const tsNames = new Set(it.fields.map((f) => f.name))
  for (const f of it.fields) {
    if (!st.tags.has(f.name)) {
      missingField.push({ key: `${it.name}.${f.name}`, iface: it.name, go: r.go, field: f.name, optional: f.optional, type: f.type, line: f.line, text: `${f.name}${f.optional ? '?' : ''}: ${f.type}` })
      continue
    }
    // ④ 可选数值 + Go 裸值 + omitempty → 三态风险提示（不 gate）
    const g = st.tags.get(f.name)
    if (f.optional && /^number$/.test(f.type.trim()) && g.opt && !g.goType.startsWith('*')) {
      triState.push({ key: `${it.name}.${f.name}`, iface: it.name, go: r.go, ts: `number?`, goType: g.goType, goField: g.goField, line: g.line, note: 'Go 裸数值+omitempty：显式 0 与未设置合流（要三态需 *float64/*int）' })
    }
  }
  for (const [tag, g] of st.tags) {
    if (!tsNames.has(tag)) reverseOnly.push({ key: `${it.name}.${tag}`, iface: it.name, go: r.go, tag, goField: g.goField, goType: g.goType, line: g.line })
  }
}

// 历史三处缺陷在场回执
const allTsKeys = new Set()
for (const it of sortedIfaces) for (const f of it.fields) allTsKeys.add(`${it.name}.${f.name}`)
const allGoTags = new Set()
for (const it of sortedIfaces) {
  const r = resolveGo(it.name)
  if (r.go) for (const tag of structs.get(r.go).tags.keys()) allGoTags.add(`${it.name}.${tag}`)
}
const historicStatus = HISTORIC.map((h) => ({
  ...h,
  present: allTsKeys.has(`${h.iface}.${h.field}`) && allGoTags.has(`${h.iface}.${h.field}`),
  tsOnly: allTsKeys.has(`${h.iface}.${h.field}`) && !allGoTags.has(`${h.iface}.${h.field}`),
}))

const scanMeta = {
  typesPath: posix(opts.types),
  goPaths: opts.go.map(posix),
  interfacesParsed: ifaces.size,
  interfacesPaired: pairs.length,
  fieldsParsed: fieldsTotal,
  goStructsParsed: structs.size,
  goTagsParsed: [...pairs].reduce((n, p) => n + p.goTags, 0),
}

const gateItems = [
  ...missingStruct.map((i) => ({ ...i, kind: 'missing-struct' })),
  ...missingField.map((i) => ({ ...i, kind: 'missing-field' })),
].sort((a, b) => (a.key < b.key ? -1 : 1))

// ── --write-baseline ───────────────────────────────────────────────────────
if (WRITE_BASELINE) {
  const baseline = {
    schema: 'gaea-contract-baseline/1',
    generatedAt: new Date().toISOString(),
    generatedBy: 'node scripts/check-contract-drift.mjs --write-baseline',
    head: gitHead(),
    audit: { doc: AUDIT_DOC, family: 'FE7-01 / FE7-02 / P0#14 / P0#15（进度计划域跨语言契约漂移）' },
    inputs: scanMeta,
    aliases: { explicit: ALIAS, autoPrefixStrip: AUTO_PREFIX },
    // 豁免项：首跑真实结果（禁手写估值）。gate 项为空 = 当前契约零漂移。
    exemptInterfaces: missingStruct.map((i) => ({ key: i.key, line: i.line, fields: i.fields, sample: i.sample, reason: '本域无 Go 对应结构体（前端本地计算/展示类型；若将来 Go 侧要落盘必须删此豁免）' })),
    exemptFields: missingField.map((i) => ({ key: i.key, line: i.line, text: i.text, reason: '首跑即缺口，待评估' })),
    triStateHints: triState.map((i) => ({ key: i.key, goType: i.goType, note: i.note })),
    reverseOnlyCount: reverseOnly.length,
  }
  try {
    fs.mkdirSync(path.dirname(opts.baseline), { recursive: true })
    fs.writeFileSync(opts.baseline, JSON.stringify(baseline, null, 2) + '\n', 'utf8')
  } catch (e) {
    console.error(`基线写入失败：${opts.baseline} — ${e.message}`)
    process.exit(2)
  }
  console.log(`基线已写入 ${opts.baseline}`)
  console.log(`  解析：TS 接口 ${ifaces.size} 个 / 字段 ${fieldsTotal} 个（配对 ${pairs.length} 个接口）· Go 结构体 ${structs.size} 个`)
  console.log(`  豁免：整接口 ${baseline.exemptInterfaces.length} 个 / 单字段 ${baseline.exemptFields.length} 个 · 三态提示 ${triState.length} 条（不 gate）`)
  console.log(`  gate 漂移：无结构体 ${missingStruct.length} + 缺字段 ${missingField.length}（豁免后应为 0 才合规）`)
  process.exit(0)
}

// ── 与基线对比 ─────────────────────────────────────────────────────────────
let baseline = null
if (fs.existsSync(opts.baseline)) {
  try {
    baseline = JSON.parse(fs.readFileSync(opts.baseline, 'utf8'))
    if (baseline.schema !== 'gaea-contract-baseline/1') throw new Error(`schema 不匹配：${baseline.schema}`)
  } catch (e) {
    console.error(`基线无法解析：${opts.baseline} — ${e.message}`)
    process.exit(2)
  }
} else {
  console.error(`基线缺失：${opts.baseline}`)
  console.error('先跑一次 node scripts/check-contract-drift.mjs --write-baseline（人工评审基线后再入库）')
  process.exit(2)
}

const baseIfaces = new Set((baseline.exemptInterfaces ?? []).map((i) => i.key))
const baseFields = new Set((baseline.exemptFields ?? []).map((i) => i.key))
const addedItems = gateItems.filter((i) => (i.kind === 'missing-struct' ? !baseIfaces.has(i.key) : !baseFields.has(i.key)))
const exempted = gateItems.filter((i) => !addedItems.includes(i))
const exitCode = STRICT && addedItems.length > 0 ? 1 : 0

// ── --json ─────────────────────────────────────────────────────────────────
if (JSON_OUT) {
  const payload = {
    tool: 'check-contract-drift',
    schema: 'gaea-contract-guard/1',
    mode: STRICT ? 'strict' : 'warn',
    ranAt: new Date().toISOString(),
    head: gitHead(),
    auditDoc: AUDIT_DOC,
    scan: scanMeta,
    aliases: { explicit: ALIAS, autoPrefixStrip: AUTO_PREFIX },
    baseline: { path: posix(opts.baseline), generatedAt: baseline.generatedAt ?? null, head: baseline.head ?? null, exemptInterfaces: baseIfaces.size, exemptFields: baseFields.size },
    counts: { interfaces: ifaces.size, fields: fieldsTotal, paired: pairs.length, missingStruct: missingStruct.length, missingField: missingField.length, reverseOnly: reverseOnly.length, triStateHints: triState.length },
    historic: historicStatus,
    added: { total: addedItems.length, items: addedItems },
    exempted: { total: exempted.length, items: exempted },
    missingStruct, missingField, reverseOnly, triStateHints: triState,
    exitCode,
  }
  console.log(JSON.stringify(payload, null, 2))
  process.exit(exitCode)
}

// ── 人读输出 ───────────────────────────────────────────────────────────────
console.log(`跨语言契约漂移守卫（进度计划域 · ${STRICT ? 'strict' : 'warn'} 档）`)
console.log(`解析：TS ${scanMeta.typesPath} → 接口 ${ifaces.size} 个 / 字段 ${fieldsTotal} 个；Go ${scanMeta.goPaths.join(' + ')} → 结构体 ${structs.size} 个`)
console.log(`      配对 ${pairs.length} 个接口（显式别名 ${Object.keys(ALIAS).length} 对 + 自动去前缀 ${AUTO_PREFIX.join('/')}）· 口径=花括号配平简易 AST（非完整 parser）`)
console.log(`基线：${posix(opts.baseline)}（${baseline.generatedAt ?? '无日期'}${baseline.head ? ` @ ${baseline.head}` : ''}）· 豁免 整接口 ${baseIfaces.size} / 单字段 ${baseFields.size}`)
console.log('')
console.log(renderTable([
  ['检测项', '当前', '基线豁免', '新增', '是否 gate'],
  ['① 前端有、Go 缺（字段级）', missingField.length, baseFields.size, `+${addedItems.filter((i) => i.kind === 'missing-field').length}`, '是'],
  ['② TS 接口无 Go 结构体', missingStruct.length, baseIfaces.size, `+${addedItems.filter((i) => i.kind === 'missing-struct').length}`, '是'],
  ['③ Go 有、TS 无（反向）', reverseOnly.length, '—', '—', '否（只提示）'],
  ['④ 可选数值三态提示', triState.length, '—', '—', '否（只提示）'],
]))
console.log('')
console.log(`历史三处缺陷回执（第一轮 FE7-01/FE7-02，必须仍“两侧都在场”）：`)
for (const h of historicStatus) {
  console.log(`  ${h.present ? 'OK  ' : h.tsOnly ? 'FAIL' : 'MISS'} ${h.iface}.${h.field}  ${h.note}${h.present ? '' : '  ← 契约回退！'}`)
}

for (const kind of ['missing-struct', 'missing-field']) {
  const label = kind === 'missing-struct' ? '② TS 接口无 Go 结构体' : '① 前端有、Go 缺'
  const added = addedItems.filter((i) => i.kind === kind)
  const ex = exempted.filter((i) => i.kind === kind)
  if (added.length) {
    console.log('')
    console.log(`  NEW   ${label} 新增 ${added.length} 项（未在基线豁免内）：`)
    for (const i of added) console.log(`        ${i.key}${i.kind === 'missing-field' ? `  [${i.type}]` : `  （${i.fields} 字段：${i.sample}…）`}  @${posix(opts.types)}:${i.line}`)
  } else {
    console.log(`  OK    ${label} 无新增（豁免 ${ex.length} 项在基线内）`)
  }
  for (const i of ex) console.log(`        EXEMPT ${i.key}  — ${(baseline.exemptInterfaces ?? []).concat(baseline.exemptFields ?? []).find((b) => b.key === i.key)?.reason ?? ''}`)
}

if (triState.length) {
  console.log('')
  console.log(`  HINT  ④ 可选数值字段在 Go 侧是裸值+omitempty（${triState.length} 条，永不 gate；要三态需指针类型）：`)
  for (const i of triState) console.log(`        ${i.key}  TS ${i.ts} → Go ${i.goType} (${i.goField})  @${posix(opts.go[0])}:${i.line}`)
}
if (reverseOnly.length) {
  console.log('')
  console.log(`  HINT  ③ Go 有、TS 无（${reverseOnly.length} 条，不 gate；可能是有意的服务端字段）：`)
  for (const i of reverseOnly) console.log(`        ${i.iface} ↔ ${i.go}: ${i.tag} (${i.goField} ${i.goType})`)
}

console.log('')
if (STRICT) {
  if (addedItems.length) {
    console.log(`契约漂移守卫 STRICT FAIL：新增漂移 ${addedItems.length} 项（见上）——接 CI 后此处 exit 1`)
    process.exit(1)
  }
  console.log(`契约漂移守卫 STRICT OK：无新增漂移（解析 TS 接口 ${ifaces.size} / 字段 ${fieldsTotal}；重命名豁免 ${baseIfaces.size + baseFields.size} 项）`)
  process.exit(0)
}
if (addedItems.length) {
  console.log(`契约漂移守卫 WARN：新增漂移 ${addedItems.length} 项（见上）——warn 档不 gate，exit 0；接 CI 的 --strict 会 exit 1`)
} else {
  console.log(`契约漂移守卫 WARN：无新增漂移 · 解析 TS 接口 ${ifaces.size} / 字段 ${fieldsTotal} / Go 结构体 ${structs.size}；三态提示 ${triState.length} 条——warn 档 exit 0（未接 CI）`)
}
process.exit(0)
