// 本地分片全量跑（2026-11-12 成本刀·杠杆 2）。
// 依据：vitest 官方性能指南——高核数机器上整个 run 只有主线程里的一个 Vite
// 服务器，所有 worker 的模块转换请求都压在它身上；32 逻辑核机器建议把
// 测试拆到多个进程，每个分片独立 Vite 服务器 + 独立 worker 池，墙钟≈CPU
// 才摊得平。分片结果走 blob 报告（默认目录 .vitest/，已 gitignore），最后
// merge-reports 合并输出一次完整摘要；退出码 = 合并结果的退出码。
//
// 用法：
//   npm run test:sharded            # 默认 2 分片 × 7 worker（16 物理核甜点：
//                                   # (1 主线程 + 7 worker) × 2 = 16，与官方配方同式）
//   node scripts/test-shard.mjs 4   # 显式 4 分片
//   GAEA_TEST_MAX_WORKERS=6 npm run test:sharded   # 覆写每分片 worker 数
import { spawn } from 'node:child_process'
import os from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const root = path.dirname(path.dirname(fileURLToPath(import.meta.url)))
const vitestEntry = path.join(root, 'node_modules', 'vitest', 'vitest.mjs')
const shards = Math.max(2, Number(process.argv[2] ?? process.env.GAEA_TEST_SHARDS ?? 2))
// 物理核≈逻辑核折半；每分片 1 主线程 + N worker，总占用量入物理核数。
const workers = Number(
  process.env.GAEA_TEST_MAX_WORKERS
    ?? Math.max(4, Math.floor((Math.floor(os.availableParallelism() / 2) - shards) / shards)),
)

const run = (args, label) =>
  new Promise((resolve) => {
    const startedAt = Date.now()
    const child = spawn(process.execPath, [vitestEntry, ...args], {
      cwd: root,
      stdio: ['ignore', 'inherit', 'inherit'],
    })
    child.on('exit', (code) => {
      console.log(`[shard] ${label} exit=${code} (${Math.round((Date.now() - startedAt) / 1000)}s)`)
      resolve(code ?? 1)
    })
  })

console.log(`[shard] ${shards} 分片 × ${workers} worker（共 ${shards * (workers + 1)} 线程含各自主 Vite）`)
const startedAt = Date.now()
const shardCodes = await Promise.all(
  Array.from({ length: shards }, (_, i) =>
    run(['run', '--shard=' + (i + 1) + '/' + shards, '--reporter=blob', '--maxWorkers=' + String(workers)], (i + 1) + '/' + shards),
  ),
)
const mergeCode = await run(['run', '--merge-reports'], 'merge')
const failed = shardCodes.some((code) => code !== 0)
console.log(`[shard] 总墙钟 ${Math.round((Date.now() - startedAt) / 1000)}s`)
process.exit(failed ? mergeCode || 1 : mergeCode)
