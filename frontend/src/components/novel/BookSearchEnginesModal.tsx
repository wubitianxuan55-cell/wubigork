// BookSearchEnginesModal.tsx — 泛搜索引擎规则编辑器（t3 余项）。
// 引擎规则是用户数据资产（%配置目录%/gaea/booksource/rules/websearch-engines.json，
// 模板即示例），此前只能手改 JSON——本编辑器给 启用/名称/地址/查询塑形/选择器
// 的结构化编辑；未知高级字段（link/linkParam/redirectHosts/crawl）按行原样保留
// 不丢失。保存走后端整体替换：逐条 fail-closed 校验（CSS only、禁 @js:）+
// 临时文件原子落盘；规则每次搜索现读，保存即生效，无缓存失效面。
import React, { useCallback, useEffect, useRef, useState } from 'react'
import { Button, Input, Modal, Space, Switch, Typography, message } from 'antd'
import { DeleteOutlined, PlusOutlined } from '@ant-design/icons'
import { app } from '../../gaea/lib/bridge'
import type { NovelBookSourceEngineRule } from '../../gaea/lib/bridge/novel'
import { C } from '../../utils/theme'
import { confirmDiscard } from './unsavedGuard'

interface BookSearchEnginesModalProps {
  open: boolean
  onClose: () => void
}

const TEMPLATE: NovelBookSourceEngineRule = {
  name: '新引擎',
  url: 'https://example.com/search?q=%s',
  queryFormat: '%s 小说 免费阅读',
  result: '.result-item',
  title: 'a.title',
}

const secondaryStyle = { color: C('color-text-secondary'), fontSize: 12 } as const
const inputStyle = { background: C('color-bg-layout'), borderColor: C('color-border'), color: C('color-text') }

const fieldLabel = (text: string) => (
  <Typography.Text style={{ ...secondaryStyle, display: 'block', marginBottom: 2 }}>{text}</Typography.Text>
)

/** 逐行基线（按引擎名索引）：判断用户是否改过任何字段。 */
type EngineBaseline = Record<string, string>

/** 行 JSON 指纹。字段顺序稳定（patch 用 `{...r, [key]: v}` 展开原对象），
 *  故 JSON.stringify 可直接作为「有没有改动」的判据。 */
const rowFingerprint = (r: NovelBookSourceEngineRule): string => JSON.stringify(r)

