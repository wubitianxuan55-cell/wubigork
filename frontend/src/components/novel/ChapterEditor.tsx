import React, { useState } from 'react'
import { Button, Space, Tag, Input, Typography, message } from 'antd'
import { PlusOutlined, DeleteOutlined, EditOutlined, ColumnWidthOutlined, RedoOutlined } from '@ant-design/icons'
import type { ChapterTabData } from '../../types'
import GhostText from './editor/GhostText'
import CommandBar from './editor/CommandBar'
import { C } from '../../utils/theme'
import { Z_INDEX } from '../../utils/zIndex'

interface ChapterEditorProps {
  tab: ChapterTabData
  onUpdate: <K extends keyof ChapterTabData>(field: K, value: ChapterTabData[K]) => void
  sceneTextareaRefs: React.MutableRefObject<Map<number, HTMLTextAreaElement>>
  ghostEnabled: boolean
}

/**
 * ChapterEditor — 章节场景多文本框编辑区
 * 包含场景新增/删除、右键菜单 AI 操作、Cmd+K 命令面板、GhostText
 */
const ChapterEditor: React.FC<ChapterEditorProps> = ({ tab, onUpdate, sceneTextareaRefs, ghostEnabled }) => {
  const [ctxMenu, setCtxMenu] = useState<{ x: number; y: number; text: string } | null>(null)
  const [cmdKVisible, setCmdKVisible] = useState(false)
  const [cmdKText, setCmdKText] = useState('')
  const lastSelectedText = React.useRef('')

  // 全局点击关闭右键菜单
  React.useEffect(() => {
    const close = () => setCtxMenu(null)
    document.addEventListener('click', close)
    return () => document.removeEventListener('click', close)
  }, [])

  const addScene = () => {
    onUpdate('scenes', [...tab.scenes, ''])
  }

  const removeScene = (i: number) => {
    if (tab.scenes.length <= 1) return
    onUpdate('scenes', tab.scenes.filter((_scene: string, j: number) => j !== i))
  }

  const updateScene = (i: number, val: string) => {
    const s = [...tab.scenes]
    s[i] = val
    onUpdate('scenes', s)
    onUpdate('saved', false)
  }

  const onSceneContextMenu = (e: React.MouseEvent<HTMLTextAreaElement>) => {
    const sel = window.getSelection()?.toString().trim()
    if (!sel || sel.length < 10) return
    e.preventDefault()
    lastSelectedText.current = sel
    setCtxMenu({ x: e.clientX, y: e.clientY, text: sel })
  }

  function triggerAI(action: 'describe' | 'expand' | 'rewrite') {
    const text = lastSelectedText.current
    if (!text) return
    setCtxMenu(null)
    const prompts: Record<string, string> = {
      describe: `请对以下段落进行「丰富描写」——增加感官细节（视觉/听觉/触觉/嗅觉），用身体反应替代情绪词（如「他握紧拳头」替代「他很生气」），让读者能「看到」画面。保持原意和风格不变。\n\n原文:\n\`\`\`\n${text}\n\`\`\``,
      expand: `请对以下段落进行「场景扩展」——在不改变情节走向的前提下，扩写为更丰富的场景。增加对话、动作细节、环境描写、内心活动。扩写后长度约为原来的 1.5-2 倍。\n\n原文:\n\`\`\`\n${text}\n\`\`\``,
      rewrite: `请重写以下段落——改进句式多样性（短句穿插长句），去除 AI 套话（「总之」「此外」「不仅如此」），让文字更自然、更像真人写的。保持原意和情节不变。\n\n原文:\n\`\`\`\n${text}\n\`\`\``,
    }
    // 通过 autoSendMsg 传递到 ChatPanel
    const ev = new CustomEvent('ai-assist-send', { detail: prompts[action] })
    window.dispatchEvent(ev)
  }

  return (
    <>
      {/* 质量检查重试状态 */}
      {tab.retryStatus && (
        <div style={{
          marginBottom: 12, padding: '8px 16px',
          background: 'rgba(245, 158, 11, 0.1)',
          border: '1px solid rgba(245, 158, 11, 0.3)',
          borderRadius: 'var(--radius-md)',
          display: 'flex', alignItems: 'center', gap: 8,
          fontSize: 12, color: '#f59e0b',
        }}>
          <span>AI 审稿评分 {tab.retryStatus.score}/10（目标 ≥{tab.retryStatus.target}），正在根据修改建议重写...</span>
        </div>
      )}

      {/* 多文本框 */}
      <div style={{ flex: 1, overflow: 'auto', padding: 12 }}>
        {tab.generating && tab.scenes[0] === '' ? (
          <div style={{ textAlign: 'center', padding: '40px 0' }}>
            <div style={{ color: 'var(--color-text-secondary)' }}>AI 正在创作第{tab.chapterNum}章...</div>
          </div>
        ) : (
          <div>
            {tab.scenes.map((scene: string, i: number) => (
              <div key={i} style={{ marginBottom: 16 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 }}>
                  <Tag style={{ fontSize: 10 }}>场景 {i + 1}</Tag>
                  <Space size={2}>
                    <Button type="text" size="small" icon={<PlusOutlined />} style={{ color: C('color-text-secondary'), fontSize: 10, padding: '0 4px' }} onClick={addScene} />
                    <Button type="text" size="small" danger icon={<DeleteOutlined />} style={{ fontSize: 10, padding: '0 4px' }} onClick={() => removeScene(i)} disabled={tab.scenes.length <= 1} />
                  </Space>
                </div>
                <div style={{ position: 'relative' }}>
                  <Input.TextArea
                    value={scene}
                    onChange={(e) => updateScene(i, e.target.value)}
                    onContextMenu={onSceneContextMenu}
                    className="writing-textarea"
                    autoSize={{ minRows: 4, maxRows: 20 }}
                    ref={(el) => {
                      const ta = (el as any)?.resizableTextArea?.textArea as HTMLTextAreaElement
                      if (ta) sceneTextareaRefs.current.set(i, ta)
                    }}
                    onKeyDown={(e: any) => {
                      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
                        e.preventDefault()
                        const ta = e.target as HTMLTextAreaElement
                        const selected = ta.value.slice(ta.selectionStart, ta.selectionEnd)
                        if (selected) { setCmdKText(selected); setCmdKVisible(true) }
                      }
                    }}
                  />
                  <GhostText
                    enabled={ghostEnabled}
                    getCursorContext={() => {
                      const ta = sceneTextareaRefs.current.get(i)
                      if (!ta) return null
                      return { textBeforeCursor: ta.value.slice(0, ta.selectionStart), textareaElement: ta }
                    }}
                  />
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Cmd+K AI 编辑 */}
      {cmdKVisible && (
        <CommandBar
          selectedText={cmdKText}
          onAccept={(editedText) => {
            const ta = document.activeElement as HTMLTextAreaElement
            if (ta?.tagName === 'TEXTAREA') {
              const start = ta.selectionStart
              const end = ta.selectionEnd
              ta.value = ta.value.slice(0, start) + editedText + ta.value.slice(end)
              ta.dispatchEvent(new Event('input', { bubbles: true }))
              message.success('已应用编辑')
            }
            setCmdKVisible(false)
          }}
          onClose={() => setCmdKVisible(false)}
        />
      )}

      {/* 右键菜单浮层 */}
      {ctxMenu && (
        <div
          style={{
            position: 'fixed', left: ctxMenu.x, top: ctxMenu.y, zIndex: Z_INDEX.CONTEXT_MENU,
            background: C('color-bg-container'), border: '1px solid ' + C('color-border'),
            borderRadius: 8, boxShadow: '0 4px 16px rgba(0,0,0,0.5)',
            padding: '4px 0', minWidth: 180,
          }}
          onContextMenu={(e) => e.preventDefault()}
        >
          <div style={{ padding: '4px 12px 6px', borderBottom: '1px solid ' + C('color-border') }}>
            <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 10 }}>
              已选 {ctxMenu.text.length} 字
            </Typography.Text>
          </div>
          <div onClick={() => triggerAI('describe')}
            style={{ padding: '8px 12px', cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 8, fontSize: 13, color: C('color-text') }}
            onMouseEnter={(e) => (e.currentTarget as HTMLElement).style.background = C('color-bg-layout')}
            onMouseLeave={(e) => (e.currentTarget as HTMLElement).style.background = 'transparent'}>
            <EditOutlined style={{ color: '#4ade80' }} /> 丰富描写
          </div>
          <div onClick={() => triggerAI('expand')}
            style={{ padding: '8px 12px', cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 8, fontSize: 13, color: C('color-text') }}
            onMouseEnter={(e) => (e.currentTarget as HTMLElement).style.background = C('color-bg-layout')}
            onMouseLeave={(e) => (e.currentTarget as HTMLElement).style.background = 'transparent'}>
            <ColumnWidthOutlined style={{ color: '#60a5fa' }} /> 扩展场景
          </div>
          <div onClick={() => triggerAI('rewrite')}
            style={{ padding: '8px 12px', cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 8, fontSize: 13, color: C('color-text') }}
            onMouseEnter={(e) => (e.currentTarget as HTMLElement).style.background = C('color-bg-layout')}
            onMouseLeave={(e) => (e.currentTarget as HTMLElement).style.background = 'transparent'}>
            <RedoOutlined style={{ color: '#f59e0b' }} /> 重写此段
          </div>
        </div>
      )}
    </>
  )
}

export default ChapterEditor
