#!/usr/bin/env node
// 假红治理（2026-09-10 收口；2026-10-02 批次十三 round 15 补自测 + 补 UNATTRIBUTED）：
// 把「vitest 全量失败」分成两类，取代无条件的「失败就重试一次」（那会把真回归和
// 真假红一起放行）。
//
//   用法：node scripts/classify-vitest-failures.mjs <vitest-json-report> [flaky 清单] [复跑清单] [--expect-failure]
//         node scripts/classify-vitest-failures.mjs --selftest    # 夹具自测（8 组 / 14 条断言）
//
// 输出与退出码（与 scripts/classify-go-failures.mjs 同款形态）：
//   OK: 无失败文件                                → exit 0
//   FLAKY-RETRY: <文件…>                          → exit 0（失败文件全在册，允许隔离复跑）
//   REGRESSION: <文件…>                           → exit 1（有册外失败，按真回归红，不重试）
//   UNATTRIBUTED（配 --expect-failure）           → exit 1（首跑非零却归不出失败文件）
//   报告读不到 / 非 JSON / 无 testResults 事件     → exit 2
//
// 清单格式（frontend/known-flaky.txt）：一行一个「路径片段」，# 开头为注释。
// 只有整份失败集合都落在清单内才允许复跑——避免复跑把真回归掩盖掉。
// 清单文件缺失/读不到一律按**空清单**处理（失败即红，方向安全不静默放行）。
//
// 与 Go 侧分类器的差异：事件源是 vitest 的 JSON **报告对象**（不是流式事件行），
// 失败判据是 testResults[].status === 'failed'。--expect-failure 与 Go 侧同义：
// CI 只在首跑非零时才调本脚本，故「报告里一条失败文件都归不出来」不许静默放行。

import { mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join, relative } from 'node:path'

const DEFAULT_LIST = 'known-flaky.txt'
const DEFAULT_RETRY = 'vitest-retry-files.txt'

