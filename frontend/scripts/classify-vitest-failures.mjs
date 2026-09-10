#!/usr/bin/env node
// 假红治理（2026-09-10）：把「vitest 全量失败」分成两类，取代无条件的
// 「失败就重试一次」（那会把真回归和真假红一起放行）。
//
//   用法：node scripts/classify-vitest-failures.mjs <vitest-json-report> [flaky 清单]
//
// 输出与退出码：
//   OK: 无失败                              → exit 0
//   FLAKY-RETRY: <文件…>                    → exit 0（全部失败文件都在册，允许隔离复跑）
//   REGRESSION: <文件…>                     → exit 1（有册外失败，按真回归红）
//   清单为空且全量失败                        → REGRESSION（本仓当前就是空清单）
//
// 清单格式（frontend/known-flaky.txt）：一行一个「路径片段」，# 开头为注释。
// 只有整份失败集合都落在清单内才允许复跑——避免复跑把真回归掩盖掉。

import { readFileSync, writeFileSync } from 'node:fs'
import { relative } from 'node:path'

const [, , reportPath, listPath = 'known-flaky.txt', retryListPath = 'vitest-retry-files.txt'] = process.argv

if (!reportPath) {
  console.error('用法: node scripts/classify-vitest-failures.mjs <vitest-json-report> [flaky 清单]')
  process.exit(2)
}

function readFlakyList(path) {
  try {
    return readFileSync(path, 'utf8')
      .split(/\r?\n/)
      .map((l) => l.trim())
      .filter((l) => l && !l.startsWith('#'))
  } catch {
    return []
  }
}

function norm(p) {
  return String(p).replace(/\\/g, '/')
}

let report
try {
  report = JSON.parse(readFileSync(reportPath, 'utf8'))
} catch (err) {
  console.error(`无法解析 vitest 报告 ${reportPath}: ${err.message}`)
  process.exit(2)
}

const cwd = process.cwd()
const failedFiles = (report.testResults ?? [])
  .filter((r) => r.status === 'failed')
  .map((r) => norm(relative(cwd, r.name ?? '')))
  .filter(Boolean)
  .sort()

if (failedFiles.length === 0) {
  console.log(`OK: 无失败（${report.numTotalTests ?? '?'} 例）`)
  process.exit(0)
}

const flaky = readFlakyList(listPath)
const isFlaky = (f) => flaky.some((entry) => norm(f).includes(norm(entry)))

const registered = failedFiles.filter(isFlaky)
const regressions = failedFiles.filter((f) => !isFlaky(f))

if (regressions.length > 0) {
  console.log(`REGRESSION: 清单外失败 ${regressions.length} 个文件（按真回归处理，不重试）`)
  for (const f of regressions) console.log(`  - ${f}`)
  if (registered.length > 0) {
    console.log(`（另有在册 flaky ${registered.length} 个：${registered.join(', ')}）`)
  }
  process.exit(1)
}

console.log(`FLAKY-RETRY: ${registered.length} 个失败文件全部在册，允许隔离复跑`)
for (const f of registered) console.log(`  - ${f}`)
// 供 CI 直接喂给 vitest 复跑（只跑这些文件，避免全量重跑把时间放大）
writeFileSync(retryListPath, registered.join('\n') + '\n')
process.exit(0)
