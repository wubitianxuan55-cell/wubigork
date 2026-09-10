// 仓库卫生守卫（2026-09-10 立项）：防这次整理的四个坑复发。
//   ① 孤儿文档：docs/ 顶层 *.md 必须登记进 docs/README.md（无登记=后续会话找不到）
//   ② 悬空引用：活跃文件里的 docs/xxx.md 必须真实存在（文件移入 archive/ 后引用常忘改）
//   ③ 指令预算：.gaea/AGENTS.md 不得超过工作区指令预算 65536 B——超了尾部纪律段
//      会对后续会话不可见（2026-09-10 实测：104 KB → 尾部「执行纪律/长期规划/发布流程」全被截断）
//   ④ 脚本编码：含非 ASCII 的 .ps1 必须带 UTF-8 BOM——否则 powershell.exe 按 GBK 解析报错
//      （2026-09-10 实撞：编辑 ci.ps1 丢了 BOM，整条门禁脚本直接语法错、CI 静默不跑）
// 用法：node scripts/check-docs.mjs   （退出码 0=过；1=有失败项）
import fs from 'node:fs'
import path from 'node:path'

const BUDGET = 65536 // 工作区指令预算（硬上限）
const WARN_AT = 58000 // 提前预警水位（留 ~7 KB 余量给下一批版本条目）

// 非本仓文档白名单（出现即合理，两类）：
//   ① 上游仓库文档路径：蒸馏方案里引用被取道项目的自身文档（univer / better-sidebar / 社区技能）
//   ② 检索评测集里的「工作区文件名」示例：retrieval-eval-set.md 的 kind:file name 字段，
//      是 gaea 运行时解析的用户工作区文件名，不是仓库文档
const NOT_REPO_DOCS = [
  /^docs\/(architecture|compatibility|external-plugin-guide|univer-turn-projection-and-surfaces|anti-ai-polish)\.md$/,
  /^docs\/(材料价格信息价|投标文件模板)\.md$/,
]

const fails = []
const warns = []

// ── ① 孤儿文档 ─────────────────────────────────────────────────────────
const index = fs.readFileSync('docs/README.md', 'utf8')
const topDocs = fs.readdirSync('docs').filter((f) => f.endsWith('.md') && f !== 'README.md')
const orphans = topDocs.filter((f) => !index.includes(f))
if (orphans.length) fails.push(`docs/README.md 未登记：${orphans.join(', ')}（新增文档必须同步索引）`)

// ── ② 悬空引用 ─────────────────────────────────────────────────────────
const sources = [
  '.gaea/AGENTS.md', '.gaea/progress.md', '.gaea/todos.md', 'README.md',
  'docs/README.md', 'docs/archive/README.md',
  ...topDocs.map((f) => 'docs/' + f),
]
const dangling = []
for (const src of sources) {
  if (!fs.existsSync(src)) continue
  const text = fs.readFileSync(src, 'utf8')
  const refs = new Set()
  for (const m of text.matchAll(/docs\/[A-Za-z0-9_.\-\u4e00-\u9fa5/]*\.md/g)) refs.add(m[0].replace(/[.,;)]+$/, ''))
  for (const r of refs) {
    if (NOT_REPO_DOCS.some((re) => re.test(r))) continue
    if (!fs.existsSync(r)) dangling.push(`${src} → ${r}`)
  }
}
if (dangling.length) fails.push(`悬空引用 ${dangling.length} 处：\n      ` + dangling.join('\n      '))

// ── ③ AGENTS.md 指令预算 ───────────────────────────────────────────────
const agentsBytes = fs.statSync('.gaea/AGENTS.md').size
if (agentsBytes > BUDGET) {
  fails.push(`.gaea/AGENTS.md ${agentsBytes} B > 指令预算 ${BUDGET} B——请把较旧版本条目迁入 docs/archive/agents-version-history-2026-09.md`)
} else if (agentsBytes > WARN_AT) {
  warns.push(`.gaea/AGENTS.md ${agentsBytes} B 已近预算水位（${WARN_AT} B）——下批版本条目可能触顶，先分流`)
}

// ── ④ 含非 ASCII 的 .ps1 必须带 UTF-8 BOM ──────────────────────────────
const SKIP_DIRS = /(^|[\\/])(node_modules|\.tmp|clones|backups|dist|build)([\\/]|$)/
const walkPs1 = (dir, out = []) => {
  for (const e of fs.readdirSync(dir, { withFileTypes: true })) {
    const p = path.join(dir, e.name)
    if (SKIP_DIRS.test(p)) continue
    if (e.isDirectory()) walkPs1(p, out)
    else if (e.name.endsWith('.ps1')) out.push(p)
  }
  return out
}
const bomLess = []
for (const p of [...walkPs1('scripts'), ...walkPs1('.gaea/skills')]) {
  const raw = fs.readFileSync(p)
  const hasBom = raw.length >= 3 && raw[0] === 0xef && raw[1] === 0xbb && raw[2] === 0xbf
  const text = raw.toString('utf8')
  const nonAscii = (text.match(/[^\x00-\x7F]/g) ?? []).length
  if (!hasBom && nonAscii > 0) bomLess.push(`${p}（非 ASCII ${nonAscii} 字符，无 BOM）`)
}
if (bomLess.length) fails.push(`.ps1 缺 BOM：\n      ` + bomLess.join('\n      '))

// ── 汇总 ───────────────────────────────────────────────────────────────
console.log(`docs 顶层文档 ${topDocs.length} 份；.gaea/AGENTS.md ${agentsBytes} B / 预算 ${BUDGET} B；.ps1 编码检查 ${walkPs1('scripts').length + walkPs1('.gaea/skills').length} 份`)
for (const w of warns) console.log('  WARN  ' + w)
if (fails.length) {
  for (const f of fails) console.log('  FAIL  ' + f)
  console.log('仓库卫生守卫 FAIL')
  process.exit(1)
}
console.log('仓库卫生守卫 OK（孤儿 0 / 悬空 0 / 指令预算内 / .ps1 编码合规）')