export function norm(p) {
  return String(p).replace(/\\/g, '/')
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

// readReport 读报告：读不到 / 非 JSON 一律返回 error 文案（调用方判 exit 2）。
export function readReport(path) {
  let text
  try {
    text = readFileSync(path, 'utf8')
  } catch (err) {
    return { error: `无法读取 ${path}: ${err.message}` }
  }
  try {
    return { report: JSON.parse(text.replace(/^\uFEFF/, '')) }
  } catch (err) {
    return { error: `无法解析 vitest 报告 ${path}: ${err.message}` }
  }
}

// classify 纯函数：报告对象 + 在册清单 → {code, status, lines, retry}。
// 报告里没有可用的 testResults 数组（vitest 崩在配置/收集阶段、报告被截断、写出
// `{}`）＝「零事件」，按 PARSE-FAIL/exit 2 红——此前这种形态落进 OK/exit 0，
// 首跑已非零却静默放行。
export function classify(report, flaky, cwd = process.cwd()) {
  const results = report?.testResults
  if (!Array.isArray(results)) {
    return {
      code: 2,
      status: 'PARSE-FAIL',
      lines: ['无法从报告里解析出任何 testResults 事件（报告缺失该字段/被截断/为空）——按红处理'],
      retry: [],
    }
  }
  const failedFiles = results
    .filter((r) => r && r.status === 'failed')
    .map((r) => norm(relative(cwd, r.name ?? '')))
    .filter(Boolean)
    .sort()

  if (failedFiles.length === 0) {
    return { code: 0, status: 'OK', lines: [`OK: 无失败（${report.numTotalTests ?? '?'} 例）`], retry: [] }
  }

  const isFlaky = (f) => flaky.some((entry) => norm(f).includes(norm(entry)))
  const registered = failedFiles.filter(isFlaky)
  const regressions = failedFiles.filter((f) => !isFlaky(f))

  if (regressions.length > 0) {
    const lines = [`REGRESSION: 清单外失败 ${regressions.length} 个文件（按真回归处理，不重试）`]
    for (const f of regressions) lines.push(`  - ${f}`)
    if (registered.length > 0) lines.push(`（另有在册 flaky ${registered.length} 个：${registered.join(', ')}）`)
    return { code: 1, status: 'REGRESSION', lines, retry: [] }
  }

  const lines = [`FLAKY-RETRY: ${registered.length} 个失败文件全部在册，允许隔离复跑`]
  for (const f of registered) lines.push(`  - ${f}`)
  return { code: 0, status: 'FLAKY-RETRY', lines, retry: registered }
}

// applyExpectFailure：首跑已非零时，把「无失败文件」从 OK 改判 UNATTRIBUTED（红）。
export function applyExpectFailure(result, expectFailure) {
  if (!expectFailure || result.status !== 'OK') return result
  return {
    code: 1,
    status: 'UNATTRIBUTED',
    lines: [
      'UNATTRIBUTED: 首跑非零，但报告里一条失败文件都归不出来（vitest 崩在收集/配置阶段？）',
      '——无法归因即按红处理，不许静默放行',
    ],
    retry: [],
  }
}

// ── 自测夹具（8 组 / 14 条断言）──────────────────────────────────────────
const p = (...segs) => norm(join(...segs))
const t = (name, status) => ({ name, status, assertionResults: [] })

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

// quiet 执行走 main() 真实退出码路径的夹具，但把该路径预期内的控制台输出压掉——
// 免得自测日志里混进「无法读取报告」这类**故意**制造的报错。
function quiet(fn) {
  const { log, error } = console
  console.log = () => {}
  console.error = () => {}
  try {
    return fn()
  } finally {
    console.log = log
    console.error = error
  }
}

export function selftest() {
  const results = []
  const tmp = mkdtempSync(join(tmpdir(), 'vitest-flaky-selftest-'))
  const cwd = process.cwd()
  const listPath = join(tmp, 'known-flaky.txt')
  const retryPath = join(tmp, 'vitest-retry-files.txt')
  const flakyLine = 'src/gaea/lib/flaky'
  writeFileSync(listPath, `# 夹具清单\n${flakyLine}\n`)
  try {
    // ① 全绿：无失败文件 → OK/exit 0（端到端走一遍 CLI 的真实绿路径）
    const green = { numTotalTests: 2, testResults: [t(p(cwd, 'src/a.test.ts'), 'passed'), t(p(cwd, 'src/b.test.ts'), 'passed')] }
    assertCase('① 全绿 → OK/exit 0', classify(green, [flakyLine], cwd), { code: 0, status: 'OK', retry: [] }, results)
    const greenPath = join(tmp, 'green-report.json')
    writeFileSync(greenPath, JSON.stringify(green))
    assertCase('① 全绿经 main() → 退出码 0', { code: quiet(() => main([greenPath, listPath, retryPath])) }, { code: 0 }, results)

    // ② 失败 ⊆ 清单 → 允许隔离复跑（并落复跑清单，只列失败文件）
    const r2 = classify(
      { numTotalTests: 3, testResults: [t(p(cwd, 'src/gaea/lib/flaky.test.ts'), 'failed'), t(p(cwd, 'src/a.test.ts'), 'passed')] },
      [flakyLine],
      cwd,
    )
    if (r2.status === 'FLAKY-RETRY' && r2.code === 0) writeFileSync(retryPath, r2.retry.join('\n') + '\n')
    assertCase('② 失败⊆清单 → FLAKY-RETRY/exit 0', r2, { code: 0, status: 'FLAKY-RETRY', retry: ['src/gaea/lib/flaky.test.ts'] }, results)
    const written = (() => {
      try {
        return readFileSync(retryPath, 'utf8').trim()
      } catch {
        return '<未写入>'
      }
    })()
    assertCase('② 复跑清单落盘内容', { retry: written }, { retry: 'src/gaea/lib/flaky.test.ts' }, results)

    // ③ 含清单外失败 → 直接红（在册的那条也要列出来，便于对照）
    const r3 = classify(
      {
        testResults: [t(p(cwd, 'src/gaea/lib/flaky.test.ts'), 'failed'), t(p(cwd, 'src/gaea/lib/real.test.ts'), 'failed'), t(p(cwd, 'src/a.test.ts'), 'passed')],
      },
      [flakyLine],
      cwd,
    )
    assertCase('③ 含清单外失败 → REGRESSION/exit 1', r3, { code: 1, status: 'REGRESSION', retry: [] }, results)
    assertCase('③ 报点只列清单外那条', { first: r3.lines[1] }, { first: '  - src/gaea/lib/real.test.ts' }, results)

    // ④ 空清单 + 有失败 → 直接红（本仓当前就是空清单）
    const r4 = classify({ testResults: [t(p(cwd, 'src/gaea/lib/flaky.test.ts'), 'failed')] }, [], cwd)
    assertCase('④ 空清单+有失败 → REGRESSION/exit 1', r4, { code: 1, status: 'REGRESSION', retry: [] }, results)

    // ⑤ 无失败事件但首跑非零 → --expect-failure 改判 UNATTRIBUTED/exit 1；不带 flag 仍 OK
    const noFail = classify({ numTotalTests: 5, testResults: [t(p(cwd, 'src/a.test.ts'), 'passed')] }, [flakyLine], cwd)
    assertCase('⑤ 无失败事件 + --expect-failure → UNATTRIBUTED/exit 1', applyExpectFailure(noFail, true), { code: 1, status: 'UNATTRIBUTED', retry: [] }, results)
    assertCase('⑤ 无失败事件 + 无 flag → OK/exit 0', applyExpectFailure(noFail, false), { code: 0, status: 'OK' }, results)

    // ⑥ 报告缺失 → exit 2（读不到不许当全绿）
    assertCase('⑥ 报告缺失 → main 退出码 2', { code: quiet(() => main([join(tmp, 'nope.json'), listPath, retryPath])) }, { code: 2 }, results)

    // ⑦ 报告不可解析（非 JSON）→ exit 2
    const bad = join(tmp, 'bad-report.json')
    writeFileSync(bad, '{ "testResults": [ truncated…\n')
    assertCase('⑦ 报告不可解析 → main 退出码 2', { code: quiet(() => main([bad, listPath, retryPath])) }, { code: 2 }, results)

    // ⑧ 报告可解析但零事件：结构不可用（无 testResults 字段）按 PARSE-FAIL/exit 2 红；
    //    「有 testResults 数组但一个用例都没收集到」（vitest 报 no test files）不带 flag
    //    仍 OK，配 --expect-failure 则 UNATTRIBUTED/exit 1——CI 两种形态都不会静默放行。
    assertCase('⑧ 无 testResults 字段 → PARSE-FAIL/exit 2', classify({ numTotalTests: 0 }, []), { code: 2, status: 'PARSE-FAIL', retry: [] }, results)
    const emptyRun = classify({ numTotalTests: 0, testResults: [] }, [], cwd)
    assertCase('⑧ 空 testResults 数组 + 无 flag → OK/exit 0', emptyRun, { code: 0, status: 'OK', retry: [] }, results)
    assertCase('⑧ 空 testResults 数组 + --expect-failure → UNATTRIBUTED/exit 1', applyExpectFailure(emptyRun, true), { code: 1, status: 'UNATTRIBUTED' }, results)
  } finally {
    rmSync(tmp, { recursive: true, force: true })
  }
  const failed = results.filter((r) => !r).length
  console.log(failed === 0 ? `vitest flaky 分类器自测全过（${results.length}/${results.length}）` : `vitest flaky 分类器自测失败 ${failed}/${results.length}`)
  return failed === 0 ? 0 : 1
}

export function main(argv = process.argv.slice(2)) {
  if (argv.includes('--selftest')) return selftest()
  const positional = argv.filter((a) => !a.startsWith('--'))
  const [reportPath, listPath = DEFAULT_LIST, retryListPath = DEFAULT_RETRY] = positional
  if (!reportPath) {
    console.error('用法: node scripts/classify-vitest-failures.mjs <vitest-json-report> [flaky 清单] [复跑清单] [--expect-failure]')
    console.error('      node scripts/classify-vitest-failures.mjs --selftest')
    return 2
  }
  const read = readReport(reportPath)
  if (read.error) {
    console.error(read.error)
    return 2
  }
  const result = applyExpectFailure(classify(read.report, readFlakyList(listPath)), argv.includes('--expect-failure'))
  for (const line of result.lines) console.log(line)
  if (result.status === 'FLAKY-RETRY' && retryListPath) {
    writeFileSync(retryListPath, result.retry.join('\n') + '\n')
    console.log(`复跑清单已写入 ${retryListPath}（${result.retry.length} 个文件）`)
  }
  return result.code
}

process.exit(main())
