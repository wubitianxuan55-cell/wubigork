// OriginalSinPage.tsx — 闲庭 · 原罪板块（图文混杂的成人向故事创作）。
//
// 用户口径（原话）：闲庭新增独立板块、复用办公的组件、主要用于与 AI 对话、
// 可以生成图文混杂的 H 故事小说，板块取名「原罪」。落地：
//   · 版面与组件复用办公板块——Composer（办公输入器：粘贴/附件/截图/表格转换
//     全继承）、Markdown（办公消息同一条渲染链）、ToolbarButton/EmptyState、
//     办公样式层（styles/tailwind/redesign 三件套，与 GaeaPage 同口径）；
//   · 文本走功能级路由 sin（模型中心「原罪」可单绑模型），插图走绘梦后端；
//   · 会话与消息复用统一聊天存储（mode=sin），与聊天板块互不串台。
// 后端：internal/app/sin_handler.go / sin_prompt.go（docs/ADULT_MODE.md 成人内容口径）。

import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Button, Dropdown, Input, message, Modal, Popconfirm, Tooltip } from 'antd'
import {
  BookOutlined, CloseOutlined, DeleteOutlined, EditOutlined, ExportOutlined, FileTextOutlined,
  MessageOutlined, PlusOutlined, ReloadOutlined, SettingOutlined,
} from '@ant-design/icons'
import { PanelRightClose, PanelRightOpen } from '../gaea/icons'
import { Composer } from '../gaea/components/Composer'
import { ToolbarButton } from '../gaea/components/ToolbarButton'
import { LocaleProvider } from '../gaea/lib/i18n'
import { saveExportBlob } from '../gaea/lib/saveFile'
import { app } from '../gaea/lib/bridge'
import { emitFrontendEvent, FRONTEND_EVENTS } from '../events'
import { useFeatureModel } from '../hooks/useFeatureModel'
import { isNearBottom } from '../utils/scroll'
import { StoryStream } from './sin/StoryStream'
import { useSinStory } from './sin/useSinStory'
import { useSinCast } from './sin/useSinCast'
import { useSinNotes } from './sin/useSinNotes'
import { readSinPanelOpen, writeSinPanelOpen } from './sin/sinPanelState'
import { SinSidePanel } from './sin/SinSidePanel'
import { SinCastPicker } from './sin/SinCastPicker'
import { suggestStoryTitle, type SinGalleryItem } from './sin/storyText'
import { enqueueIllustration } from './sin/illustrationQueue'
import '../gaea/styles.css'
import '../gaea/tailwind.css'
import '../gaea/redesign.css'
import './sin/sin.css'

/** 进入原罪的方式：模型中心「原罪」功能绑定（与 FeatureModelBar 同数据源）。 */
function navigateToBoard(target: string): void {
  emitFrontendEvent(FRONTEND_EVENTS.NAVIGATE, target)
}

