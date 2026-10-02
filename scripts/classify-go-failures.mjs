#!/usr/bin/env node
// 假红治理（2026-10-02，审计 X1-12）：把「go test 全量失败」分成两类，取代
// backend job 原先的「整批重试一次」——那会把真回归和真假红一起放行。
//
//   用法：node scripts/classify-go-failures.mjs <go-test-json> [flaky 清单] [复跑包清单] [--expect-failure]
//         node scripts/classify-go-failures.mjs --selftest    # 夹具自测（9 组 / 14 条断言）
//
// 输入是 `go test -json` 的 **stdout**（stderr 请另存，混进来只会多几条被忽略的
// 非 JSON 行——解析器容忍但不建议）。
//
// 输出与退出码：
//   OK: 无失败包                                        → exit 0
//   FLAKY-RETRY: <包…>                                  → exit 0（失败包全在册，允许隔离复跑）
//   REGRESSION: <包…>                                   → exit 1（有册外失败，按真回归红，不重试）
//   UNATTRIBUTED（配 --expect-failure）                  → exit 1（首跑非零却归不出失败包）
//   解析失败（文件读不到 / 没有一条 go test 事件）        → exit 2
//
// 清单格式（scripts/go-known-flaky.txt）：一行一个「包路径片段」，# 开头为注释。
// 只有整份失败包集合都落在清单内才允许复跑——避免复跑把真回归掩盖掉。
// 清单文件缺失/读不到一律按**空清单**处理（失败即红，方向安全不静默放行）。
//
// 与前端同款分类器（frontend/scripts/classify-vitest-failures.mjs）的差异两处：
//   ① 事件源是 `go test -json` 的流式 JSON 行，不是一个报告对象；
//   ② 多一个 --expect-failure：首跑已非零、而报告里一条失败包都归不出来时**不许
//      静默放行**（前端那边报告解析失败天然红，Go 侧存在「有输出却无 fail 事件」
//      的空档，例如 go list 失败后 go test 空跑）。CI 首跑非零时才调本脚本，故必带。

import { mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'

const DEFAULT_LIST = 'scripts/go-known-flaky.txt'
const DEFAULT_RETRY = 'scripts/go-retry-pkgs.txt'

function norm(p) {
  return String(p).replace(/\\/g, '/')
}

// 构建失败的 ImportPath 形如 "example.com/m/p [example.com/m/p.test]"——去掉测试二进制后缀。
function stripTestSuffix(p) {
  return norm(p).replace(/\s*\[[^\]]*\]\s*$/, '')
}

export function readFlakyList(path) {
  try {
    return readFileSync(path, 'utf8')
      .split(/\r?\n/)
      .map((l) => l.trim())
      .filter((l) => l && !l.startsWith('#'))
  } catch {
    return []
  }
}

// parseEvents 解析 go test -json 的流式输出，返回失败包集合。
// 失败信号三类都收（宁可红不可静默绿）：
//   ① 包级 {"Action":"fail","Package":"…"}——用例失败；构建失败同形并带 FailedBuild
//   ② 子测试级 {"Action":"fail","Package":"…","Test":"…"}——进程被杀/日志截断时可能只剩这类
//   ③ go 1.21+ 构建失败 {"Action":"build-fail","ImportPath":"… [pkg.test]"}（无 Package 字段）
export function parseEvents(text) {
  const failed = new Set()
  let sawPackage = false
  let malformed = 0
  // 容错：Windows PowerShell 的重定向（`go test -json … > x.json`）默认落 UTF-16LE，
  // 读成 UTF-8 就是一堆 NUL。BOM 剥掉再按行切，避免「有输出却零事件」的假 PARSE-FAIL。
  for (const line of String(text).replace(/^\uFEFF/, '').split(/\r?\n/)) {
    const t = line.trim()
    if (!t || t[0] !== '{') continue
    let ev
    try {
      ev = JSON.parse(t)
    } catch {
      malformed++
      continue
    }
    const pkg = ev.Package ?? ev.ImportPath
    if (pkg) sawPackage = true
    if (ev.Action === 'build-fail') {
      if (pkg) failed.add(stripTestSuffix(pkg))
      continue
    }
    if (ev.Action !== 'fail' || !pkg) continue
    failed.add(stripTestSuffix(pkg))
  }
  return { failed: [...failed].sort(), sawPackage, malformed }
}

