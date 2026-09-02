// ForeshadowPanel.tsx — 伏笔登记表面板（v4.3f 闭环版）
// 登记→埋设→回收闭环：GetForeshadows 展示 + SaveForeshadows 全量写回。
// ①「登记伏笔」表单（类别/描述/埋设章节/是否长线，manual_ 前缀 ID）；
// ② 每条状态流转按钮（planted→hinted→revealed，revealed 可回退）；
// ③ 删除（confirm）；④ 描述可编辑。操作乐观更新，写回失败回滚并提示。
// 纯逻辑（ID 生成/状态机/载荷收窄）抽在 foreshadowLogic.ts。
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  Alert, Button, Checkbox, Empty, Input, InputNumber, message, Popconfirm, Select, Spin, Tag, Tooltip,
} from 'antd'
import { DeleteOutlined, EditOutlined, FlagOutlined, PlusOutlined, ReloadOutlined } from '@ant-design/icons'
import * as App from '../../../src/wailsjsCompat'
import type { ForeshadowItemData, ForeshadowStatus } from '../../types'
import {
  advanceForeshadowStatus,
  buildManualForeshadow,
  foreshadowFlowLabel,
  normalizeForeshadowItems,
} from './foreshadowLogic'

const STATUS_META: Record<ForeshadowStatus, { label: string; color: string }> = {
  planted: { label: '已埋设', color: 'blue' },
  hinted: { label: '已暗示', color: 'gold' },
  revealed: { label: '已回收', color: 'green' },
}

const CATEGORY_LABELS: Record<string, string> = {
  character: '角色',
  plot: '剧情',
  world: '世界观',
  relationship: '关系',
}

const CATEGORY_OPTIONS = Object.entries(CATEGORY_LABELS).map(([value, label]) => ({ value, label }))

interface ForeshadowPanelProps {
  /** 未打开项目时仅展示空态引导，不触发加载 */
  disabled?: boolean
}

