import React, { useState } from 'react'
import { Button, Space, Tag, Input, Select, Modal, Typography, message } from 'antd'
import { ArrowUpOutlined, ArrowDownOutlined, PlusOutlined, DeleteOutlined, EditOutlined, ColumnWidthOutlined, RedoOutlined, ThunderboltOutlined, InfoCircleOutlined, EyeOutlined } from '@ant-design/icons'
import { GetChapterScenes, GenerateScene, CreateScene, SaveSceneMeta, ReorderScenes } from '../../../wailsjs/go/app/NovelB'
import SceneBibleDrawer from './SceneBibleDrawer'
import { getCharacters } from './api/character'
import type { ChapterTabData } from '../../types'
import GhostText from './editor/GhostText'
import CommandBar from './editor/CommandBar'
import { C } from '../../utils/theme'
import { countTextChars } from '../../utils/text'
import { Z_INDEX } from '../../utils/zIndex'

interface ChapterEditorProps {
  tab: ChapterTabData
  onUpdate: <K extends keyof ChapterTabData>(field: K, value: ChapterTabData[K]) => void
  sceneTextareaRefs: React.MutableRefObject<Map<number, HTMLTextAreaElement>>
  ghostEnabled: boolean
}

/** 窄化场景未知负载 → 取其 id（非对象/缺 id → undefined）。 */
function sceneIdOf(scene: unknown): string | undefined {
  if (typeof scene === 'object' && scene !== null) {
    const id = (scene as Record<string, unknown>).id
    return typeof id === 'string' ? id : undefined
  }
  return undefined
}

/** 窄化 GenerateScene 返回负载 → 生成后的正文 content。 */
function sceneContentOf(value: unknown): string | undefined {
  if (typeof value === 'object' && value !== null) {
    const c = (value as Record<string, unknown>).content
    return typeof c === 'string' && c.length > 0 ? c : undefined
  }
  return undefined
}

/** 窄化 GenerateScene 返回负载 → aiTaste 分 / deSlop 简讯（供按钮下展示）。 */
function sceneMetaOf(value: unknown): { aiTaste?: number; beforeScore?: number; afterScore?: number; changes?: number } {
  if (typeof value === 'object' && value !== null) {
    const rec = value as Record<string, unknown>
    const aiTaste = typeof rec.aiTaste === 'number' ? rec.aiTaste : undefined
    let beforeScore: number | undefined
    let afterScore: number | undefined
    let changes: number | undefined
    if (typeof rec.deSlop === 'object' && rec.deSlop !== null) {
      const d = rec.deSlop as Record<string, unknown>
      beforeScore = typeof d.beforeScore === 'number' ? d.beforeScore : undefined
      afterScore = typeof d.afterScore === 'number' ? d.afterScore : undefined
      changes = Array.isArray(d.changes) ? d.changes.length : undefined
    }
    return { aiTaste, beforeScore, afterScore, changes }
  }
  return {}
}

/**
 * ChapterEditor — 章节场景多文本框编辑区
 * 包含场景新增/删除、右键菜单 AI 操作、Cmd+K 命令面板、GhostText
 */
