// NovelSettingPage.tsx — 小说「设定」面板
// 纯 Markdown 文本编辑 + 直接渲染：编辑 / 分屏 / 渲染三种模式，
// 不做结构化维度拆分与任何反解析；导入导出、设定 Agent 对话直接操作整篇文本。
// v4.3e/f：新增「维度化」模式（6 维度卡片分卡片编辑）与伏笔登记表 / 一致性检查面板。
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  Alert, Button, Input, Modal, Segmented, Space, Spin, Tag, message,
} from 'antd'
import {
  ColumnWidthOutlined, EditOutlined, ExportOutlined, EyeOutlined,
  FileTextOutlined, ImportOutlined, SaveOutlined, AppstoreOutlined,
  DownOutlined, FlagOutlined, UpOutlined,
} from '@ant-design/icons'
import ChatPanel from '../components/ChatPanel'
import type { Message } from '../components/ChatPanel'
import { MarkdownContent, mdStyles } from '../components/MarkdownContent'
import WorldviewSectionsEditor from '../components/novel/WorldviewSectionsEditor'
import V3Empty from '../components/V3Empty'
import ForeshadowPanel from '../components/novel/ForeshadowPanel'
import ConsistencyPanel from '../components/novel/ConsistencyPanel'
import { confirmDiscard } from '../components/novel/unsavedGuard'
import { registerNovelDirtyProvider, takeDiscardConfirmed } from '../components/novel/novelSwitchGuard'
import { useAppStore } from '../stores/appStore'
import { countTextChars, extractSettingText } from '../utils/text'
import { app } from '../gaea/lib/bridge'
import { inShellEnv, pickFileAsFile } from '../gaea/lib/pickFile'
import { saveExportBlob } from '../gaea/lib/saveFile'
import { useDebouncedValue } from '../hooks/useDebouncedValue'

type EditorMode = 'edit' | 'split' | 'preview' | 'sections'

/** 下行「伏笔 + 一致性」折叠态持久化键（本域 gaea.novel.* 命名风格） */
const PANELS_COLLAPSED_KEY = 'gaea.novel.settingPanelsCollapsed'

/** 简化批：默认收起（设定页主体是编辑器，300px 双面板不该常驻占屏）；
 * 显式展开过（'0'）的用户保持展开。畸形值一律按收起。 */
function loadPanelsCollapsed(): boolean {
  try { return localStorage.getItem(PANELS_COLLAPSED_KEY) !== '0' } catch { return true }
}

interface NovelSettingPageProps {
  /** 本页是否为当前可见子页（NovelPage 五 pane 常驻挂载，按 activeTab 下发）。
   *  默认 true：既有测试/独立渲染一律按可见处理；仅 `active === false` 时窗口级
   *  Ctrl+S 不响应、也不 preventDefault（否则在创作页/阅读页按 Ctrl+S 会顺带存设定）。 */
  active?: boolean
}

