// useLegacyWriteBack — 副本→库回写域：两段式（只读预览→非空冲突逐字段确认）
// 与覆盖清单收集。跨域依赖（引用表刷新）显式注入。
// 2026-10-04 自 CharacterPage.tsx hook 分域首刀逐字搬迁。
import { useState } from 'react'
import { message } from 'antd'
import { importProjectCharacters, previewProjectImport, type ImportPreview } from '../../api/characterlib'
import { errText } from '../../utils/errText'

export interface LegacyWriteBackDeps {
  loadRefs: () => Promise<void>
}

export function useLegacyWriteBack(deps: LegacyWriteBackDeps) {
  const [wbOpen, setWbOpen] = useState(false)
  const [wbBusy, setWbBusy] = useState(false)
  const [wbPreview, setWbPreview] = useState<ImportPreview | null>(null)
  const [wbChecked, setWbChecked] = useState<Record<string, true>>({})

  const handleImportLegacy = async () => {
    setWbBusy(true)
    try {
      // 两段式回写：先只读预览——有非空冲突才弹逐字段确认，否则直接安全回写
      const pv = await previewProjectImport()
      if (pv.conflicts.length === 0) {
        await doWriteBack({})
        return
      }
      setWbChecked({})
      setWbPreview(pv)
      setWbOpen(true)
    } catch (err: unknown) {
      message.error(errText(err, '回写预览失败'))
    } finally {
      setWbBusy(false)
    }
  }

  const doWriteBack = async (overwrites: Record<string, string[]>) => {
    try {
      const { imported, filled, overwritten } = await importProjectCharacters(overwrites)
      await deps.loadRefs()
      if (imported === 0 && filled === 0 && overwritten === 0) {
        message.info('本书角色与角色库已一致，无需回写')
        return
      }
      const parts = [`新迁入 ${imported} 个`]
      if (filled > 0) parts.push(`补全 ${filled} 个角色的空缺设定`)
      if (overwritten > 0) parts.push(`按确认覆盖 ${overwritten} 处已有设定`)
      message.success(`回写完成：${parts.join('，')}（本书此后只引用角色库）`)
    } catch (err: unknown) {
      message.error(errText(err, '回写失败'))
    }
  }

  /** 弹窗里逐字段勾选 → 覆盖清单；只统计勾选键，未勾选一律保持库内原值 */
  const collectOverwrites = (): Record<string, string[]> => {
    const ov: Record<string, string[]> = {}
    for (const c of wbPreview?.conflicts ?? []) {
      if (wbChecked[`${c.characterId}::${c.field}`]) {
        if (!ov[c.characterId]) ov[c.characterId] = []
        ov[c.characterId].push(c.field)
      }
    }
    return ov
  }

  return {
    wbOpen, setWbOpen, wbBusy, wbPreview, wbChecked, setWbChecked,
    handleImportLegacy, doWriteBack, collectOverwrites,
  }
}
