// gaea-settings-walk.mjs — 设置中枢真机走查驱动（CDP 断言回归配方，2026-11-20 转正）
// 覆盖面：设置内容头方位感/板块索引卡+NAVIGATE 直达/搜索零死端/inspector 相关分组/
//        模型中心 rail 7tab+引擎管理两段——设置中枢重构线（批 1~7+回摆+重设计+索引卡）
//        的实机断言面。后续设置域动刀复跑本脚本即回归。
// 前置（必读 docs/gaea-build-guide.md 坑册）：
//   1. 实盘壳必须先关闭——gaea 壳因 --disable-features 被 Wails 连带默认开
//      127.0.0.1:9333，实盘壳恒占口，走查壳绑不上（2026-10-09 事故红线：严禁直接驱动实盘壳）。
//   2. 开走查沙箱壳（数据面全隔离到 Temp\gaea-walkthrough）：
//        GAEA_WALKTHROUGH=1 GAEA_WEBVIEW_DEBUG=1 build\bin\gaea.exe
//   3. node scripts/gaea-settings-walk.mjs   （仓根执行；截图落 .tmp/）
//   4. 跑完回收走查壳，再 gaea_install.ps1 -Launch 装回实盘。
// 断言 17 条，FAIL 时 exit 1；两条历史口径注意：NAVIGATE 断言钉「最后一条事件」
//（__navSeen 累积）、SectionHead 标题过滤空 span 噪音后比对。
import { readFileSync, writeFileSync } from 'node:fs'

const PORT = process.env.GAEA_WALK_PORT || '9333'
const list = await (await fetch(`http://127.0.0.1:${PORT}/json`)).json()
const pages = list.filter((t) => t.type === 'page')
const page = pages.find((t) => (t.url ?? '').includes('wails')) ?? pages[0]
if (!page) throw new Error(`no page target（现有：${pages.map((p) => p.url).join(', ')}）`)
console.log('target:', page.url)

const ws = new WebSocket(page.webSocketDebuggerUrl)
let id = 0
const pending = new Map()
function send(method, params = {}) {
  return new Promise((res, rej) => {
    const mid = ++id
    pending.set(mid, { res, rej })
    ws.send(JSON.stringify({ id: mid, method, params }))
  })
}
ws.onmessage = (ev) => {
  const msg = JSON.parse(ev.data)
  if (msg.id && pending.has(msg.id)) {
    const { res, rej } = pending.get(msg.id)
    pending.delete(msg.id)
    msg.error ? rej(new Error(JSON.stringify(msg.error))) : res(msg.result)
  }
}
await new Promise((res, rej) => { ws.onopen = res; ws.onerror = rej })

const sleep = (ms) => new Promise((r) => setTimeout(r, ms))
async function evalJs(expr, awaitPromise = false) {
  const r = await send('Runtime.evaluate', {
    expression: expr, awaitPromise, returnByValue: true,
  })
  if (r.exceptionDetails) throw new Error('页面内异常: ' + JSON.stringify(r.exceptionDetails.exception?.description ?? r.exceptionDetails.text))
  return r.result?.value
}
async function shot(name) {
  const r = await send('Page.captureScreenshot', { format: 'png' })
  writeFileSync(`.tmp/walk-20261120-${name}.png`, Buffer.from(r.data, 'base64'))
}

const results = []
function check(name, ok, detail = '') {
  results.push({ name, ok, detail })
  console.log(`${ok ? 'PASS' : 'FAIL'}  ${name}${detail ? `  [${detail}]` : ''}`)
}
const waitEq = async (expr, want, name, tries = 20) => {
  for (let i = 0; i < tries; i++) {
    const v = await evalJs(expr)
    if (String(v) === String(want)) { check(name, true); return }
    await sleep(150)
  }
  const v = await evalJs(expr)
  check(name, false, `实际=${v} 期望=${want}`)
}

// ── 0. 监听 NAVIGATE 事件（索引卡直达契约验证） ──
await evalJs(`(() => {
  window.__navSeen = []
  const orig = window.dispatchEvent.bind(window)
  window.dispatchEvent = (e) => { if (e && e.type === 'navigate') window.__navSeen.push(e.detail && e.detail.page); return orig(e) }
  return 'hooked'
})()`)

// ── 1. 设置页：内容头方位感 ──
await evalJs(`window.dispatchEvent(new CustomEvent('navigate', { detail: { page: 'settings' } }))`)
await waitEq(`(document.querySelector('.settings-head__title')||{}).textContent`, '通用', '内容头默认组=通用')
check('内容头 zone 徽标=全局', await evalJs(`(document.querySelector('.settings-head__zone')||{}).textContent`) === '全局')