const NovelSettingPage: React.FC<NovelSettingPageProps> = ({ active = true }) => {
  const projectPath = useAppStore((s) => s.projectPath)
  const projectOpen = useAppStore((s) => s.projectOpen)

  const [content, setContent] = useState('')
  const [savedSnapshot, setSavedSnapshot] = useState('')
  const [loading, setLoading] = useState(true)
  // v4.361：读取失败可见化——此前失败被吞成空编辑器，用户随手输入再 Ctrl+S
  // 会把真实设定文件覆盖成残文。失败期间禁用保存，横幅提供重试。
  const [loadFailed, setLoadFailed] = useState(false)
  const [saving, setSaving] = useState(false)
  const [mode, setMode] = useState<EditorMode>('split')
  // 简化批：下行双面板默认收起（显式展开态持久化），见 loadPanelsCollapsed
  const [panelsCollapsed, setPanelsCollapsed] = useState(loadPanelsCollapsed)
  const [messages, setMessages] = useState<Message[]>([])
  const [lastSavedAt, setLastSavedAt] = useState('')
  const fileInputRef = useRef<HTMLInputElement>(null)
  const loadToken = useRef(0)
  // v4.425 跨页切书保护：脏探针与切书兜底提示都跑在 effect / 闸门回调里，必须读
  // **最新**的 content/snapshot（走 ref），否则闭包里的旧值会让「脏」判定失真。
  const contentRef = useRef('')
  contentRef.current = content
  const savedSnapshotRef = useRef('')
  savedSnapshotRef.current = savedSnapshot
  const loadFailedRef = useRef(false)
  loadFailedRef.current = loadFailed
  const savingRef = useRef(false)
  savingRef.current = saving
  // 「当前快照属于哪本书」：切书后的异步窗口里 content/savedSnapshot 都还停在上一本，
  // 单看两者相等会漏判脏；挂上快照所属路径后，快照不属于当前书即一律不脏。
  const snapshotPathRef = useRef(projectPath)

  const loadContent = useCallback(async () => {
    const token = ++loadToken.current
    if (!projectPath) {
      setContent('')
      setSavedSnapshot('')
      snapshotPathRef.current = ''
      setLoadFailed(false)
      setLoading(false)
      return
    }
    // 切书闸门已确认「放弃修改并切换」时静默换书；否则上一本的未保存设定会
    // 被 setContent 直接覆盖——连提示都没有（对齐 CreatePage 口径：不静默、也不对
    // 已确认的切书重复惊吓）。
    const discardConfirmed = takeDiscardConfirmed('settings-worldview')
    if (!discardConfirmed && contentRef.current !== savedSnapshotRef.current) {
      message.warning('已切换小说：上一本未保存的设定修改未保留')
    }
    // 脏态随书本上下文失效（本次拉取结束前不许再判脏）：否则「上一本脏 + 新书内容
    // 恰与本地缓冲相同」会让脏标志悬挂在已切换的新书上。
    setLoading(true)
    try {
      const text = await app.GetWorldview()
      if (token !== loadToken.current) return
      setContent(text || '')
      setSavedSnapshot(text || '')
      snapshotPathRef.current = projectPath
      setLoadFailed(false)
    } catch {
      if (token !== loadToken.current) return
      setContent('')
      setSavedSnapshot('')
      snapshotPathRef.current = projectPath
      setLoadFailed(true)
    } finally {
      if (token === loadToken.current) setLoading(false)
    }
  }, [projectPath])

  useEffect(() => { loadContent() }, [loadContent])
  useEffect(() => { setMessages([]) }, [projectPath])

  const needsProject = !projectOpen && !projectPath
  // 脏判定要连「快照属于哪本书」一起看：切书后的异步窗口里 content/savedSnapshot 还是
  // 上一本的值（此时不得判脏，否则会把上一本的脏挂到新书上）。
  const snapshotBelongsToCurrent = snapshotPathRef.current === projectPath
  const dirty = useMemo(
    () => !needsProject && snapshotBelongsToCurrent && content !== savedSnapshot,
    [content, savedSnapshot, needsProject, snapshotBelongsToCurrent],
  )
  // v4.365 性能轮：渲染/统计防抖——受控编辑器每键 setState 全文，分屏预览的
  // Markdown 全文重解析与字数全文扫描按 300ms 防抖跟随（最终值一致）。
  const debouncedContent = useDebouncedValue(content, 300)
  const wordCount = useMemo(() => countTextChars(debouncedContent.trim()), [debouncedContent])

  const handleSave = useCallback(async (): Promise<boolean> => {
    if (loadFailed) {
      message.warning('设定读取失败，已暂停保存以防覆盖，请先重试')
      return false
    }
    setSaving(true)
    try {
      await app.SaveWorldview(content)
      setSavedSnapshot(content)
      snapshotPathRef.current = projectPath
      setLastSavedAt(new Date().toLocaleTimeString())
      message.success('设定已保存')
      return true
    } catch (err: unknown) {
      message.error('保存失败: ' + (err instanceof Error ? err.message : String(err)))
      return false
    } finally {
      setSaving(false)
    }
  }, [content, loadFailed, projectPath])

  // 跨页切书未保存保护（`novelSwitchGuard` 的登记端）：把「设定页 markdown 脏不脏、
  // 能否先保存」暴露给书架页的切书闸门。本页有显式保存钮（handleSave）故一并提供
  // save 能力（三选）；读失败期间不能保存 → canSave=false 自动降级两选。
  // dirty/save 经 ref 读最新值，故只登记一次。
  const handleSaveRef = useRef(handleSave)
  handleSaveRef.current = handleSave
  useEffect(() => registerNovelDirtyProvider({
    id: 'settings-worldview',
    label: () => '设定页未保存的设定',
    // 快照不属于当前书时一律不脏（切书异步窗口内 content/snapshot 仍是上一本的值）
    dirty: () => !loadFailedRef.current
      && snapshotPathRef.current === useAppStore.getState().projectPath
      && contentRef.current !== savedSnapshotRef.current,
    canSave: () => !loadFailedRef.current && !savingRef.current,
    save: () => handleSaveRef.current(),
  }), [])

  // Ctrl/Cmd+S 保存——仅在「本页为当前子页」时挂监听：五子页常驻挂载，
  // 无条件挂 window 会让在创作页/阅读页按下的 Ctrl+S 顺带保存设定（规格线2/线3）。
  useEffect(() => {
    if (!active) return
    const handler = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 's') {
        e.preventDefault()
        handleSave()
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [active, handleSave])

  const togglePanelsCollapsed = useCallback(() => {
    setPanelsCollapsed((prev) => {
      const next = !prev
      try { localStorage.setItem(PANELS_COLLAPSED_KEY, next ? '1' : '0') } catch { /* 存储不可用则仅本会话生效 */ }
      return next
    })
  }, [])

  /** 导入内容进编辑器（FileReader 读文本回填）——壳内 PickFiles 与浏览器 input 共用管线 */
  const readFileIntoContent = useCallback((file: File) => {
    const reader = new FileReader()
    reader.onload = () => {
      setContent((reader.result as string) || '')
      message.success(`已导入「${file.name}」`)
    }
    reader.readAsText(file)
  }, [])

  /** 导入入口统一走这里：dirty 时先经共享原语确认（✕/Esc＝取消，绝不静默覆盖）。 */
  const importFile = useCallback((file: File) => {
    if (dirty) {
      confirmDiscard({
        title: '导入会覆盖未保存的设定',
        message: `当前设定有未保存的修改，导入「${file.name}」会用文件内容覆盖编辑器，这些修改将丢失。`,
        discardLabel: '覆盖导入',
        onDiscard: () => readFileIntoContent(file),
      })
      return
    }
    readFileIntoContent(file)
  }, [dirty, readFileIntoContent])

  // 审计刀B b：壳内 <input type=file> 不弹框（v4.162）→ pickFileAsFile 系统
  // 对话框（扩展名同 input accept 口径）还原 File 喂原 FileReader 管线；
  // 浏览器回退隐藏 input.click()（原生弹框，口径不变）。
  const handleImport = async () => {
    if (!inShellEnv()) { fileInputRef.current?.click(); return }
    try {
      const file = await pickFileAsFile(['md', 'txt', 'json'])
      if (file) importFile(file)
    } catch (err: unknown) {
      message.error('导入失败: ' + (err instanceof Error ? err.message : String(err)))
    }
  }

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    importFile(file)
    e.target.value = ''
  }

  // 审计刀B b：壳内 <a download> 不落盘（v4.162）→ saveExportBlob 系统另存为；
  // 浏览器回退原 <a download>（saveExportBlob 内置）。
  const handleExport = async () => {
    const blob = new Blob([content], { type: 'text/markdown;charset=utf-8' })
    const saved = await saveExportBlob(blob, 'novel_setting.md')
    if (saved) message.success('已导出')
  }

  const handleChatSend = async (userMsg: string): Promise<string> => {
    try {
      // 回填守卫（观察池#2）：await 前记快照——等待期作者可能直接改设定正文
      //（ChatPanel 输入区虽禁，编辑器不禁），AI 修改稿不得静默覆盖手改。
      const contentBefore = contentRef.current
      const result = await app.ChatWorldview(userMsg, contentBefore)
      // ChatWorldview 返回结构化 Record（reply/worldview 均为未知字段）——
      // reply 按 string 收窄取用，worldview 非空 string 时回填编辑器
      const reply = typeof result?.reply === 'string' ? result.reply : ''
      // AI 返回更新后的设定文本，直接回填编辑器（不解析、不拆分）
      if (typeof result?.worldview === 'string' && result.worldview) {
        if (contentRef.current === contentBefore) {
          setContent(result.worldview)
        } else {
          // 等待期正文已变：不覆盖。全文附在回复里（markdown 围栏，可被
          // 「应用」按钮的 extractSettingText 提取），由作者决定是否采纳。
          return `${reply}\n\n你在等待期间修改过设定正文，AI 修改稿未自动应用。如需采纳，请点本条消息的「应用」：\n\n\`\`\`markdown\n${result.worldview}\n\`\`\``
        }
      }
      return reply
    } catch (err: unknown) {
      const msg = typeof err === 'string' ? err : (err instanceof Error ? (err.message || '对话失败') : '对话失败')
      throw new Error(msg)
    }
  }

  /** 手动将某条 AI 回复应用到设定编辑器（覆盖） */
  const handleApplyAiOutput = useCallback((msg: Message) => {
    const text = extractSettingText(msg.content)
    if (!text) {
      message.warning('这条消息没有可应用的内容')
      return
    }
    Modal.confirm({
      title: '将 AI 输出应用到设定',
      content: text !== msg.content.trim()
        ? '已提取回复中的 Markdown 代码块，将覆盖当前设定编辑器内容；未保存的修改会丢失。'
        : '将用这条 AI 回复的完整内容覆盖当前设定编辑器；未保存的修改会丢失。',
      okText: '覆盖设定',
      okButtonProps: { danger: true },
      cancelText: '取消',
      onOk: () => {
        setContent(text)
        message.success('已应用到设定编辑器，点击「保存」生效')
      },
    })
  }, [])

  const editorPane = (
    <Input.TextArea
      className="novel-editor"
      value={content}
      onChange={(e) => setContent(e.target.value)}
      placeholder="在此撰写或粘贴小说设定（世界观、剧情框架、人物关系、规则体系等）…"
      style={{ flex: 1, minHeight: 0, minWidth: 0, resize: 'none' }}
    />
  )

  const previewPane = (
    <div className="novel-setting-preview md-content">
      <MarkdownContent source={debouncedContent} />
    </div>
  )

  const sectionsPane = (
    <WorldviewSectionsEditor disabled={needsProject} projectPath={projectPath} />
  )

  const editorBody = needsProject ? (
    <V3Empty style={{ margin: 'auto' }}
      description="请先在「书架」打开或创建一部小说项目" />
  ) : loading ? (
    <div style={{ margin: 'auto' }}><Spin size="large" /></div>
  ) : mode === 'edit' ? (
    editorPane
  ) : mode === 'preview' ? (
    previewPane
  ) : mode === 'sections' ? (
    sectionsPane
  ) : (
    <>
      {editorPane}
      {previewPane}
    </>
  )

  return (
    <div style={{
      display: 'flex', flexDirection: 'column', width: '100%', height: '100%',
      padding: '16px', maxWidth: 1240, margin: '0 auto', gap: 14, minHeight: 0, minWidth: 0,
    }}>
      <style>{mdStyles}</style>

      {/* 上行：设定编辑器（Markdown + 维度化 + 渲染）+ 设定 Agent */}
      <div style={{ display: 'flex', flexDirection: 'row', flex: 1, minHeight: 0, gap: 14 }}>
        {/* 左栏：设定编辑器（Markdown + 渲染） */}
        <div className="novel-panel" style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden', minHeight: 0 }}>
          <div className="novel-panel-head" style={{ flexWrap: 'wrap', rowGap: 6 }}>
            <span className="novel-panel-title"><FileTextOutlined />设定编辑器</span>
            <div style={{ flex: 1 }} />
            <span className="novel-setting-meta">{wordCount.toLocaleString()} 字</span>
            <Space size={8}>
              <Button size="small" icon={<ImportOutlined />} onClick={handleImport}
                style={{ borderColor: 'var(--color-primary)', color: 'var(--color-primary)' }}>
                导入
              </Button>
              <Button size="small" icon={<ExportOutlined />} onClick={handleExport}
                style={{ borderColor: 'var(--color-warning)', color: 'var(--color-warning)' }}>
                导出
              </Button>
              <Button size="small" type="primary" icon={<SaveOutlined />} onClick={handleSave} loading={saving} disabled={loadFailed}>
                保存
              </Button>
            </Space>
          </div>

          {loadFailed && (
            <Alert
              type="error" showIcon style={{ margin: '8px 12px 0' }}
              data-testid="novel-setting-load-failed"
              message="设定读取失败"
              description="为防把真实设定文件覆盖成空稿，保存已暂停。"
              action={<Button size="small" danger onClick={() => loadContent()}>重试</Button>}
            />
          )}

          <input
            ref={fileInputRef}
            type="file"
            accept=".txt,.md,.json"
            style={{ display: 'none' }}
            onChange={handleFileChange}
          />

          <div className="novel-setting-toolbar">
            <Segmented
              size="small"
              value={mode}
              onChange={(val) => setMode(val as EditorMode)}
              options={[
                { label: '编辑', value: 'edit', icon: <EditOutlined /> },
                { label: '分屏', value: 'split', icon: <ColumnWidthOutlined /> },
                { label: '渲染', value: 'preview', icon: <EyeOutlined /> },
                { label: '维度化', value: 'sections', icon: <AppstoreOutlined /> },
              ]}
            />
            <div style={{ flex: 1 }} />
            <Tag style={{ marginInlineEnd: 0 }} color={dirty ? 'warning' : 'success'}>
              {dirty ? '有未保存修改' : lastSavedAt ? `已保存 ${lastSavedAt}` : '无修改'}
            </Tag>
          </div>

          <div
            className="novel-setting-body"
            style={mode === 'split' && !needsProject ? { display: 'flex', flexDirection: 'row', gap: 12 } : { display: 'flex' }}
          >
            {editorBody}
          </div>
        </div>

        {/* 右栏：设定 Agent 常驻对话 */}
        <div className="novel-panel" style={{ width: 380, flexShrink: 0, minHeight: 0, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
          <ChatPanel
            title="设定 Agent"
            messages={messages}
            onSend={handleChatSend}
            onMessagesChange={setMessages}
            onApply={handleApplyAiOutput}
            placeholder="描述你想要的设定修改，AI 帮你调整…"
            fillHeight
          />
        </div>
      </div>

      {/* 下行：伏笔登记表 + 一致性检查（可折叠；折叠态持久化，收起时仅留一行标题） */}
      {!needsProject && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8, flexShrink: 0, minHeight: 0 }}>
          <div className="novel-panel-head" style={{ flexShrink: 0 }}>
            <span className="novel-panel-title"><FlagOutlined />伏笔 + 一致性</span>
            <div style={{ flex: 1 }} />
            <span className="novel-setting-meta">
              {panelsCollapsed ? '已收起（内容未变）' : '伏笔登记表 · 一致性检查'}
            </span>
            <Button
              size="small" type="text"
              icon={panelsCollapsed ? <DownOutlined /> : <UpOutlined />}
              onClick={togglePanelsCollapsed}
              aria-expanded={!panelsCollapsed}
              aria-label={panelsCollapsed ? '展开伏笔与一致性面板' : '收起伏笔与一致性面板'}
              data-testid="novel-setting-panels-toggle"
            >
              {panelsCollapsed ? '展开' : '收起'}
            </Button>
          </div>
          {!panelsCollapsed && (
            <div
              data-testid="novel-setting-panels"
              style={{ display: 'flex', flexDirection: 'row', gap: 14, height: 300, flexShrink: 0, minHeight: 0 }}
            >
              <ForeshadowPanel disabled={needsProject} />
              <ConsistencyPanel disabled={needsProject} />
            </div>
          )}
        </div>
      )}
    </div>
  )
}

export default NovelSettingPage