// classify 纯函数：事件文本 + 在册清单 → {code, status, lines, retry}。
export function classify(text, flaky) {
  const { failed, sawPackage, malformed } = parseEvents(text)
  if (!sawPackage) {
    return {
      code: 2,
      status: 'PARSE-FAIL',
      lines: [`无法解析出任何 go test 事件（非 JSON 行 ${malformed} 条）——-json 输出缺失/为空/被截断`],
      retry: [],
    }
  }
  if (failed.length === 0) {
    return { code: 0, status: 'OK', lines: ['OK: 无失败包'], retry: [] }
  }
  const isFlaky = (p) => flaky.some((entry) => p.includes(norm(entry)))
  const registered = failed.filter(isFlaky)
  const regressions = failed.filter((p) => !isFlaky(p))
  if (regressions.length > 0) {
    const lines = [`REGRESSION: 清单外失败 ${regressions.length} 个包（按真回归处理，不重试）`]
    for (const p of regressions) lines.push(`  - ${p}`)
    if (registered.length > 0) lines.push(`（另有在册 flaky ${registered.length} 个：${registered.join(', ')}）`)
    return { code: 1, status: 'REGRESSION', lines, retry: [] }
  }
  const lines = [`FLAKY-RETRY: ${registered.length} 个失败包全部在册，允许隔离复跑`]
  for (const p of registered) lines.push(`  - ${p}`)
  return { code: 0, status: 'FLAKY-RETRY', lines, retry: registered }
}

// applyExpectFailure：首跑已非零时，把「无失败包」从 OK 改判 UNATTRIBUTED（红）。
export function applyExpectFailure(result, expectFailure) {
  if (!expectFailure || result.status !== 'OK') return result
  return {
    code: 1,
    status: 'UNATTRIBUTED',
    lines: [
      'UNATTRIBUTED: 首跑非零，但 -json 报告里一条失败包都归不出来（go list/go test 未产出事件）',
      '——无法归因即按红处理，不许静默放行',
    ],
    retry: [],
  }
}

// readReport 读报告：按 BOM 判编码（UTF-16LE 是 Windows PowerShell 重定向的默认产物，
// 2026-10-02 实撞；CI 走 bash 是纯 UTF-8），失败返回 error 文案（调用方判 exit 2）。
export function readReport(path) {
  let buf
  try {
    buf = readFileSync(path)
  } catch (err) {
    return { error: `无法读取 ${path}: ${err.message}` }
  }
  if (buf.length >= 2 && buf[0] === 0xff && buf[1] === 0xfe) return { text: buf.subarray(2).toString('utf16le') }
  if (buf.length >= 2 && buf[0] === 0xfe && buf[1] === 0xff) {
    const swapped = Buffer.from(buf.subarray(2))
    swapped.swap16()
    return { text: swapped.toString('utf16le') }
  }
  return { text: buf.toString('utf8') }
}

// ── 自测夹具（≥4 组）──────────────────────────────────────────────────────
const ev = (...events) => events.map((e) => JSON.stringify(e)).join('\n') + '\n'
const pass = (p) => [{ Action: 'pass', Package: p, Elapsed: 0.1 }]

function assertCase(name, got, want, results) {
  const bad = []
  for (const [k, v] of Object.entries(want)) {
    const actual = k === 'retry' ? JSON.stringify(got[k]) : got[k]
    const expect = k === 'retry' ? JSON.stringify(v) : v
    if (actual !== expect) bad.push(`${k}: 期望 ${expect} / 实得 ${actual}`)
  }
  results.push(bad.length === 0)
  console.log(`${bad.length === 0 ? 'PASS' : 'FAIL'}  ${name}${bad.length ? ' —— ' + bad.join('；') : ''}`)
}

