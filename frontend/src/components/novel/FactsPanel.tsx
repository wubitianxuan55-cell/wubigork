import React, { useEffect, useRef, useState } from 'react'
import { Button, Input, InputNumber, Modal, Popconfirm, Space, Table, Tag, Typography, message } from 'antd'
import { C } from '../../utils/theme'
import { app } from '../../gaea/lib/bridge'
import type { NovelFact } from '../../gaea/lib/bridge/novel'

/**
 * 设定事实词典面板（chinese-novelist-skill「设定与事实词典」蒸馏）：
 * 双时间线账本——「读者已知」是读者视角口径上限，「完整真相」未到揭示章
 * 不得提前揭示（章节生成时由后端注入为口径约束）。整表写回（作者审批制），
 * v1 操作面=登记/标记已揭示/删除；改内容=删了重登（词典是短清单，够用）。
 */
const FactsPanel: React.FC = () => {
  const [rows, setRows] = useState<NovelFact[]>([])
  const [loading, setLoading] = useState(true)
  const [dirty, setDirty] = useState(false)
  const [saving, setSaving] = useState(false)
  const [addOpen, setAddOpen] = useState(false)
  const [form, setForm] = useState({ name: '', category: '', firstChapter: 1, readerKnown: '', fullTruth: '', revealChapter: 0 })
  const rowsRef = useRef(rows)
  rowsRef.current = rows

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      try {
        const res = await app.NovelFactsList()
        if (!cancelled) setRows(res.facts ?? [])
      } catch {
        if (!cancelled) setRows([]) // 读失败不阻断面板（保存时如实报错）
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => { cancelled = true }
  }, [])

  const save = async () => {
    if (saving) return
    setSaving(true)
    try {
      await app.NovelFactsSave(JSON.stringify(rowsRef.current))
      setDirty(false)
      void message.success(`已保存 ${rowsRef.current.length} 条设定事实`)
    } catch (e) {
      void message.error(e instanceof Error ? e.message : String(e))
    } finally {
      setSaving(false)
    }
  }

  const addRow = () => {
    const name = form.name.trim()
    const readerKnown = form.readerKnown.trim()
    if (!name || !readerKnown) {
      void message.warning('名称与读者已知为必填')
      return
    }
    setRows((r) => [...r, {
      id: '',
      name,
      category: form.category.trim(),
      first_chapter: form.firstChapter || 0,
      reader_known: readerKnown,
      full_truth: form.fullTruth.trim(),
      reveal_chapter: form.revealChapter || 0,
      revealed: false,
    }])
    setDirty(true)
    setAddOpen(false)
    setForm({ name: '', category: '', firstChapter: 1, readerKnown: '', fullTruth: '', revealChapter: 0 })
  }

  const toggleRevealed = (idx: number) => {
    setRows((r) => r.map((row, i) => (i === idx ? { ...row, revealed: !row.revealed } : row)))
    setDirty(true)
  }

  const removeRow = (idx: number) => {
    setRows((r) => r.filter((_, i) => i !== idx))
    setDirty(true)
  }

  const columns = [
    { title: '名称', dataIndex: 'name', width: 110 },
    {
      title: '类别', dataIndex: 'category', width: 80,
      render: (v: string) => v ? <Tag>{v}</Tag> : <span style={{ color: C('color-text-tertiary') }}>—</span>,
    },
    { title: '读者已知（口径上限）', dataIndex: 'reader_known' },
    {
      title: '完整真相', dataIndex: 'full_truth',
      render: (v: string, row: NovelFact) => {
        if (!v) return <span style={{ color: C('color-text-tertiary') }}>（无隐藏层）</span>
        return row.revealed ? v : <span style={{ color: C('color-text-secondary') }}>{v}</span>
      },
    },
    {
      title: '揭示', dataIndex: 'reveal_chapter', width: 130,
      render: (v: number, row: NovelFact) =>
        row.revealed ? <Tag color="green">已揭示</Tag> : v > 0 ? <Tag color="orange">第{v}章</Tag> : <Tag>未定</Tag>,
    },
    {
      title: '操作', width: 150,
      render: (_: unknown, _row: NovelFact, idx: number) => (
        <Space size={4}>
          <Button size="small" disabled={saving} onClick={() => toggleRevealed(idx)}>
            {rows[idx]?.revealed ? '撤揭示' : '标记已揭示'}
          </Button>
          <Popconfirm title="删除这条设定事实？" onConfirm={() => removeRow(idx)}>
            <Button size="small" danger disabled={saving}>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div data-testid="facts-panel">
      <Space size={8} style={{ marginBottom: 8 }} wrap>
        <Button size="small" data-testid="facts-add" onClick={() => setAddOpen(true)}>登记事实</Button>
        <Button size="small" type="primary" data-testid="facts-save" disabled={!dirty || saving} loading={saving} onClick={() => void save()}>
          保存词典
        </Button>
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          双时间线：未到揭示章的完整真相不会提前写进正文（生成时按口径约束注入）
        </Typography.Text>
      </Space>
      <Table
        size="small"
        rowKey={(_, i) => String(i)}
        loading={loading}
        columns={columns}
        dataSource={rows}
        pagination={false}
        locale={{ emptyText: '尚未登记设定事实' }}
      />

      <Modal
        title={<span style={{ color: C('color-text') }}>登记设定事实</span>}
        open={addOpen}
        onOk={addRow}
        onCancel={() => setAddOpen(false)}
        okText="登记"
        cancelText="取消"
        width={560}
        destroyOnHidden
        transitionName=""
        maskTransitionName=""
        styles={{ body: { background: 'transparent' }, header: { background: 'transparent' } }}
      >
        <Space direction="vertical" size={10} style={{ width: '100%' }}>
          <Input placeholder="名称（必填，如：倒计时）" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
          <Space size={8} wrap>
            <Input placeholder="类别（设定/规则/秘密/身世/金手指…）" style={{ width: 220 }} value={form.category} onChange={(e) => setForm({ ...form, category: e.target.value })} />
            <InputNumber min={0} placeholder="首现章" value={form.firstChapter} onChange={(v) => setForm({ ...form, firstChapter: v ?? 0 })} />
            <InputNumber min={0} placeholder="计划揭示章（0=未定）" value={form.revealChapter} onChange={(v) => setForm({ ...form, revealChapter: v ?? 0 })} />
          </Space>
          <Input.TextArea rows={2} placeholder="读者已知（必填）：读者当前视角知道什么" value={form.readerKnown} onChange={(e) => setForm({ ...form, readerKnown: e.target.value })} />
          <Input.TextArea rows={2} placeholder="完整真相：作者视角的全部事实（无隐藏层可留空）" value={form.fullTruth} onChange={(e) => setForm({ ...form, fullTruth: e.target.value })} />
        </Space>
      </Modal>
    </div>
  )
}

export default FactsPanel
