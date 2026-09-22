// VariantChainModal.tsx — 变体簇溯源（绘梦阶段二刀 D，规格
// 进度计划/gaea-variant-chain-20260923.md）：编辑/扩图产物沿台账 parent_id
// 上溯成链（根→…→当前），每项缩略图+模型/时间/指令摘要；「用到画布」把任一
// 祖先并入画布=回到该变体继续改。链断（祖先缺档/导入图）如实标注不造假。
import { softTextStyle } from '../../utils/uiStyles'
import React, { useEffect, useState } from 'react'
import { Alert, Button, Modal, Spin, Tag, Typography } from 'antd'
import { imageHubAssets, readFileAsDataURL } from '../../api/image'
import type { GenResult } from './types'

interface ChainEntry {
  id: string
  path?: string
  model?: string
  createdAt?: string
  prompt?: string
  image?: string
  current?: boolean
}

export default function VariantChainModal({ open, assetId, onClose, onApply }: {
  open: boolean
  /** 当前编辑产物（链尾）的台账条目 id。 */
  assetId: string
  onClose: () => void
  /** 「用到画布」：把选中条目并入画布与历史（页面级镜像）。 */
  onApply: (r: GenResult) => void
}) {
  const [chain, setChain] = useState<ChainEntry[]>([])
  const [broken, setBroken] = useState(false)
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (!open || !assetId) return
    let cancelled = false
    setLoading(true)
    setChain([])
    setBroken(false)
    void (async () => {
      try {
        const entries = await imageHubAssets('', 'imagegen', 500)
        const byId = new Map(entries.map(e => [e.id ?? '', e]))
        // 沿 parent_id 上溯（防环：上限 64 代）
        const list: ChainEntry[] = []
        let cur = byId.get(assetId)
        const seen = new Set<string>()
        while (cur && !cancelled) {
          const id = cur.id ?? ''
          if (!id || seen.has(id)) break
          seen.add(id)
          list.push({
            id, path: cur.path, model: cur.model, createdAt: cur.created_at,
            prompt: cur.prompt_truncate,
            current: id === assetId,
          })
          const pid = cur.parent_id
          if (!pid) break
          const parent = byId.get(pid)
          if (!parent) { setBroken(true); break } // 祖先缺档：诚实断链标注
          cur = parent
        }
        if (cancelled) return
        // 根→…→当前（倒序展示为时间正序）
        list.reverse()
        setChain(list)
        // 缩略图懒加载（失败留空位不阻塞链展示）
        await Promise.all(list.map(async e => {
          if (!e.path) return
          try {
            const dataUrl = await readFileAsDataURL(e.path)
            if (!cancelled) setChain(prev => prev.map(x => (x.id === e.id ? { ...x, image: dataUrl } : x)))
          } catch { /* 读图失败留空位 */ }
        }))
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => { cancelled = true }
  }, [open, assetId])

  const apply = (e: ChainEntry) => {
    if (!e.image) return
    onApply({
      image: e.image, seed: 0, time: 0,
      prompt: e.prompt || '（变体簇条目）',
      model: e.model || '', size: '',
      file_path: e.path, asset_id: e.id,
    })
    onClose()
  }

  return (
    <Modal
      open={open} title="变体溯源（同源编辑链）" width={560} destroyOnHidden
      footer={<Button key="close" size="small" onClick={onClose}>关闭</Button>}
      onCancel={onClose}
    >
      {loading ? (
        <div style={{ padding: 32, textAlign: 'center' }}><Spin /><div style={{ ...softTextStyle, marginTop: 6 }}>正在梳理变体链…</div></div>
      ) : chain.length === 0 ? (
        <Typography.Text type="secondary">未找到变体记录</Typography.Text>
      ) : (
        <div data-testid="variant-chain-list" style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {chain.map((e, i) => (
            <div key={e.id} data-testid={`variant-chain-item-${i}`}
              style={{ display: 'flex', gap: 10, alignItems: 'center', padding: '6px 8px', borderRadius: 8,
                border: e.current ? '1px solid rgba(22,119,255,0.45)' : '1px solid rgba(127,127,127,0.25)',
                background: e.current ? 'rgba(22,119,255,0.06)' : 'transparent' }}>
              <div style={{ width: 64, height: 64, borderRadius: 6, overflow: 'hidden', background: 'rgba(0,0,0,0.06)',
                display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                {e.image
                  ? <img src={e.image} alt="变体" style={{ maxWidth: '100%', maxHeight: '100%', objectFit: 'contain' }} />
                  : <span style={{ ...softTextStyle, fontSize: 11 }}>无预览</span>}
              </div>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
                  <Typography.Text strong style={{ fontSize: 13 }}>第 {i + 1} 代</Typography.Text>
                  {e.current && <Tag color="blue" style={{ marginRight: 0 }}>当前</Tag>}
                  {i === 0 && <Tag style={{ marginRight: 0 }}>源图</Tag>}
                </div>
                <div style={{ ...softTextStyle, fontSize: 12, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  {e.model || '—'} · {e.createdAt || '—'} · {e.prompt || '（无指令记录）'}
                </div>
              </div>
              {!e.current && (
                <Button size="small" data-testid={`variant-chain-apply-${i}`} disabled={!e.image}
                  onClick={() => apply(e)}>用到画布</Button>
              )}
            </div>
          ))}
          {broken && (
            <Alert type="info" showIcon data-testid="variant-chain-broken"
              message="更早的祖先不在台账中（跨空间或导入图）——链条到此为止。" />
          )}
        </div>
      )}
    </Modal>
  )
}