const ForeshadowPanel: React.FC<ForeshadowPanelProps> = ({ disabled }) => {
  const [items, setItems] = useState<ForeshadowItemData[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const loadToken = useRef(0)

  // 登记表单
  const [formOpen, setFormOpen] = useState(false)
  const [category, setCategory] = useState('plot')
  const [description, setDescription] = useState('')
  const [chapterNum, setChapterNum] = useState(1)
  const [isLongTerm, setIsLongTerm] = useState(false)
  // 行内描述编辑
  const [editing, setEditing] = useState<{ id: string; desc: string } | null>(null)

  const load = useCallback(async () => {
    const token = ++loadToken.current
    if (disabled) {
      setItems([])
      setLoading(false)
      setError('')
      return
    }
    setLoading(true)
    setError('')
    try {
      const res = await App.GetForeshadows()
      if (token !== loadToken.current) return
      setItems(normalizeForeshadowItems(res))
    } catch (err: unknown) {
      if (token !== loadToken.current) return
      setError(err instanceof Error ? err.message : '伏笔加载失败')
    } finally {
      if (token === loadToken.current) setLoading(false)
    }
  }, [disabled])

  useEffect(() => { void load() }, [load])

  // persist 全量写回（乐观更新 + 失败回滚提示）
  const persist = useCallback(async (prev: ForeshadowItemData[], next: ForeshadowItemData[]) => {
    setItems(next)
    try {
      await App.SaveForeshadows(JSON.stringify(next))
      return true
    } catch (err: unknown) {
      setItems(prev)
      message.error(`伏笔保存失败，已回滚：${err instanceof Error ? err.message : '未知错误'}`)
      return false
    }
  }, [])

  const register = () => {
    const desc = description.trim()
    if (!desc) {
      message.warning('请填写伏笔描述')
      return
    }
    const entry = buildManualForeshadow({ category, description: desc, chapterNum, isLongTerm })
    void persist(items, [...items, entry])
    setDescription('')
    setIsLongTerm(false)
    setFormOpen(false)
    message.success('伏笔已登记')
  }

  const flow = (id: string) => {
    const target = items.find((it) => it.id === id)
    if (!target) return
    const nextStatus = advanceForeshadowStatus(target.status)
    void persist(items, items.map((it) => (it.id !== id ? it : {
      ...it,
      status: nextStatus,
      revealed_in: nextStatus === 'revealed' ? (it.revealed_in ?? it.planted_in) : undefined,
    })))
  }

  const remove = (id: string) => {
    void persist(items, items.filter((it) => it.id !== id))
  }

  const saveEdit = () => {
    if (!editing) return
    const desc = editing.desc.trim()
    if (!desc) {
      message.warning('描述不能为空')
      return
    }
    const id = editing.id
    setEditing(null)
    void persist(items, items.map((it) => (it.id !== id ? it : { ...it, description: desc })))
  }

  const stats = useMemo(() => {
    const total = items.length
    const revealed = items.filter((i) => i.status === 'revealed').length
    const hinted = items.filter((i) => i.status === 'hinted').length
    const planted = items.filter((i) => i.status === 'planted').length
    const rate = total === 0 ? 0 : Math.round((revealed / total) * 100)
    return { total, revealed, hinted, planted, rate }
  }, [items])

  const flowLegend = (
    <div className="fs-flow-legend">
      <span>埋设</span><span className="fs-arrow">→</span>
      <span>暗示</span><span className="fs-arrow">→</span>
      <span>回收</span>
    </div>
  )

  return (
    <div className="novel-panel fs-panel" style={{ flex: 1, minWidth: 0, minHeight: 0, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
      <div className="novel-panel-head" style={{ flexWrap: 'wrap', rowGap: 4 }}>
        <span className="novel-panel-title"><FlagOutlined />伏笔登记</span>
        <div style={{ flex: 1 }} />
        <span className="novel-setting-meta">
          回收率 {stats.rate}%（{stats.revealed}/{stats.total}）
        </span>
        <Button size="small" icon={<PlusOutlined />} disabled={disabled} onClick={() => setFormOpen((o) => !o)}>
          登记伏笔
        </Button>
        <Button size="small" icon={<ReloadOutlined />} onClick={() => void load()} loading={loading} disabled={disabled}>
          刷新
        </Button>
      </div>
      <div className="novel-setting-body" style={{ padding: 8 }}>
        {disabled ? (
          <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="请先在「书架」打开一部小说项目" style={{ margin: 'auto' }} />
        ) : loading ? (
          <div style={{ margin: 'auto' }}><Spin size="small" /></div>
        ) : error ? (
          <Alert
            type="error" showIcon style={{ width: '100%' }}
            message="伏笔加载失败"
            description={error}
            action={<Button size="small" icon={<ReloadOutlined />} onClick={() => void load()}>重试</Button>}
          />
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', flex: 1, minHeight: 0 }}>
            {/* ① 手工登记表单 */}
            {formOpen && (
              <div className="fs-form" style={{ display: 'flex', flexDirection: 'column', gap: 6, marginBottom: 8, flexShrink: 0 }}>
                <div style={{ display: 'flex', gap: 6, alignItems: 'center', flexWrap: 'wrap' }}>
                  <Select size="small" style={{ width: 96 }} value={category} onChange={setCategory} options={CATEGORY_OPTIONS} />
                  <span className="novel-setting-meta">埋设章节</span>
                  <InputNumber size="small" min={1} max={9999} style={{ width: 72 }} value={chapterNum} onChange={(v) => setChapterNum(Number(v) || 1)} />
                  <Checkbox checked={isLongTerm} onChange={(e) => setIsLongTerm(e.target.checked)} style={{ fontSize: 12 }}>
                    长线伏笔
                  </Checkbox>
                </div>
                <Input.TextArea
                  size="small" rows={2} maxLength={500}
                  placeholder="伏笔描述（如：主角左臂旧伤的来历）"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                />
                <div style={{ display: 'flex', gap: 6, justifyContent: 'flex-end' }}>
                  <Button size="small" type="primary" icon={<FlagOutlined />} onClick={register}>登记</Button>
                  <Button size="small" onClick={() => setFormOpen(false)}>取消</Button>
                </div>
              </div>
            )}
            {items.length === 0 ? (
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description={
                  <span>
                    还没有伏笔登记。生成章节并分析后会自动登记，也可点「登记伏笔」手工记录。
                  </span>
                }
                style={{ margin: 'auto' }}
              />
            ) : (
              <div className="fs-list" style={{ flex: 1, minHeight: 0, overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: 6 }}>
                <div className="fs-stats-row">
                  {flowLegend}
                  <div style={{ flex: 1 }} />
                  <span className="novel-setting-meta">埋设 {stats.planted} · 暗示 {stats.hinted} · 回收 {stats.revealed}</span>
                </div>
                {items.map((it) => (
                  <div key={it.id} className="fs-item">
                    <div className="fs-item-head">
                      <Tag style={{ marginInlineEnd: 6, fontSize: 11 }} color="default">{CATEGORY_LABELS[it.category] || it.category}</Tag>
                      {editing?.id === it.id ? (
                        <Input.TextArea
                          size="small" rows={2} autoFocus style={{ flex: 1, minWidth: 0 }}
                          value={editing.desc}
                          onChange={(e) => setEditing({ id: it.id, desc: e.target.value })}
                        />
                      ) : (
                        <span className="fs-item-desc">{it.description}</span>
                      )}
                      <div style={{ flex: 1 }} />
                      <Tooltip title={`章节：${it.planted_in}`}>
                        <span className="novel-setting-meta">{it.planted_in}</span>
                      </Tooltip>
                      <Tag style={{ marginInlineEnd: 0, fontSize: 11 }} color={STATUS_META[it.status].color}>
                        {STATUS_META[it.status].label}
                      </Tag>
                    </div>
                    <div className="fs-item-foot" style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                      {it.is_long_term && <Tag style={{ fontSize: 10, marginInlineEnd: 0 }} color="purple">长期伏笔</Tag>}
                      {it.status === 'revealed' && it.revealed_in && (
                        <span className="novel-setting-meta">回收于 {it.revealed_in}</span>
                      )}
                      <div style={{ flex: 1 }} />
                      {editing?.id === it.id ? (
                        <>
                          <Button size="small" type="link" style={{ padding: 0 }} onClick={saveEdit}>保存</Button>
                          <Button size="small" type="link" style={{ padding: 0 }} onClick={() => setEditing(null)}>取消</Button>
                        </>
                      ) : (
                        <>
                          {/* ② 状态流转：planted→hinted→revealed，revealed 可回退 */}
                          <Button size="small" type="link" style={{ padding: 0 }} onClick={() => flow(it.id)}>
                            {foreshadowFlowLabel(it.status)}
                          </Button>
                          {/* ④ 描述编辑 */}
                          <Button
                            size="small" type="link" icon={<EditOutlined />}
                            aria-label={`编辑伏笔：${it.description}`}
                            onClick={() => setEditing({ id: it.id, desc: it.description })}
                          />
                          {/* ③ 删除（confirm） */}
                          <Popconfirm title="删除该伏笔？" okText="删除" cancelText="取消" onConfirm={() => remove(it.id)}>
                            <Button
                              size="small" type="link" danger icon={<DeleteOutlined />}
                              aria-label={`删除伏笔：${it.description}`}
                            />
                          </Popconfirm>
                        </>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

export default ForeshadowPanel
