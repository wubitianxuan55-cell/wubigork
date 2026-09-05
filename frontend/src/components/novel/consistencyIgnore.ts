// consistencyIgnore.ts — 一致性深检「单条忽略」的项目内记忆（localStorage 小模块）。
//
// 设计口径（误报缓解配套：忽略绝不静默吞真问题）：
//   - 存储键：'gaea.novel.consistencyIgnore.v1' → { [projectPath]: string[] }，
//     指纹按项目隔离，仅本项目的检查面板消费；
//   - 指纹 = [category, entity_name, location, reason] 归一空白后拼接。刻意不含
//     description/evidence（其中的章节号会随修改漂移），同一章同一实体的同类
//     告警会一并忽略/恢复；
//   - 每项目指纹上限 500 条（超出丢最旧），localStorage 读写全部 try/catch：
//     隐私模式/配额损坏时忽略记忆整体失效但不影响检查主流程；
//   - 恢复入口：clearIgnoredIssues 清空该项目全部忽略记录（面板「恢复显示」），
//     被忽略条目以计数横幅保持可见。

const IGNORE_KEY = 'gaea.novel.consistencyIgnore.v1'
const MAX_PER_PROJECT = 500

/** 参与指纹的字段（ConsistencyDeepIssue 的子集，便于纯函数测试构造） */
export interface IgnoreFingerprintSource {
  category: string
  entity_name: string
  location: string
  reason?: string
}

/** 告警指纹：类别 + 实体 + 位置 + 缓解原因（不含会漂移的描述/证据文本） */
export function deepIssueFingerprint(iss: IgnoreFingerprintSource): string {
  const norm = (s: unknown) => (typeof s === 'string' ? s.trim() : '')
  return [norm(iss.category), norm(iss.entity_name), norm(iss.location), norm(iss.reason)].join('|')
}

function readStore(): Record<string, string[]> {
  try {
    const raw = window.localStorage.getItem(IGNORE_KEY)
    if (!raw) return {}
    const parsed: unknown = JSON.parse(raw)
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return {}
    const out: Record<string, string[]> = {}
    for (const [k, v] of Object.entries(parsed as Record<string, unknown>)) {
      if (Array.isArray(v)) {
        out[k] = v.filter((x): x is string => typeof x === 'string')
      }
    }
    return out
  } catch {
    return {}
  }
}

function writeStore(store: Record<string, string[]>): void {
  try {
    window.localStorage.setItem(IGNORE_KEY, JSON.stringify(store))
  } catch {
    // 写失败（隐私模式/配额）静默：忽略记忆是体验增强，不阻塞检查主流程
  }
}

/** 读某项目已忽略的指纹列表（未打开项目/无记录返回空数组） */
export function loadIgnoredFingerprints(projectPath: string): string[] {
  if (!projectPath) return []
  return readStore()[projectPath] ?? []
}

/** 该告警是否已被本项目忽略 */
export function isIssueIgnored(projectPath: string, iss: IgnoreFingerprintSource): boolean {
  return loadIgnoredFingerprints(projectPath).includes(deepIssueFingerprint(iss))
}

/** 忽略单条告警并记忆（同指纹去重；超上限丢最旧） */
export function ignoreIssue(projectPath: string, iss: IgnoreFingerprintSource): void {
  if (!projectPath) return
  const store = readStore()
  const list = store[projectPath] ?? []
  const fp = deepIssueFingerprint(iss)
  if (list.includes(fp)) return
  list.push(fp)
  store[projectPath] = list.length > MAX_PER_PROJECT ? list.slice(list.length - MAX_PER_PROJECT) : list
  writeStore(store)
}

/** 恢复显示：清空该项目的全部忽略记录 */
export function clearIgnoredIssues(projectPath: string): void {
  if (!projectPath) return
  const store = readStore()
  if (!(projectPath in store)) return
  delete store[projectPath]
  writeStore(store)
}
