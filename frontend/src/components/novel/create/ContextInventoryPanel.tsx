// ContextInventoryPanel.tsx — 上下文清单（v4.441）：NovelContextInventory dry-run
// 的用户可见面（v4.434 绑定首发时零前端消费，「同样信息量下 prompt 更短」至此
// 可量化可看）。恒列「成人向工艺区段」行如实回显档位生效面（v4.439 注入在模板
// 槽里，作者须有处确认）。零生成零写盘。
import React, { useEffect, useState } from 'react'
import { Modal, Table, Typography, message } from 'antd'
import { C } from '../../../utils/theme'
import { app } from '../../../gaea/lib/bridge'

interface InventoryRow {
  name?: string
  runes?: number
  note?: string
  preview?: string
}

export default function ContextInventoryPanel({ open, chapterNum, onClose }: {
  open: boolean
  /** 当前激活章；0=未选章（清单按整书公共区段口径返回） */
  chapterNum: number
  onClose: () => void
}) {
  const [rows, setRows] = useState<InventoryRow[]>([])
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (!open) return
    let cancelled = false
    ;(async () => {
      setLoading(true)
      try {
        const items = (await app.NovelContextInventory(chapterNum)) as InventoryRow[] | null
        if (!cancelled) setRows(items ?? [])
      } catch (e) {
        if (!cancelled) message.error(e instanceof Error ? e.message : '上下文清单读取失败')
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => { cancelled = true }
  }, [open, chapterNum])

  const total = rows.find((r) => r.name?.startsWith('合计'))

  return (
    <Modal
      title={<span style={{ color: C('color-text') }}>上下文清单{chapterNum > 0 ? `（第 ${chapterNum} 章生成口径）` : ''}</span>}
      open={open}
      onCancel={onClose}
      footer={null}
      width={720}
      destroyOnHidden
      transitionName=""
      maskTransitionName=""
      styles={{ body: { background: 'transparent' }, header: { background: 'transparent' } }}
    >
      <Typography.Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 8 }}>
        本章生成将注入的上下文区段（dry-run，零生成零写盘）；字数为 rune 口径，各区段另受分项与总预算约束。
      </Typography.Text>
      <Table<InventoryRow>
        size="small"
        loading={loading}
        rowKey={(r) => r.name ?? String(r.runes)}
        dataSource={rows}
        pagination={false}
        columns={[
          {
            title: '区段', dataIndex: 'name', width: 150,
            render: (v: string | undefined, r) =>
              r.name?.startsWith('合计')
                ? <b style={{ color: C('color-text') }}>{v}</b>
                : <span style={{ color: C('color-text') }}>{v}</span>,
          },
          {
            title: '字数', dataIndex: 'runes', width: 80, align: 'right',
            render: (v: number | undefined, r) =>
              <span style={{ color: r.name?.startsWith('合计') ? C('color-text') : 'var(--color-text-secondary)' }}>{(v ?? 0).toLocaleString()}</span>,
          },
          {
            title: '说明', dataIndex: 'note', width: 190,
            render: (v: string | undefined) => <span style={{ color: 'var(--color-text-secondary)', fontSize: 12 }}>{v}</span>,
          },
          {
            title: '预览', dataIndex: 'preview',
            render: (v: string | undefined) => <span style={{ color: 'var(--color-text-secondary)', fontSize: 12 }}>{v}</span>,
          },
        ]}
      />
      {total ? (
        <Typography.Text type="secondary" style={{ fontSize: 12, display: 'block', marginTop: 8 }}>
          合计 {(total.runes ?? 0).toLocaleString()} rune（不含 setting 槽；setting 由编辑区传入，超预算截断并附提示）。
        </Typography.Text>
      ) : null}
    </Modal>
  )
}