/** 泛搜索引擎规则编辑器：结构化编辑 + 整体替换保存（后端 fail-closed 校验）。 */
const BookSearchEnginesModal: React.FC<BookSearchEnginesModalProps> = ({ open, onClose }) => {
  const [rows, setRows] = useState<NovelBookSourceEngineRule[]>([])
  const [path, setPath] = useState('')
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  // v4.425（B8）：打开即 reload() 覆盖 rows，关闭又无脏检查 → 编辑直接蒸发。
  // 用「首次 reload 的逐行快照」当基线判脏；行在编辑中改名会让 key 对不上，
  // 按索引兜底，避免改名后被误判为「没改过」而静默丢弃。
  const baselineRef = useRef<EngineBaseline | null>(null)
  const baselineOrderRef = useRef<string[]>([])

  const isRowDirty = (r: NovelBookSourceEngineRule, idx: number): boolean => {
    const base = baselineRef.current
    if (!base) return false
    const before = base[String(r.name)] ?? base[baselineOrderRef.current[idx] ?? '']
    if (before === undefined) return true // 新增行：基线里查无此条
    return before !== rowFingerprint(r)
  }
  const dirty = baselineRef.current !== null && rows.some((r, idx) => isRowDirty(r, idx))

  const reload = useCallback(async () => {
    setLoading(true)
    try {
      const res = await app.NovelBookSourceEnginesGet()
      const list = res.rules ?? []
      setRows(list)
      setPath(res.path ?? '')
      const base: EngineBaseline = {}
      for (const r of list) base[String(r.name)] = rowFingerprint(r)
      baselineRef.current = base
      baselineOrderRef.current = list.map((r) => String(r.name))
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '读取引擎规则失败')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (open) void reload()
  }, [open, reload])

  const patch = (idx: number, key: keyof NovelBookSourceEngineRule, value: unknown) => {
    setRows((rs) => rs.map((r, i) => (i === idx ? { ...r, [key]: value } : r)))
  }

  const handleSave = async () => {
    if (saving) return
    setSaving(true)
    try {
      const n = await app.NovelBookSourceEnginesSave(JSON.stringify(rows))
      message.success(`已保存 ${n} 个搜索引擎，下次搜索即生效`)
      onClose()
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '保存失败（规则未写入）')
    } finally {
      setSaving(false)
    }
  }

  /** 关闭入口（✕/Esc 与「取消」同义）：有未保存编辑先问过；保存在途禁止关闭。 */
  const requestClose = () => {
    if (saving) {
      message.warning('规则保存中，请稍候再关闭')
      return
    }
    if (!dirty) { onClose(); return }
    confirmDiscard({
      title: '引擎规则有未保存的修改',
      message: '关闭会丢弃这些编辑（规则文件未写入，下次搜索仍用旧规则）。',
      discardLabel: '放弃修改',
      onDiscard: onClose,
    })
  }

  return (
    <Modal
      title={<span style={{ color: C('color-text') }}>搜索引擎规则</span>}
      open={open}
      onCancel={requestClose}
      confirmLoading={saving}
      maskClosable={!saving}
      closable={!saving}
      keyboard={!saving}
      width={760}
      footer={[
        <Button key="add" icon={<PlusOutlined aria-hidden />} onClick={() => setRows((rs) => [...rs, { ...TEMPLATE, name: `新引擎 ${rs.length + 1}` }])}>
          新增引擎
        </Button>,
        <Button key="save" type="primary" loading={saving} disabled={loading} onClick={() => void handleSave()}>
          保存
        </Button>,
      ]}
      destroyOnHidden
      transitionName=""
      maskTransitionName=""
      styles={{ body: { background: 'transparent' }, header: { background: 'transparent' } }}
    >
      <Space direction="vertical" size={8} style={{ width: '100%' }}>
        <Typography.Text style={secondaryStyle}>
          SERP 结构随时间漂移，搜索不出结果时改「结果条目/标题链接」选择器或换启用别的引擎。保存即整体替换并逐条校验（禁 @js:/XPath）。
        </Typography.Text>
        <Typography.Text style={secondaryStyle}>规则文件：{path}</Typography.Text>

        {rows.map((r, idx) => (
          <div key={idx} style={{ border: `1px solid ${C('color-border')}`, borderRadius: 8, padding: '8px 12px' }}>
            <Space size={8} align="center" wrap style={{ width: '100%' }}>
              <Switch
                checked={!r.disabled}
                checkedChildren="启用"
                unCheckedChildren="停用"
                onChange={(v) => patch(idx, 'disabled', !v)}
                aria-label={`引擎启用开关 ${r.name}`}
              />
              <Input
                value={r.name}
                onChange={(e) => patch(idx, 'name', e.target.value)}
                style={{ ...inputStyle, width: 120 }}
                placeholder="名称"
                aria-label={`引擎名称 ${idx + 1}`}
              />
              <Input
                value={r.url}
                onChange={(e) => patch(idx, 'url', e.target.value)}
                style={{ ...inputStyle, width: 300 }}
                placeholder="SERP 地址（%s=查询词）"
                aria-label={`引擎地址 ${idx + 1}`}
              />
              <Input
                value={r.queryFormat ?? ''}
                onChange={(e) => patch(idx, 'queryFormat', e.target.value)}
                style={{ ...inputStyle, width: 160 }}
                placeholder="查询塑形（可空）"
                aria-label={`查询塑形 ${idx + 1}`}
              />
              <Button
                danger
                type="text"
                icon={<DeleteOutlined aria-hidden />}
                onClick={() => setRows((rs) => rs.filter((_, i) => i !== idx))}
                aria-label={`删除引擎 ${idx + 1}`}
              />
            </Space>
            <Space size={8} wrap style={{ marginTop: 6 }}>
              <div>
                {fieldLabel('结果条目选择器')}
                <Input
                  value={r.result}
                  onChange={(e) => patch(idx, 'result', e.target.value)}
                  style={{ ...inputStyle, width: 240 }}
                  aria-label={`结果条目选择器 ${idx + 1}`}
                />
              </div>
              <div>
                {fieldLabel('标题链接选择器')}
                <Input
                  value={r.title}
                  onChange={(e) => patch(idx, 'title', e.target.value)}
                  style={{ ...inputStyle, width: 240 }}
                  aria-label={`标题链接选择器 ${idx + 1}`}
                />
              </div>
            </Space>
          </div>
        ))}

        {rows.length === 0 && !loading && (
          <Typography.Text style={secondaryStyle}>还没有引擎规则——新增一个，或先保存一次书源模板再打开。</Typography.Text>
        )}
      </Space>
    </Modal>
  )
}

export default BookSearchEnginesModal
