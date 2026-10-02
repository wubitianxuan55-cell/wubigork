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

// ── ⑤ 数字守卫（P1-7：计数类陈述不再靠手写——自述数 vs 实存数对账）────────
// a) docs/README.md 顶层文档计数
{
  const rm = fs.readFileSync('docs/README.md', 'utf8')
  const m = rm.match(/顶层文档\s*(\d+)\s*份/)
  if (m) {
    const claimed = Number(m[1])
    if (claimed !== topDocs.length) {
      fails.push(`docs/README.md 自述顶层文档 ${claimed} 份，实际 ${topDocs.length} 份——改 README 计数或补登新档`)
    }
  }
}
// b) releases/README.md 的计数陈述（P1-7 立项；2026-10-02 审计 X1-10 加固）
//    加固点：原正则只认「校验和 422 份」这一种语序，而 README 写的是「422 份校验和」，
//    整段守卫静默失效（自述 422 / 实存 427 一直没人发现）。现在两种语序都认，且
//    「一处都认不出」本身判 FAIL——措辞被改掉时守卫会红，不再静默。
{
  const rm = fs.readFileSync('releases/README.md', 'utf8')
  const walk = (dir) =>
    fs.readdirSync(dir, { withFileTypes: true }).flatMap((e) =>
      e.isDirectory() ? walk(path.join(dir, e.name)) : [path.join(dir, e.name)])
  const relFiles = walk('releases')
  const sums = relFiles.filter((f) => /SHA256SUMS-/.test(path.basename(f))).length
  const notes = relFiles.filter((f) => /^v\d+\.\d+\.\d+\.md$/.test(path.basename(f))).length
  const tarballs = relFiles.filter((f) => /-source\.tar\.gz$/.test(f)).length
  const exeRoot = fs.readdirSync('releases').filter((f) => /^gaea-v\d+\.\d+\.\d+\.exe$/.test(f))
  const claims = (text, word) => {
    const out = []
    for (const re of [
      new RegExp(`${word}\\s*(\\d+)\\s*[份个]`, 'g'),
      new RegExp(`(\\d+)\\s*[份个]${word}`, 'g'),
    ]) {
      for (const m of text.matchAll(re)) out.push(Number(m[1]))
    }
    return out
  }
  const checkCount = (word, actual, detail) => {
    const got = claims(rm, word)
    if (got.length === 0) {
      fails.push(
        `releases/README.md 找不到可对账的「${word}」计数措辞——守卫会静默失效，请写成「${word} N 份」或「N 份${word}」（${detail}；实测 ${actual}）`,
      )
    } else if (got.some((n) => n !== actual)) {
      fails.push(`releases/README.md 自述${word} ${got.join(' / ')} 与实测 ${actual} 不符（${detail}）——改 README 计数或补口径`)
    }
  }
  checkCount('校验和', sums, 'releases/ 全目录（含 archive/）的 SHA256SUMS-*')
  checkCount('发布说明', notes, 'releases/ 全目录（含 archive/）的 vX.Y.Z.md')
  // V4_RECENT 区块条数 vs README 自述「最近 N 版」（区块是机器维护的，条数可数）。
  {
    const block = rm.split('<!-- V4_RECENT:start -->')[1]?.split('<!-- V4_RECENT:end -->')[0]
    const entries = block ? (block.match(/^- \[v/gm) ?? []).length : 0
    const claimed = rm.match(/最近\s*(\d+)\s*版/)
    if (block && !claimed) {
      fails.push(`releases/README.md 有 V4_RECENT 区块但找不到「最近 N 版」计数——守卫会静默失效（区块现有 ${entries} 条）`)
    } else if (claimed && Number(claimed[1]) !== entries) {
      fails.push(`releases/README.md 自述最近 ${claimed[1]} 版，V4_RECENT 区块实际 ${entries} 条——随发版同步（本项对账靠区块条数）`)
    }
  }
  // 以下两类产物不入库（.gitignore）：只在本机确有文件时对账，新克隆/CI 上跳过。
  if (tarballs > 0) checkCount('已有', tarballs, 'releases/ 本机 *-source.tar.gz')
  if (exeRoot.length > 0) {
    const m = rm.match(/当前实存二进制\s*=\s*([^\n）)]+)/)
    const listed = m ? [...m[1].matchAll(/v\d+\.\d+\.\d+/g)].map((x) => x[0].replace(/^v/, '')) : []
    const actual = exeRoot.map((f) => f.replace(/^gaea-v/, '').replace(/\.exe$/, '')).sort()
    const missing = actual.filter((v) => !listed.includes(v))
    const stale = listed.filter((v) => !actual.includes(v))
    if (listed.length === 0 || missing.length || stale.length) {
      fails.push(
        `releases/README.md「当前实存二进制 = …」与实存 exe 不符——实存 ${actual.join(', ')}；README 列 ${listed.join(', ') || '（无）'}`,
      )
    }
  }
}

// ── 汇总 ───────────────────────────────────────────────────────────────
console.log(`docs 顶层文档 ${topDocs.length} 份；.gaea/AGENTS.md ${agentsBytes} B / 预算 ${BUDGET} B；.ps1 编码检查 ${walkPs1('scripts').length + walkPs1('.gaea/skills').length} 份`)
{
  const walkRel = (dir) =>
    fs.readdirSync(dir, { withFileTypes: true }).flatMap((e) =>
      e.isDirectory() ? walkRel(path.join(dir, e.name)) : [path.join(dir, e.name)])
  const rel = walkRel('releases')
  console.log(
    `releases/ 校验和 ${rel.filter((f) => /SHA256SUMS-/.test(path.basename(f))).length} 份；` +
      `发布说明 ${rel.filter((f) => /^v\d+\.\d+\.\d+\.md$/.test(path.basename(f))).length} 份；` +
      `exe（本机，含 archive/）${rel.filter((f) => /\.exe$/.test(f)).length} 个；源码包（本机）${rel.filter((f) => /-source\.tar\.gz$/.test(f)).length} 个`,
  )
}
for (const w of warns) console.log('  WARN  ' + w)
if (fails.length) {
  for (const f of fails) console.log('  FAIL  ' + f)
  console.log('仓库卫生守卫 FAIL')
  process.exit(1)
}
console.log('仓库卫生守卫 OK（孤儿 0 / 悬空 0 / 指令预算内 / .ps1 编码合规）')
