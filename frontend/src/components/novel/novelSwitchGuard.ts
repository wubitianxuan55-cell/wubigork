/**
 * novelSwitchGuard — 小说板块「跨页切书」未保存保护 + 脏状态登记（共享件）。
 *
 * ## 为什么需要（这是小说板块最后一条会丢作者正文的路径）
 * **切书动作由书架页（`pages/HomePage.tsx`）发起，脏状态却分别在创作页与阅读页
 * 内部**——两页都是「受控缓冲 + 显式保存钮」，没有任何自动保存。v4.421.0 的
 * 未保存保护只覆盖了**页内**动作（切章/生成/删除/重写/关标签），跨页这一层当时
 * 明确列为观察池（规格 §2「跨页切书确认（需跨页共享脏态）」），只做了两步退让：
 *   - 创作页 `CreatePage.tsx:395-401`：切完**事后**提示「上一本未保存的正文修改未保留」；
 *   - 阅读页 `ChapterPage.tsx:279-297`：`projectPath` effect 直接 `setTabs([])`，
 *     **连提示都没有**——多标签里未保存的正文静默蒸发。
 *
 * ## 形态：登记 + 单一拦截点
 * - 脏状态持有者（创作页 / 阅读页）用 `registerNovelDirtyProvider` 登记一个
 *   「现在脏不脏 / 能不能先保存」的探针，卸载即注销（按 id 幂等，重复挂载不会重复问）。
 * - 切书发起方（书架）用 `guardNovelSwitch(proceed, targetLabel)` 包住**真正的**
 *   切书动作：无脏直接放行；有脏且**都能先保存** → 三选（先保存 / 放弃修改 / 取消）；
 *   否则 → 两选（放弃修改并切换 / 取消）。
 *
 * ## 语义纪律（沿用 `unsavedGuard` 的既有纪律）
 * **✕/Esc 一律＝取消**，绝不把破坏性分支挂在 `Modal.confirm` 的 `onCancel` 上。
 */

import { message } from 'antd'
import { chooseAction, chooseUnsavedAction } from './unsavedGuard'

export interface NovelDirtyProvider {
  /** 稳定标识（同一持有者重复登记以最后一次为准，卸载注销） */
  id: string
  /** 用户可读的「脏的是什么」，用于确认文案（如「创作页正文」「阅读页 2 个未保存章节」） */
  label: () => string
  /** 现在是否有未保存内容 */
  dirty: () => boolean
  /** 是否具备「先保存」能力（如阅读页多个未保存标签无法一次存齐 → 返回 false） */
  canSave?: () => boolean
  /** 保存并返回是否成功；失败必须返回 false（切换会被中止并如实报错） */
  save?: () => Promise<boolean>
}

const providers = new Map<string, NovelDirtyProvider>()

/** 登记脏状态探针；返回注销函数（组件卸载时调用）。 */
export function registerNovelDirtyProvider(p: NovelDirtyProvider): () => void {
  providers.set(p.id, p)
  return () => { if (providers.get(p.id) === p) providers.delete(p.id) }
}

/** 测试用：清空登记表与确认记账。 */
export function resetNovelDirtyProviders(): void {
  providers.clear()
  confirmedGen = 0
  seenGen.clear()
}

function safeDirty(p: NovelDirtyProvider): boolean {
  try {
    return p.dirty()
  } catch {
    // 探针自身抛错一律按「不脏」处理（fail-open，并留一眼日志）：按脏处理会把作者
    // 永久锁在「切不了书」且弹窗给不出理由——比漏一次提示更糟。真实缺陷应在测试里暴露。
    console.warn(`[novelSwitchGuard] dirty 探针抛错，按不脏处理：${p.id}`)
    return false
  }
}

/** 当前所有「脏」的持有者（无脏即空数组）。 */
export function dirtyNovelProviders(): NovelDirtyProvider[] {
  return [...providers.values()].filter(safeDirty)
}

/** 「放弃修改并切换」已确认的代数（每次确认 +1）。 */
let confirmedGen = 0
/** 各消费者已经看过哪一代（按消费者 id 记账，互不干扰）。 */
const seenGen = new Map<string, number>()

/** 读取并登记「本消费者已看过这次放弃确认」——每个消费者每次确认只返回一次 true。 */
export function takeDiscardConfirmed(consumerId: string): boolean {
  if (confirmedGen === 0) return false
  if (seenGen.get(consumerId) === confirmedGen) return false
  seenGen.set(consumerId, confirmedGen)
  return true
}

function describe(list: NovelDirtyProvider[]): string {
  const labels = list.map((p) => {
    try { return p.label() } catch { return p.id }
  })
  return labels.filter(Boolean).join('、')
}

/**
 * 切书前的统一闸门。`proceed` 是真正的切书动作。
 *
 * @returns `true` = **已拦截**（确认弹窗已弹出，`proceed` 未执行）；
 *          `false` = 无未保存内容，调用方应继续执行 `proceed`。
 *
 * 用法（书架页）：
 * ```ts
 * if (guardNovelSwitch(() => void doSwitch(), card.title)) return
 * void doSwitch()
 * ```
 */
export function guardNovelSwitch(proceed: () => void, targetLabel?: string): boolean {
  const dirty = dirtyNovelProviders()
  if (dirty.length === 0) return false

  const what = describe(dirty)
  const where = targetLabel ? `切换到「${targetLabel}」` : '切换小说'
  const savable = dirty.every((p) => !!p.save && (p.canSave ? p.canSave() : true))
  const discard = () => {
    confirmedGen += 1
    proceed()
  }

  if (savable) {
    chooseUnsavedAction({
      title: '有未保存的内容',
      message: `${where}会丢弃${what}。先保存，或放弃这些修改？`,
      saveLabel: '先保存再切换',
      discardLabel: '放弃修改并切换',
      onSave: () => {
        void (async () => {
          for (const p of dirty) {
            const ok = await p.save!()
            if (!ok) {
              message.error(`${what}保存失败，已取消切换`)
              return
            }
          }
          proceed()
        })()
      },
      onDiscard: discard,
    })
    return true
  }

  chooseAction({
    title: '有未保存的内容',
    message: `${where}会丢弃${what}（这些内容无法一次性自动保存，请切回对应页手动保存）。`,
    options: [{ label: '放弃修改并切换', tone: 'danger', run: discard }],
  })
  return true
}