const ChapterEditorInner: React.FC<ChapterEditorProps> = ({ tab, onUpdate, sceneTextareaRefs, ghostEnabled }) => {
  const [ctxMenu, setCtxMenu] = useState<{ x: number; y: number; text: string } | null>(null)
  const [cmdKVisible, setCmdKVisible] = useState(false)
  const [cmdKText, setCmdKText] = useState('')
  const lastSelectedText = React.useRef('')
  // 逐场景 AI 生成：sceneIds 与 tab.scenes 按索引对齐（V4 场景制由 ChapterPage
  // 载入时随场景正文一并读入；无 id 的场景按钮禁用）
  const sceneIds = tab.sceneIds ?? []
  const [scenePlots, setScenePlots] = useState<string[]>([])
  const [sceneGen, setSceneGen] = useState<Record<number, { loading: boolean; aiTaste?: number; beforeScore?: number; afterScore?: number; changes?: number }>>({})
  // POV 视图（刀7续收官）：场景「视角」抽屉——场景圣经（已知/不知情对照）。
  const [bibleScene, setBibleScene] = useState<number | null>(null)
  const [addingScene, setAddingScene] = useState(false)
  const [moving, setMoving] = useState(false)

  // ── v4.199 场景元数据（POV/地点/时间/情感）编辑面：每场景 ⓘ 钮 → 弹窗 ──
  interface SceneMetaDraft { title: string; summary: string; povCharId: string; location: string; timeOfDay: string; emotion: string; tags: string; status: string }
  const emptyMeta: SceneMetaDraft = { title: '', summary: '', povCharId: '', location: '', timeOfDay: '', emotion: '', tags: '', status: 'draft' }
  const [metaTarget, setMetaTarget] = useState<{ index: number; sceneId: string } | null>(null)
  const [metaDraft, setMetaDraft] = useState<SceneMetaDraft>(emptyMeta)
  const [metaLoading, setMetaLoading] = useState(false)
  const [metaSaving, setMetaSaving] = useState(false)
  const [povOptions, setPovOptions] = useState<{ value: string; label: string }[]>([])

  const openMeta = async (i: number) => {
    const sceneId = sceneIds[i]
    if (!sceneId) return
    setMetaTarget({ index: i, sceneId })
    setMetaLoading(true)
    setMetaDraft(emptyMeta)
    try {
      const scenes = (await GetChapterScenes(tab.chapterNum)) as Array<Record<string, unknown>>
      const cur = scenes.find((sc) => sc.id === sceneId)
      if (cur) {
        setMetaDraft({
          title: typeof cur.title === 'string' ? cur.title : '',
          summary: typeof cur.summary === 'string' ? cur.summary : '',
          povCharId: typeof cur.povCharId === 'string' ? cur.povCharId : '',
          location: typeof cur.location === 'string' ? cur.location : '',
          timeOfDay: typeof cur.timeOfDay === 'string' ? cur.timeOfDay : '',
          emotion: typeof cur.emotion === 'string' ? cur.emotion : '',
          tags: Array.isArray(cur.tags) ? (cur.tags as string[]).join(',') : '',
          status: typeof cur.status === 'string' ? cur.status : 'draft',
        })
      }
    } catch {
      message.error('场景信息读取失败')
    } finally {
      setMetaLoading(false)
    }
    // POV 候选 = 本书角色（characters.json；POV 圣经同源）。
    try {
      const page = await getCharacters()
      setPovOptions((page.characters ?? []).map((c) => ({ value: c.id, label: c.name })))
    } catch {
      setPovOptions([])
    }
  }

  const saveMeta = async () => {
    if (!metaTarget) return
    setMetaSaving(true)
    try {
      const d = metaDraft
      const payload = {
        title: d.title,
        summary: d.summary,
        povCharId: d.povCharId,
        location: d.location,
        timeOfDay: d.timeOfDay,
        emotion: d.emotion,
        tags: d.tags.split(/[,，]/).map((t) => t.trim()).filter(Boolean),
        status: d.status,
      }
      await SaveSceneMeta(tab.chapterNum, metaTarget.sceneId, JSON.stringify(payload))
      message.success('场景信息已保存（POV 将影响本场景的 AI 生成视角）')
      setMetaTarget(null)
    } catch (e) {
      message.error(String(e))
    } finally {
      setMetaSaving(false)
    }
  }

  // 场景重排：本地框/id 同步换位 + ReorderScenes 落盘（blob 投影由 Go 侧同调用同步）。
  // 乐观换位，失败回滚还原；任一框缺 id（本地降级态/分支章）只换本地不调绑定。
  const moveScene = async (i: number, dir: -1 | 1) => {
    if (moving || tab.sceneBacked !== true) return
    const j = i + dir
    if (j < 0 || j >= tab.scenes.length) return
    const swap = <T,>(arr: T[]): T[] => { const c = [...arr]; c[i] = c[j]; c[j] = arr[i]; return c }
    const ids = tab.sceneIds ?? []
    const canPersist = ids.length === tab.scenes.length && !!ids[i] && !!ids[j]
    onUpdate('scenes', swap(tab.scenes))
    if (ids.length > 0) onUpdate('sceneIds', swap(ids))
    if (!canPersist) return
    setMoving(true)
    try {
      await ReorderScenes(tab.chapterNum, swap(ids))
    } catch {
      onUpdate('scenes', tab.scenes)
      onUpdate('sceneIds', ids)
      message.error('场景排序失败，已还原')
    } finally {
      setMoving(false)
    }
  }

  // 全局点击关闭右键菜单
  React.useEffect(() => {
    const close = () => setCtxMenu(null)
    document.addEventListener('click', close)
    return () => document.removeEventListener('click', close)
  }, [])

  // 加场景：走 CreateScene 真落盘（后端会把纯 blob 章先物化出首场景），
  // 然后重拉权威 id 序、对齐文本框与 id（经 onUpdate 回喂父层）；
  // 分支章无场景 API 语义，降级为本地加框。
  const addScene = async () => {
    if (addingScene) return
    if (tab.chapterNum < 1 || tab.node.branch) {
      onUpdate('scenes', [...tab.scenes, ''])
      return
    }
    setAddingScene(true)
    try {
      const n = tab.scenes.length + 1
      await CreateScene(tab.chapterNum, `scene-${n}`, `场景 ${n}`)
      let idList: string[] = []
      try {
        const value = await GetChapterScenes(tab.chapterNum)
        idList = Array.isArray(value) ? value.map((s: unknown) => sceneIdOf(s) || '') : []
      } catch { idList = [] }
      const boxes = Math.max(tab.scenes.length, idList.length)
      onUpdate('scenes', Array.from({ length: boxes }, (_v, i) => tab.scenes[i] ?? ''))
      onUpdate('sceneIds', idList)
      onUpdate('sceneBacked', true)
    } catch {
      message.error('创建场景失败，已本地添加（不会落盘）')
      onUpdate('scenes', [...tab.scenes, ''])
    } finally {
      setAddingScene(false)
    }
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

  const setScenePlot = (i: number, val: string) => {
    setScenePlots((prev) => {
      const next = [...prev]
      while (next.length <= i) next.push('')
      next[i] = val
      return next
    })
  }

  // 逐场景 AI 生成：用索引对齐的 sceneId 调 GenerateScene，写回 content 并展示 aiTaste/deSlop。
  const handleAiGenerate = async (i: number) => {
    const sceneId = sceneIds[i]
    if (!sceneId) { message.warning('该场景无绑定 ID，无法生成'); return }
    if (sceneGen[i]?.loading) return
    const plot = scenePlots[i] ?? ''
    const minWords = tab.targetWords || 800
    setSceneGen((prev) => ({ ...prev, [i]: { loading: true } }))
    try {
      const value = await GenerateScene(tab.chapterNum, sceneId, plot, minWords)
      const content = sceneContentOf(value)
      if (content) {
        const s = [...tab.scenes]
        s[i] = content
        onUpdate('scenes', s)
        onUpdate('saved', false)
      }
      const meta = sceneMetaOf(value)
      setSceneGen((prev) => ({ ...prev, [i]: { loading: false, aiTaste: meta.aiTaste, beforeScore: meta.beforeScore, afterScore: meta.afterScore, changes: meta.changes } }))
      message.success(`场景 ${i + 1} 已生成`)
    } catch (err: unknown) {
      setSceneGen((prev) => ({ ...prev, [i]: { loading: false } }))
      message.error(err instanceof Error ? err.message : '场景生成失败')
    }
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
          background: 'color-mix(in srgb, var(--color-warning) 10%, transparent)',
          border: '1px solid color-mix(in srgb, var(--color-warning) 30%, transparent)',
          borderRadius: 'var(--radius-md)',
          display: 'flex', alignItems: 'center', gap: 8,
          fontSize: 12, color: 'var(--color-warning)',
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
            {tab.scenes.map((scene: string, i: number) => {
              const g = sceneGen[i]
              const aiHint = g?.loading
                ? '正在生成…'
                : g?.beforeScore != null && g?.afterScore != null
                  ? `AI 味 ${g.aiTaste ?? '−'} → 去味后 ${g.afterScore}（改 ${g.changes ?? 0} 处）`
                  : g?.aiTaste != null
                    ? `AI 味检测 ${g.aiTaste} 分`
                    : ''
              return (
                <div key={i} style={{ marginBottom: 16 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 }}>
                    <Tag style={{ fontSize: 10 }}>场景 {i + 1}</Tag>
                    <Space size={2}>
                      <Button type="text" size="small" icon={<EyeOutlined />} style={{ color: C('color-text-secondary'), fontSize: 10, padding: '0 4px' }}
                        disabled={!sceneIds[i]}
                        onClick={() => setBibleScene(i)} aria-label={`场景 ${i + 1} 视角`} title="视角：该场景通过谁的眼睛在看——已知事实 / 不知情约束 / 出场角色 / 伏笔" />
                      <Button type="text" size="small" icon={<InfoCircleOutlined />} style={{ color: C('color-text-secondary'), fontSize: 10, padding: '0 4px' }}
                        disabled={tab.sceneBacked !== true}
                        onClick={() => void openMeta(i)} aria-label={`场景 ${i + 1} 信息`} title="场景信息：标题 / 概要 / POV / 地点 / 时间 / 情感 / 标签 / 状态" />
                      <Button type="text" size="small" icon={<ArrowUpOutlined />} style={{ color: C('color-text-secondary'), fontSize: 10, padding: '0 4px' }}
                        disabled={i === 0 || tab.sceneBacked !== true || moving}
                        onClick={() => void moveScene(i, -1)} aria-label={`场景 ${i + 1} 上移`} title="上移场景" />
                      <Button type="text" size="small" icon={<ArrowDownOutlined />} style={{ color: C('color-text-secondary'), fontSize: 10, padding: '0 4px' }}
                        disabled={i === tab.scenes.length - 1 || tab.sceneBacked !== true || moving}
                        onClick={() => void moveScene(i, 1)} aria-label={`场景 ${i + 1} 下移`} title="下移场景" />
                      <Button type="text" size="small" icon={<PlusOutlined />} aria-label="添加场景" style={{ color: C('color-text-secondary'), fontSize: 10, padding: '0 4px' }} loading={addingScene} onClick={() => void addScene()} />
                      <Button type="text" size="small" danger icon={<DeleteOutlined />} aria-label={`删除场景 ${i + 1}`} style={{ fontSize: 10, padding: '0 4px' }} onClick={() => removeScene(i)} disabled={tab.scenes.length <= 1} />
                    </Space>
                  </div>
                  {/* 逐场景 AI 生成：剧情要点（可选） + 生成按钮 + aiTaste/deSlop 简讯 */}
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap', marginBottom: 4 }}>
                    <Input
                      size="small"
                      placeholder="剧情要点（可选）"
                      value={scenePlots[i] ?? ''}
                      onChange={(e) => setScenePlot(i, e.target.value)}
                      style={{ width: 200 }}
                    />
                    <Button
                      size="small"
                      icon={<ThunderboltOutlined />}
                      loading={!!g?.loading}
                      disabled={!sceneIds[i]}
                      onClick={() => void handleAiGenerate(i)}
                    >
                      AI 生成
                    </Button>
                    {aiHint && (
                      <span style={{ fontSize: 10, color: (g?.beforeScore ?? g?.aiTaste ?? 0) >= 60 ? 'var(--color-warning)' : 'var(--color-text-secondary)' }}>
                        {aiHint}
                      </span>
                    )}
                  </div>
                  <div style={{ position: 'relative' }}>
                    <Input.TextArea
                      value={scene}
                      onChange={(e) => updateScene(i, e.target.value)}
                      onContextMenu={onSceneContextMenu}
                      className="writing-textarea"
                      autoSize={{ minRows: 4, maxRows: 20 }}
                      ref={(el: React.ComponentRef<typeof Input.TextArea> | null) => {
                        const ta = el?.resizableTextArea?.textArea
                        if (ta) sceneTextareaRefs.current.set(i, ta)
                      }}
                      onKeyDown={(e: React.KeyboardEvent<HTMLTextAreaElement>) => {
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
              )
            })}
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
              已选 {countTextChars(ctxMenu.text)} 字
            </Typography.Text>
          </div>
          <div onClick={() => triggerAI('describe')}
            style={{ padding: '8px 12px', cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 8, fontSize: 13, color: C('color-text') }}
            onMouseEnter={(e) => (e.currentTarget as HTMLElement).style.background = C('color-bg-layout')}
            onMouseLeave={(e) => (e.currentTarget as HTMLElement).style.background = 'transparent'}>
            <EditOutlined style={{ color: 'var(--color-success)' }} /> 丰富描写
          </div>
          <div onClick={() => triggerAI('expand')}
            style={{ padding: '8px 12px', cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 8, fontSize: 13, color: C('color-text') }}
            onMouseEnter={(e) => (e.currentTarget as HTMLElement).style.background = C('color-bg-layout')}
            onMouseLeave={(e) => (e.currentTarget as HTMLElement).style.background = 'transparent'}>
            <ColumnWidthOutlined style={{ color: 'var(--color-primary)' }} /> 扩展场景
          </div>
          <div onClick={() => triggerAI('rewrite')}
            style={{ padding: '8px 12px', cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 8, fontSize: 13, color: C('color-text') }}
            onMouseEnter={(e) => (e.currentTarget as HTMLElement).style.background = C('color-bg-layout')}
            onMouseLeave={(e) => (e.currentTarget as HTMLElement).style.background = 'transparent'}>
            <RedoOutlined style={{ color: 'var(--color-warning)' }} /> 重写此段
          </div>
        </div>
      )}

      {/* v4.199 场景元数据弹窗：POV/地点/时间/情感（元数据不进正文投影） */}
      <Modal
        open={!!metaTarget}
        title={`场景 ${metaTarget ? metaTarget.index + 1 : ''} 信息`}
        onCancel={() => setMetaTarget(null)}
        onOk={() => void saveMeta()}
        okText="保存"
        cancelText="取消"
        confirmLoading={metaSaving}
        destroyOnHidden
        transitionName=""
        maskTransitionName=""
        width={520}
      >
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
          <label style={{ display: 'flex', flexDirection: 'column', gap: 4, fontSize: 12 }}>
            标题
            <Input size="small" value={metaDraft.title} disabled={metaLoading}
              onChange={(e) => setMetaDraft((d) => ({ ...d, title: e.target.value }))} placeholder="保持原题可留空" />
          </label>
          <label style={{ display: 'flex', flexDirection: 'column', gap: 4, fontSize: 12 }}>
            POV 角色（影响 AI 生成视角）
            <Select size="small" showSearch allowClear placeholder="选择本书角色" loading={metaLoading}
              value={metaDraft.povCharId || undefined}
              options={povOptions}
              onChange={(v) => setMetaDraft((d) => ({ ...d, povCharId: v ?? '' }))} />
          </label>
          <label style={{ display: 'flex', flexDirection: 'column', gap: 4, fontSize: 12 }}>
            地点
            <Input size="small" value={metaDraft.location} disabled={metaLoading}
              onChange={(e) => setMetaDraft((d) => ({ ...d, location: e.target.value }))} placeholder="如：码头仓库" />
          </label>
          <label style={{ display: 'flex', flexDirection: 'column', gap: 4, fontSize: 12 }}>
            时间
            <Select size="small" allowClear placeholder="选择时间段" disabled={metaLoading}
              value={metaDraft.timeOfDay || undefined}
              options={['黎明', '早晨', '下午', '黄昏', '夜晚', '深夜'].map((t) => ({ value: t, label: t }))}
              onChange={(v) => setMetaDraft((d) => ({ ...d, timeOfDay: v ?? '' }))} />
          </label>
          <label style={{ display: 'flex', flexDirection: 'column', gap: 4, fontSize: 12 }}>
            情感基调
            <Input size="small" value={metaDraft.emotion} disabled={metaLoading}
              onChange={(e) => setMetaDraft((d) => ({ ...d, emotion: e.target.value }))} placeholder="如：紧张 / 温情" />
          </label>
          <label style={{ display: 'flex', flexDirection: 'column', gap: 4, fontSize: 12 }}>
            状态
            <Select size="small" disabled={metaLoading}
              value={metaDraft.status || 'draft'}
              options={[{ value: 'draft', label: '草稿' }, { value: 'revising', label: '修改中' }, { value: 'done', label: '完成' }]}
              onChange={(v) => setMetaDraft((d) => ({ ...d, status: v }))} />
          </label>
          <label style={{ display: 'flex', flexDirection: 'column', gap: 4, fontSize: 12, gridColumn: '1 / -1' }}>
            概要
            <Input size="small" value={metaDraft.summary} disabled={metaLoading}
              onChange={(e) => setMetaDraft((d) => ({ ...d, summary: e.target.value }))} placeholder="一句话概要" />
          </label>
          <label style={{ display: 'flex', flexDirection: 'column', gap: 4, fontSize: 12, gridColumn: '1 / -1' }}>
            标签（逗号分隔，如：climax, action）
            <Input size="small" value={metaDraft.tags} disabled={metaLoading}
              onChange={(e) => setMetaDraft((d) => ({ ...d, tags: e.target.value }))} placeholder="climax, action" />
          </label>
        </div>
      </Modal>
    {bibleScene != null && (
        <SceneBibleDrawer
          open
          chapterNum={tab.chapterNum}
          sceneID={sceneIds[bibleScene] ?? ''}
          sceneLabel={`场景 ${bibleScene + 1}`}
          onClose={() => setBibleScene(null)}
        />
      )}
    </>
  )
}

// v4.365 性能轮：memo——父页其它 state（focus/ghost/多开 tab）变化不再重渲染编辑区
export default React.memo(ChapterEditorInner)