export function selftest() {
  const results = []
  const tmp = mkdtempSync(join(tmpdir(), 'go-flaky-selftest-'))
  try {
    // ① 全绿
    assertCase(
      '① 全绿：无失败包 → OK/exit 0',
      classify(ev(...pass('m/a'), ...pass('m/b')), []),
      { code: 0, status: 'OK', retry: [] },
      results,
    )

    // ② 失败 ⊆ 清单 → 允许隔离复跑（并落复跑清单）
    const r2 = classify(ev(...pass('github.com/x/gaea/internal/office'), { Action: 'fail', Package: 'github.com/x/gaea/internal/app', Elapsed: 3 }), [
      'internal/app',
    ])
    const retryPath = join(tmp, 'go-retry-pkgs.txt')
    if (r2.status === 'FLAKY-RETRY' && r2.code === 0) writeFileSync(retryPath, r2.retry.join('\n') + '\n')
    const written = (() => {
      try {
        return readFileSync(retryPath, 'utf8').trim()
      } catch {
        return '<未写入>'
      }
    })()
    assertCase('② 失败⊆清单 → FLAKY-RETRY/exit 0', r2, { code: 0, status: 'FLAKY-RETRY', retry: ['github.com/x/gaea/internal/app'] }, results)
    assertCase('② 复跑清单落盘内容', { retry: written }, { retry: 'github.com/x/gaea/internal/app' }, results)

    // ③ 含清单外失败 → 直接红（在册的那条也要列出来，便于对照）
    const r3 = classify(
      ev({ Action: 'fail', Package: 'github.com/x/gaea/internal/app' }, { Action: 'fail', Package: 'github.com/x/gaea/internal/gaea/cost' }),
      ['internal/app'],
    )
    assertCase('③ 含清单外失败 → REGRESSION/exit 1', r3, { code: 1, status: 'REGRESSION', retry: [] }, results)

    // ④ 空清单 + 有失败 → 直接红（本仓当前就是空清单）
    assertCase(
      '④ 空清单+有失败 → REGRESSION/exit 1',
      classify(ev({ Action: 'fail', Package: 'github.com/x/gaea/internal/app' }), []),
      { code: 1, status: 'REGRESSION', retry: [] },
      results,
    )

    // ⑤ 首跑非零却归不出失败包 → --expect-failure 改判 UNATTRIBUTED/exit 1；不带 flag 仍 OK
    const noFail = classify(ev({ Action: 'start', Package: 'm/a' }, { Action: 'output', Package: 'm/a', Output: 'x\n' }), [])
    assertCase('⑤ 无 fail 事件 + --expect-failure → UNATTRIBUTED/exit 1', applyExpectFailure(noFail, true), { code: 1, status: 'UNATTRIBUTED' }, results)
    assertCase('⑤ 无 fail 事件 + 无 flag → OK/exit 0', applyExpectFailure(noFail, false), { code: 0, status: 'OK' }, results)

    // ⑥ 报告解析失败（非 JSON） → exit 2
    assertCase('⑥ 非 JSON 报告 → PARSE-FAIL/exit 2', classify('go: downloading something\n', []), { code: 2, status: 'PARSE-FAIL' }, results)

    // ⑦ 构建失败形态（build-fail ImportPath 带 [pkg.test] 后缀）→ 归到包并红
    const r7 = classify(ev({ Action: 'build-output', ImportPath: 'm/broken [m/broken.test]', Output: '# m/broken\n' }, { Action: 'build-fail', ImportPath: 'm/broken [m/broken.test]' }), [])
    assertCase('⑦ 构建失败（build-fail）→ 归包 + REGRESSION/exit 1', r7, { code: 1, status: 'REGRESSION', retry: [] }, results)
    assertCase('⑦ 构建失败报点已去 [pkg.test] 后缀', { first: r7.lines[1] }, { first: '  - m/broken' }, results)

    // ⑧ 子测试级失败（无包级 fail，日志截断场景）→ 仍按包红，不静默绿
    const r8 = classify(ev(...pass('m/a'), { Action: 'fail', Package: 'm/a', Test: 'TestX' }, { Action: 'fail', Package: 'm/b', Test: 'TestY/sub' }), ['m/b'])
    assertCase('⑧ 仅子测试级失败也归包（截断兜底）→ REGRESSION/exit 1', r8, { code: 1, status: 'REGRESSION' }, results)

    // ⑨ 编码容错：UTF-16LE（Windows PowerShell 重定向默认）/ 带 BOM 的 UTF-8 都要能读
    const u16 = join(tmp, 'report-utf16.json')
    writeFileSync(u16, Buffer.concat([Buffer.from([0xff, 0xfe]), Buffer.from(ev({ Action: 'fail', Package: 'm/u16' }), 'utf16le')]))
    const u16res = readReport(u16)
    assertCase('⑨ UTF-16LE 报告可读 → 事件归包', classify(u16res.text ?? '', []), { code: 1, status: 'REGRESSION', retry: [] }, results)
    const bom = join(tmp, 'report-bom.json')
    writeFileSync(bom, Buffer.concat([Buffer.from([0xef, 0xbb, 0xbf]), Buffer.from(ev(...pass('m/a')), 'utf8')]))
    assertCase('⑨ 带 BOM 的 UTF-8 报告可读 → OK/exit 0', classify(readReport(bom).text ?? '', []), { code: 0, status: 'OK' }, results)
    assertCase('⑨ 报告缺失 → error（调用方 exit 2）', { hasError: Boolean(readReport(join(tmp, 'nope.json')).error) }, { hasError: true }, results)
  } finally {
    rmSync(tmp, { recursive: true, force: true })
  }
  const failed = results.filter((r) => !r).length
  console.log(failed === 0 ? `go flaky 分类器自测全过（${results.length}/${results.length}）` : `go flaky 分类器自测失败 ${failed}/${results.length}`)
  return failed === 0 ? 0 : 1
}

export function main(argv = process.argv.slice(2)) {
  if (argv.includes('--selftest')) return selftest()
  const positional = argv.filter((a) => !a.startsWith('--'))
  const [reportPath, listPath = DEFAULT_LIST, retryListPath = DEFAULT_RETRY] = positional
  if (!reportPath) {
    console.error('用法: node scripts/classify-go-failures.mjs <go-test-json> [flaky 清单] [复跑包清单] [--expect-failure]')
    console.error('      node scripts/classify-go-failures.mjs --selftest')
    return 2
  }
  const report = readReport(reportPath)
  if (report.error) {
    console.error(report.error)
    return 2
  }
  const text = report.text
  const result = applyExpectFailure(classify(text, readFlakyList(listPath)), argv.includes('--expect-failure'))
  for (const line of result.lines) console.log(line)
  if (result.status === 'FLAKY-RETRY' && retryListPath) {
    writeFileSync(retryListPath, result.retry.join('\n') + '\n')
    console.log(`复跑清单已写入 ${retryListPath}（${result.retry.length} 个包）`)
  }
  return result.code
}

process.exit(main())
