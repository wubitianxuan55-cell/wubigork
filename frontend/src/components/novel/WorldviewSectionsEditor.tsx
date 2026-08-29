// WorldviewSectionsEditor.tsx — 设定页「维度化」编辑器（v4.3e）
// 复用 GetWorldviewSections / SaveAllWorldviewSections 绑定，6 维度分卡片就地编辑、整存。
// 与整篇 Markdown 编辑器共用同一份 worldview.json（后端 SaveWorldview 会把整篇文本
// 写回第一个非 legacy 维度），因此两种保存互相覆盖的风险在 UI 上显式提示。
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  Alert, Button, Collapse, Input, Tag, message,
} from 'antd'
import { CaretRightOutlined, SaveOutlined, ReloadOutlined, AppstoreOutlined } from '@ant-design/icons'
import { countTextChars } from '../../utils/text'
import * as App from '../../../src/wailsjsCompat'
import type { WorldviewSectionData } from '../../types'

/** 后端缺省 6 维度（对齐 internal/worldview GetSections 兜底） */
const DEFAULT_SECTIONS: WorldviewSectionData[] = [
  { id: 'era', title: '时代背景', content: '', order: 1 },
  { id: 'geography', title: '地理风貌', content: '', order: 2 },
  { id: 'factions', title: '势力格局', content: '', order: 3 },
  { id: 'rules', title: '规则体系', content: '', order: 4 },
  { id: 'culture', title: '文化习俗', content: '', order: 5 },
  { id: 'history', title: '历史事件', content: '', order: 6 },
]

/** 归一化绑定返回：{sections:[...]} → WorldviewSectionData[]，缺省补齐 6 维度 */
function normalizeSections(raw: unknown): WorldviewSectionData[] {
  const list = Array.isArray(raw)
    ? raw as WorldviewSectionData[]
    : ((raw as { sections?: WorldviewSectionData[] } | null)?.sections ?? [])
  const byId = new Map(list.map((s) => [s.id, s]))
  const merged = DEFAULT_SECTIONS.map((d) => byId.get(d.id) ?? d)
  for (const s of list) {
    if (!merged.some((m) => m.id === s.id)) merged.push(s)
  }
  return merged.sort((a, b) => (a.order ?? 0) - (b.order ?? 0))
}

interface WorldviewSectionsEditorProps {
  /** 未打开项目时仅展示空态引导，不触发加载 */
  disabled?: boolean
}

const WorldviewSectionsEditor: React.FC<WorldviewSectionsEditorProps> = ({ disabled }) => {
  const [sections, setSections] = useState<WorldviewSectionData[]>(DEFAULT_SECTIONS)
  const [savedSnapshot, setSavedSnapshot] = useState('')
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')
  const [saving, setSaving] = useState(false)
  const [savedAt, setSavedAt] = useState('')
  const loadToken = useRef(0)

  const load = useCallback(async () => {
    const token = ++loadToken.current
    if (disabled) {
      setSections(DEFAULT_SECTIONS)
      setSavedSnapshot('')
      setLoading(false)
      setLoadError('')
      return
    }
    setLoading(true)
    setLoadError('')
    try {
      const res = await App.GetWorldviewSections()
      if (token !== loadToken.current) return
      const list = normalizeSections(res)
      setSections(list)
      setSavedSnapshot(JSON.stringify(list))
    } catch (err: unknown) {
      if (token !== loadToken.current) return
      setLoadError(err instanceof Error ? err.message : '维度加载失败')
      // 降级：保留缺省维度可编辑，避免整页不可用
      setSections((prev) => (prev.length ? prev : DEFAULT_SECTIONS))
    } finally {
      if (token === loadToken.current) setLoading(false)
    }
  }, [disabled])

  useEffect(() => { void load() }, [load])

  const dirty = useMemo(
    () => !disabled && savedSnapshot !== '' && JSON.stringify(sections) !== savedSnapshot,
    [sections, savedSnapshot, disabled],
  )

  const updateContent = useCallback((id: string, content: string) => {
    setSections((prev) => prev.map((s) => (s.id === id ? { ...s, content } : s)))
  }, [])

  const handleSave = useCallback(async () => {
    setSaving(true)
    try {
      await App.SaveAllWorldviewSections(JSON.stringify(sections))
      setSavedSnapshot(JSON.stringify(sections))
      setSavedAt(new Date().toLocaleTimeString())
      message.success('世界观维度已保存')
    } catch (err: unknown) {
      message.error('保存失败: ' + (err instanceof Error ? err.message : String(err)))
    } finally {
      setSaving(false)
    }
  }, [sections])

  return (
    <div className="wv-sections-editor" style={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column', gap: 10 }}>
      <Alert
        type="info"
        showIcon
        banner={false}
        message="两种保存方式共用同一份世界观数据"
        description="「维度化」与「整篇 Markdown」保存到同一份 worldview.json（整篇保存会把全文写回第一个维度），两种方式混用可能互相覆盖。建议固定使用一种方式维护设定。"
        style={{ flexShrink: 0 }}
      />
      <div className="novel-setting-toolbar" style={{ flexShrink: 0 }}>
        <span className="novel-setting-meta">
          <AppstoreOutlined style={{ marginRight: 6, color: 'var(--gaea-glow)' }} />
          六维度世界观
        </span>
        <div style={{ flex: 1 }} />
        <Tag style={{ marginInlineEnd: 8 }} color={dirty ? 'warning' : 'success'}>
          {disabled ? '未打开项目' : dirty ? '维度有未保存修改' : savedAt ? `已保存 ${savedAt}` : '无修改'}
        </Tag>
        <Button size="small" icon={<ReloadOutlined />} onClick={() => void load()} disabled={loading || disabled}>
          重新加载
        </Button>
        <Button size="small" type="primary" icon={<SaveOutlined />} onClick={handleSave} loading={saving} disabled={disabled}>
          保存全部维度
        </Button>
      </div>

      {loading ? (
        <div style={{ margin: 'auto' }}>加载中…</div>
      ) : disabled ? (
        <div className="wv-sections-empty">请先在「书架」打开或创建一部小说项目，再编辑六维度世界观。</div>
      ) : (
        <div style={{ flex: 1, minHeight: 0, overflowY: 'auto' }}>
          {loadError && (
            <Alert
              type="warning"
              showIcon
              message="维度数据加载失败，已降级为默认维度"
              description={loadError}
              action={
                <Button size="small" icon={<ReloadOutlined />} onClick={() => void load()}>重试</Button>
              }
              style={{ marginBottom: 10 }}
            />
          )}
          <Collapse
            className="wv-sections-collapse"
            defaultActiveKey={DEFAULT_SECTIONS.map((s) => s.id)}
            expandIcon={({ isActive }) => <CaretRightOutlined rotate={isActive ? 90 : 0} />}
            items={sections.map((sec) => ({
              key: sec.id,
              label: (
                <span className="wv-section-label">
                  <span className="wv-section-title">{sec.title}</span>
                  <span className="wv-section-chars">
                    {sec.content.trim() ? `${countTextChars(sec.content.trim()).toLocaleString()} 字` : '空'}
                  </span>
                </span>
              ),
              children: (
                <Input.TextArea
                  className="novel-editor wv-section-input"
                  value={sec.content}
                  onChange={(e) => updateContent(sec.id, e.target.value)}
                  placeholder={`撰写「${sec.title}」设定…`}
                  autoSize={{ minRows: 4, maxRows: 16 }}
                  style={{ fontSize: 13 }}
                />
              ),
            }))}
          />
        </div>
      )}
    </div>
  )
}

export default WorldviewSectionsEditor
