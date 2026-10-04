// useProtagonistRelations — AI 主角关系批量生成域：批量入口（all 须确认/
// missing 直跑）、单角色入口、进度事件订阅与三态消息。跨域依赖（全量刷新、
// 抽屉快照同步）显式注入。2026-10-04 自 CharacterPage.tsx hook 分域首刀逐字搬迁。
import { useState } from 'react'
import { message, Modal } from 'antd'
import { generateProtagonistRelations } from '../../components/novel/api/character'
import { errText } from '../../utils/errText'
import { subscribeWailsEvent } from '../../gaea/lib/wailsEvents'
import type { CharacterData } from '../../types'

export interface ProtagonistRelationDeps {
  characters: CharacterData[]
  refreshAll: () => Promise<void>
  projectEdit: CharacterData | null
  reloadProjectEdit: (charID: string) => Promise<void>
}

export function useProtagonistRelations(deps: ProtagonistRelationDeps) {
  const [relBusy, setRelBusy] = useState(false)
  const [relProgress, setRelProgress] = useState('')

  // ── AI 主角关系：批量生成「与主角的关系」短语（项目级，进度独立通道）──
  const runProtagonistRelations = async (mode: 'all' | 'missing' | 'one', name = '') => {
    const onProgress = (ev: unknown) => {
      const raw = ev as { detail?: unknown } | null | undefined
      const d = (raw && typeof raw === 'object' && 'detail' in raw && raw.detail ? raw.detail : raw) as { current?: number; total?: number; name?: string } | null | undefined
      if (d && d.current && d.total) setRelProgress(`正在生成 ${d.current}/${d.total}：${d.name || ''}`)
    }
    const off = window.runtime?.EventsOn
      ? subscribeWailsEvent(window.runtime, 'protagonist-relation-progress', onProgress)
      : () => { /* 无 wails runtime（浏览器 mock/测试）：不订阅也不报错 */ }
    try {
      setRelBusy(true)
      setRelProgress('准备中…')
      const res = await generateProtagonistRelations(mode, name)
      const { updated, failed, failNames } = res || {}
      if (failed > 0) {
        message.warning(
          `主角关系生成完成：更新 ${updated} 位，失败 ${failed} 位` +
          (failNames?.length ? `（${failNames.slice(0, 3).join('、')}${failNames.length > 3 ? '…' : ''}）` : ''),
        )
      } else if (updated === 0) {
        message.info('没有需要生成的角色（剩余全部=已都有主角关系；或仅主角本人）')
      } else {
        message.success(`已生成 ${updated} 位角色的主角关系`)
      }
      await deps.refreshAll()
      // 抽屉打开时同步其快照（复用职业链的 reloadProjectEdit，取最新 characters.json 值）
      if (deps.projectEdit?.id) await deps.reloadProjectEdit(deps.projectEdit.id)
    } catch (err: unknown) {
      message.error(`主角关系生成失败：${errText(err, String(err))}`)
    } finally {
      off()
      setRelBusy(false)
      setRelProgress('')
    }
  }

  /** 批量入口：全部=覆盖重写须确认；剩余全部只补空白直接跑 */
  const handleGenRelations = (mode: 'all' | 'missing') => {
    if (!deps.characters.length) return
    if (mode === 'all') {
      Modal.confirm({
        title: 'AI 重写全部角色的主角关系？',
        content: '将为除主角本人外的全部本书角色重新随机「与主角的关系」（已有关系会被覆盖）。角色较多时耗时较长。',
        okText: '开始生成',
        cancelText: '取消',
        onOk: () => { void runProtagonistRelations('all') },
      })
      return
    }
    void runProtagonistRelations('missing')
  }

  return {
    relBusy, relProgress, runProtagonistRelations, handleGenRelations,
  }
}