// ── 2. 索引卡（进度计划）+ NAVIGATE 直达 ──
await evalJs(`[...document.querySelectorAll('.settings-nav-item')].find(b => b.textContent.includes('进度计划'))?.click()`, true)
await waitEq(`!!document.querySelector('.board-index-card')`, 'true', '进度计划索引卡渲染')
check('索引卡题文案', await evalJs(`(document.querySelector('.board-index-card__title')||{}).textContent`) === '进度计划偏好（板块内即时调）')
check('可调项胶囊≥5', (await evalJs(`document.querySelectorAll('.board-index-card__item').length`)) >= 5)
await evalJs(`[...document.querySelectorAll('.board-index-card .ant-btn')].find(b => b.textContent.includes('打开进度计划'))?.click()`, true)
await sleep(500)
// __navSeen 为历史累积（步骤 1 的 navigate settings 也在内），钉最后一条=索引卡直达目标
check('NAVIGATE 直达 schedule', ((await evalJs(`(window.__navSeen||[]).join(',')`)) || '').endsWith('schedule'), await evalJs(`(window.__navSeen||[]).join(',')`))

// ── 3. 回设置页：搜索零死端 ──
await evalJs(`window.dispatchEvent(new CustomEvent('navigate', { detail: { page: 'settings' } }))`)
await sleep(300)
await evalJs(`(() => {
  const input = document.querySelector('.settings-page__search input')
  if (!input) return 'no-input'
  const setter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value').set
  setter.call(input, 'zzzzq')
  input.dispatchEvent(new Event('input', { bubbles: true }))
  return 'ok'
})()`, true)
await waitEq(`document.querySelectorAll('.settings-empty__chip').length`, 4, '无匹配建议胶囊×4')
await waitEq(`(document.querySelector('.settings-empty__title')||{}).textContent.length > 0`, 'true', '无匹配文案在')
await evalJs(`[...document.querySelectorAll('.settings-empty__chip')].find(b => b.textContent.includes('模型'))?.click()`, true)
await waitEq(`!!document.querySelector('.settings-empty__chip')`, 'false', '点击胶囊清搜索恢复')
await waitEq(`(document.querySelector('.settings-head__title')||{}).textContent`, '模型', '直达后内容头=模型')

// ── 4. inspector 相关分组快跳 ──
await evalJs(`document.querySelector('.settings-bar__toggle')?.click()`, true)
await sleep(300)
await waitEq(`[...document.querySelectorAll('.settings-help-card__title')].some(h => h.textContent === '相关分组')`, 'true', '相关分组卡在（模型组关键词交集非空）')
check('相关分组胶囊>0', (await evalJs(`document.querySelectorAll('.settings-related-item').length`)) > 0)
await evalJs(`document.querySelector('.settings-inspector__close')?.click()`, true)
await evalJs(`[...document.querySelectorAll('.settings-nav-item')].find(b => b.textContent.includes('通用'))?.click()`, true)

await shot('settings')

// ── 5. 模型中心：rail 7 tab / 4 节 + 引擎管理两段 ──
await evalJs(`window.dispatchEvent(new CustomEvent('navigate', { detail: { page: 'modelcenter' } }))`)
await waitEq(`document.querySelectorAll('.mc-rail-item').length`, 7, 'rail 7 tab（回摆断言）', 60)
await waitEq(`document.querySelectorAll('.mc-rail-group').length`, 4, 'rail 4 节', 60)
check('接入节标在', (await evalJs(`[...document.querySelectorAll('.mc-rail-group')].some(g => g.textContent === '接入')`)) === true)
check('功能绑定 tab 在', (await evalJs(`[...document.querySelectorAll('.mc-rail-label')].some(l => l.textContent === '功能绑定')`)) === true)
await evalJs(`[...document.querySelectorAll('.mc-rail-item')].find(b => b.textContent.includes('引擎管理'))?.click()`, true)
await sleep(300)
// 过滤空 span 噪音（SectionHead 辅助 span），钉两段标题顺序
await waitEq(`[...document.querySelectorAll('.mc-section-title span')].map(s => s.textContent).filter(t => t && t.trim()).join('|')`, '本地调度|引擎管理', '引擎管理=调度+引擎两段', 30)
await shot('modelcenter')

ws.close()
const fails = results.filter((r) => !r.ok)
console.log(`\n汇总：${results.length - fails.length}/${results.length} PASS${fails.length ? `，FAIL=${fails.map((f) => f.name).join('; ')}` : ''}`)
process.exit(fails.length ? 1 : 0)