const OriginalSinPage: React.FC = () => {
  const story = useSinStory()
  const cast = useSinCast(story.activeId)
  const notes = useSinNotes(story.activeId)
  const model = useFeatureModel('sin')
  const [castPickerOpen, setCastPickerOpen] = useState(false)
  const [panelOpen, setPanelOpen] = useState<boolean>(() => readSinPanelOpen())
  const [renameTarget, setRenameTarget] = useState('')
  const [renameDraft, setRenameDraft] = useState('')
  const [exporting, setExporting] = useState(false)
  const listRef = useRef<HTMLDivElement>(null)
  const stickRef = useRef(true)

  useEffect(() => {
    writeSinPanelOpen(panelOpen)
  }, [panelOpen])

  // 每个回合结束后重读便签/大纲（AI 可能在回合里用 sin_notes/sin_outline 写了底稿）
  const { reload: reloadNotes } = notes
  const prevSendingRef = useRef(false)
  useEffect(() => {
    if (prevSendingRef.current && !story.sending) reloadNotes()
    prevSendingRef.current = story.sending
  }, [story.sending, reloadNotes])

  // ── 智能滚动：贴底时跟随流式输出；用户上翻阅读时不打断 ──
  const onScroll = useCallback(() => {
    const el = listRef.current
    if (!el) return
    stickRef.current = isNearBottom(el.scrollHeight - el.scrollTop - el.clientHeight)
  }, [])
  const lastLen = story.messages.reduce((n, m) => n + m.content.length, 0)
  useEffect(() => {
    const el = listRef.current
    if (el && stickRef.current) el.scrollTop = el.scrollHeight
  }, [lastLen, story.messages.length, story.activeId])

  // 滚动容器由 StoryStream 渲染（forwardRef），切故事时重置贴底
  useEffect(() => {
    stickRef.current = true
  }, [story.activeId])

  const onSend = useCallback((display: string) => {
    void story.send(display)
  }, [story])

  // 画廊「重新生成」：与流内插图同一串行队列（一次一张）→ SinIllustrate 按
  // messageId+cue 覆盖回写 → 消息重载刷新画廊与流内。失败/未回写如实提示，
  // 原图不受影响（回写失败时 extra 保留旧路径）。
  const onRegenerate = useCallback(async (item: SinGalleryItem) => {
    if (!story.activeId || story.sending) return
    if (!item.messageId || !item.prompt.trim()) {
      story.showNotice('该插图缺少定位或画面描述，无法重新生成')
      return
    }
    const { promise } = enqueueIllustration(() =>
      app.SinIllustrate(story.activeId, item.messageId, item.cue, item.prompt, ''),
    )
    try {
      const res = (await promise) as { persisted?: unknown }
      if (res?.persisted === false) {
        story.showNotice('新图已生成但回写失败，画廊保留原图')
      }
      await story.reloadMessages()
    } catch (err) {
      story.showNotice(err instanceof Error ? err.message : '重新生成失败')
    }
  }, [story])

  const onExport = useCallback(async () => {
    if (!story.activeId) return
    setExporting(true)
    try {
      const md = await app.SinExportMarkdown(story.activeId)
      const name = `${(story.activeStory?.title ?? '原罪故事').replace(/[\\/:*?"<>|]/g, '_')}.md`
      await saveExportBlob(new Blob([md], { type: 'text/markdown;charset=utf-8' }), name)
    } catch (err) {
      Modal.error({ title: '导出失败', content: err instanceof Error ? err.message : String(err) })
    } finally {
      setExporting(false)
    }
  }, [story])

  // EPUB 电子书：后端直接落原罪导出目录（插图内嵌进书），返回路径告知即可
  // ——二进制不走前端「另存为」（壳内 a[download] 是死的），与书源线 EPUB 导出同口径。
  const onExportEpub = useCallback(async () => {
    if (!story.activeId) return
    setExporting(true)
    try {
      const path = await app.SinExportEpub(story.activeId)
      message.success({ content: `已导出 EPUB：${path}`, duration: 6 })
    } catch (err) {
      Modal.error({ title: '导出失败', content: err instanceof Error ? err.message : String(err) })
    } finally {
      setExporting(false)
    }
  }, [story])

  const modelText = model.engine && model.model
    ? `${model.engine} / ${model.model}${model.enabled ? '' : '（已停用·回退全局）'}`
    : '未绑定 · 跟随全局激活模型'

  const stories = useMemo(() => story.stories, [story.stories])

  return (
    <LocaleProvider>
      <div className="sin-page">
        {/* ── 顶栏：板块名 + 当前故事 + 模型状态 + 动作 ── */}
        <header className="sin-head">
          <div className="sin-head-title">
            <span className="sin-mark" aria-hidden="true">原罪</span>
            <span className="sin-head-sub">
              {story.activeStory?.title ?? '新故事'}
              <span className="sin-dot" aria-hidden="true" />
              图文故事 · 与 AI 对话式创作
            </span>
          </div>
          <div className="sin-head-actions">
            <ToolbarButton
              title={panelOpen ? '收起创作面板' : '展开创作面板'}
              onClick={() => setPanelOpen((v) => !v)}
            >
              {panelOpen ? <PanelRightClose size={13} /> : <PanelRightOpen size={13} />}
            </ToolbarButton>
            <Tooltip title={`故事模型：${modelText}（在模型中心「原罪」功能绑定里更换）`}>
              <Button size="small" type="text" icon={<SettingOutlined />} onClick={() => navigateToBoard('modelcenter')}>
                {model.engine && model.model ? `${model.model}` : '未绑定模型'}
              </Button>
            </Tooltip>
            <Dropdown
              trigger={['click']}
              disabled={exporting || !story.activeId || story.messages.length === 0}
              menu={{
                items: [
                  { key: 'md', icon: <FileTextOutlined />, label: '图文 Markdown（另存为）' },
                  { key: 'epub', icon: <BookOutlined />, label: 'EPUB 电子书（插图内嵌）' },
                ],
                onClick: ({ key }) => {
                  if (key === 'md') void onExport()
                  else void onExportEpub()
                },
              }}
            >
              <ToolbarButton title="导出故事" onClick={() => {}}>
                <ExportOutlined />
              </ToolbarButton>
            </Dropdown>
            <ToolbarButton
              title="清空本故事消息"
              // ToolbarButton 无 forwardRef，Popconfirm 包裹会丢事件——用命令式
              // Modal.confirm（同页删除故事走 Popconfirm 是 Button 先例，此处不适用）
              onClick={() => {
                if (!story.activeId) return;
                Modal.confirm({
                  title: '清空本故事全部消息？',
                  content: '清空后不可恢复',
                  okText: '清空',
                  okButtonProps: { danger: true },
                  cancelText: '取消',
                  onOk: () => story.clearStory(story.activeId),
                });
              }}
              disabled={!story.activeId || story.messages.length === 0}
            >
              <ReloadOutlined />
            </ToolbarButton>
          </div>
        </header>

        <div className="sin-body">
          {/* ── 左：故事架 ── */}
          <aside className="sin-shelf">
            <div className="sin-shelf-head">
              <span>故事</span>
              <Button size="small" type="text" icon={<PlusOutlined />} onClick={() => void story.createStory()}>
                新建
              </Button>
            </div>
            <div className="sin-shelf-list">
              {stories.map((s) => (
                <div
                  key={s.id}
                  className={`sin-shelf-item${s.id === story.activeId ? ' is-active' : ''}`}
                  onClick={() => void story.selectStory(s.id)}
                  role="button"
                  tabIndex={0}
                  onKeyDown={(e) => { if (e.key === 'Enter') void story.selectStory(s.id) }}
                >
                  <MessageOutlined className="sin-shelf-icon" />
                  <div className="sin-shelf-text">
                    <div className="sin-shelf-title" title={s.title}>{s.title}</div>
                    <div className="sin-shelf-preview" title={s.preview}>{s.preview || '（还没有内容）'}</div>
                  </div>
                  <Popconfirm
                    title="删除这个故事？"
                    description="故事与其消息会一并删除，不可恢复。"
                    okText="删除"
                    cancelText="取消"
                    onConfirm={() => void story.deleteStory(s.id)}
                  >
                    <Button
                      size="small"
                      type="text"
                      danger
                      icon={<DeleteOutlined />}
                      onClick={(e) => e.stopPropagation()}
                    />
                  </Popconfirm>
                  <Button
                    size="small"
                    type="text"
                    icon={<EditOutlined />}
                    aria-label="重命名"
                    title="重命名"
                    onClick={(e) => {
                      e.stopPropagation()
                      setRenameTarget(s.id)
                      setRenameDraft(s.title)
                    }}
                  />
                </div>
              ))}
              {stories.length === 0 && <div className="sin-shelf-empty">还没有故事</div>}
            </div>
          </aside>

          {/* ── 中：故事流 + 输入 ── */}
          <main className="sin-main">
            <div className="sin-stream-host">
              <StoryStream
                ref={listRef}
                storyId={story.activeId}
                messages={story.messages}
                onIllustrationGenerated={story.setIllustration}
                onIllustrationError={story.showNotice}
                onScroll={onScroll}
              />
            </div>
            {story.notice && (
              <div className="sin-notice" role="status">
                <span className="sin-notice-text">{story.notice}</span>
                <Button size="small" type="text" icon={<CloseOutlined />} onClick={story.clearNotice} aria-label="关闭提示" />
              </div>
            )}
            <div className="sin-composer">
              <Composer
                running={story.sending}
                onSend={onSend}
                onCancel={story.cancel}
                onPickFolder={async () => ''}
                disabled={story.initializing}
              />
            </div>
          </main>

          {/* ── 右：创作面板（办公同款标签页：角色/大纲/设定/插图 + 玩法气泡） ── */}
          {panelOpen && (
            <SinSidePanel
              cast={cast.cast}
              castSaving={cast.saving}
              onOpenPicker={() => setCastPickerOpen(true)}
              onRemoveCast={(id) => void cast.saveCast(cast.castIds.filter((x) => x !== id))}
              notesDoc={notes.doc}
              notesError={notes.error}
              notesLoading={notes.loading}
              messages={story.messages}
              sending={story.sending}
              onRegenerate={onRegenerate}
              onSaveNotes={notes.save}
            />
          )}
        </div>

        <SinCastPicker
          open={castPickerOpen}
          library={cast.library}
          loading={cast.libraryLoading}
          error={cast.libraryError}
          selectedIds={cast.castIds}
          saving={cast.saving}
          onSave={(ids) => { void cast.saveCast(ids); setCastPickerOpen(false) }}
          onClose={() => setCastPickerOpen(false)}
        />

        {/* 重命名弹窗（内联，避免在侧栏塞输入框破坏密度） */}
        <Modal
          open={renameTarget !== ''}
          title="重命名故事"
          okText="保存"
          cancelText="取消"
          onCancel={() => setRenameTarget('')}
          onOk={() => {
            void story.renameStory(renameTarget, renameDraft.trim() || suggestStoryTitle(renameDraft))
            setRenameTarget('')
          }}
        >
          <Input
            value={renameDraft}
            onChange={(e) => setRenameDraft(e.target.value)}
            placeholder="给这个故事起个名字"
            maxLength={40}
          />
        </Modal>
      </div>
    </LocaleProvider>
  )
}

export default OriginalSinPage
