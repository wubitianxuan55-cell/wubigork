// AppBoard.tsx — 原罪板块壳（2026-11-15 升格刀 fork 办公 App 架构）。
//
// 布局三栏：左=书架工作空间（SinShelf）/ 中=会话区（ChatTabs：故事/轨迹/上下文
// + SinTranscript + Composer）/ 右=创作面板（刀6 接入，暂留 stub）。
// 数据层：sinboard/lib/store（独立 zustand 实例 + gaea-event-sin 通道 + SinEngine*
// 绑定）；审批/提问/工具卡/Markdown 渲染复用办公叶组件（props 纯）。
// 隔离纪律：不 import 办公 controller store 单例；办公板块挂载与否不影响本页。

import React, { useCallback, useEffect, useState } from "react"
import { Button, Modal } from "antd"
import { ExportOutlined, PlusOutlined } from "@ant-design/icons"
import { Composer } from "../gaea/components/Composer"
import { ContextView } from "../gaea/components/ContextView"
import { TrajectoryView } from "../gaea/components/TrajectoryView"
import { ApprovalModal } from "../gaea/components/ApprovalModal"
import { AskCard } from "../gaea/components/AskCard"
import { LocaleProvider } from "../gaea/lib/i18n"
import { app } from "../gaea/lib/bridge"
import { subscribe } from "../events"
import { openNovelProject } from "../components/novel/openNovelProject"
import { useSinController } from "./lib/store/controller"
import SinShelf, { type SinShelfBook, type SinShelfSession } from "./components/SinShelf"
import { Transcript as SinTranscript } from "./components/SinTranscript"
import { ChatTabs } from "./components/SinChatTabs"
import SinReadingPane from "./components/SinReadingPane"
import "../gaea/styles.css"
import "../gaea/tailwind.css"
import "../gaea/redesign.css"
import "../pages/sin/sin.css"
import "./sinboard.css"

type SinTab = "chat" | "trajectory" | "context" | "reading" | "setting"

/** 设定 tab：小说板块 SettingHub 整件 rehost（世界观/人物内切，薄壳只读复用）。 */
const SettingHub = React.lazy(() => import("../pages/SettingHub")) as React.ComponentType<{ active?: boolean }>

/** ProjectCard（Go 投影）→ 书架条目；字段按 Record 弱类型读取，缺省诚实兜底。 */
function toShelfBook(raw: Record<string, unknown>): SinShelfBook {
  return {
    path: typeof raw.path === "string" ? raw.path : "",
    title: typeof raw.title === "string" && raw.title ? raw.title : "未命名",
  }
}

/** SessionMeta（Go 投影）→ 书架会话条目。 */
function toShelfSession(raw: Record<string, unknown>, currentPath: string): SinShelfSession {
  return {
    path: typeof raw.path === "string" ? raw.path : "",
    title: typeof raw.title === "string" ? raw.title : "",
    preview: typeof raw.preview === "string" ? raw.preview : "",
    current: typeof raw.path === "string" && raw.path !== "" && raw.path === currentPath,
    modTime: typeof raw.modTime === "number" ? raw.modTime : 0,
  }
}

const AppBoard: React.FC = () => {
  const ctl = useSinController()
  const { state } = ctl
  const [books, setBooks] = useState<SinShelfBook[]>([])
  const [sessions, setSessions] = useState<SinShelfSession[]>([])
  const [engineReady, setEngineReady] = useState(false)
  const [bookPath, setBookPath] = useState("")
  const [bookTitle, setBookTitle] = useState("")
  const [opening, setOpening] = useState(false)
  const [sinTab, setSinTab] = useState<SinTab>("chat")
  // 落章联动信号：回合结束（gaea-event-sin turn_done）递增 → 阅读 tab 重拉章节清单
  const [chapterTick, setChapterTick] = useState(0)

  // 书架清单（书架目录扫描；失败给空清单不报错阻断）
  const reloadBooks = useCallback(async () => {
    try {
      const list = await app.ListProjects()
      setBooks((list || []).map((r) => toShelfBook(r as unknown as Record<string, unknown>)))
    } catch { setBooks([]) }
  }, [])

  // 当前书下的会话清单（引擎未就绪=空）
  const reloadSessions = useCallback(async () => {
    try {
      const list = await app.SinEngineListSessions()
      setSessions((list || []).map((r) => toShelfSession(r as unknown as Record<string, unknown>, "")))
    } catch { setSessions([]) }
  }, [])

  // 书标题快查表（引擎状态只带路径，标题从书架清单对齐）
  const booksRef = React.useRef(new Map<string, string>())
  useEffect(() => {
    booksRef.current = new Map(books.map((b) => [b.path, b.title]))
  }, [books])

  // 引擎状态回读（书/就绪）
  const refreshEngineState = useCallback(async () => {
    try {
      const st = await app.SinEngineState()
      setEngineReady(st.ready)
      setBookPath(st.bookPath)
      setBookTitle(st.bookPath ? (booksRef.current.get(st.bookPath) ?? "") : "")
    } catch { setEngineReady(false) }
  }, [])

  // 启动：书架 + 引擎状态 + 会话；落章联动订阅（turn_done → chapterTick）
  useEffect(() => {
    void reloadBooks()
    void refreshEngineState()
    const off = subscribe("gaea-event-sin", (raw) => {
      const e = (raw || {}) as { kind?: string }
      if (e.kind === "turn_done") setChapterTick((t) => t + 1)
    })
    return off
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])
  useEffect(() => {
    if (engineReady) void reloadSessions()
  }, [engineReady, reloadSessions])

  // 切书 = 切工作空间：引擎 OpenBook + 小说项目落位（openNovelProject——
  // 设定/阅读页签的 NovelB 绑定吃全局当前项目）→ 新会话 → 装载 → 刷新
  const handleOpenBook = useCallback(async (path: string) => {
    if (opening) return
    setOpening(true)
    try {
      await app.SinEngineOpenBook(path)
      const title = booksRef.current.get(path) ?? ""
      await openNovelProject(path, title)
      setEngineReady(true)
      setBookPath(path)
      setBookTitle(title)
      await ctl.newSession()
      await reloadSessions()
      const st = await app.SinEngineState()
      setBookPath(st.bookPath || path)
    } catch (err) {
      Modal.error({ title: "打开书失败", content: err instanceof Error ? err.message : String(err) })
    } finally {
      setOpening(false)
    }
  }, [ctl, opening, reloadSessions])

  const handleNewSession = useCallback(async () => {
    await ctl.newSession()
    await reloadSessions()
  }, [ctl, reloadSessions])

  const handleResumeSession = useCallback(async (path: string) => {
    await ctl.resumeSession(path)
    await reloadSessions()
  }, [ctl, reloadSessions])

  const modelText = state.meta ? "" : ""
  void modelText

  const fetchSinTrajectory = useCallback(() => app.SinEngineTrajectory(), [])
  const fetchSinContext = useCallback(() => app.SinEngineContextView(), [])
  const fetchSinNodeDetail = useCallback((seq: number) => app.SinEngineContextNodeDetail(seq), [])

  return (
    <LocaleProvider>
      <div className="sin-page" data-testid="sinboard-root">
        {/* ── 顶栏：板块名 + 当前书 + 动作 ── */}
        <header className="sin-head">
          <div className="sin-head-title">
            <span className="sin-mark" aria-hidden="true">原罪</span>
            <span className="sin-head-sub">
              {bookTitle || (engineReady ? "已就绪" : "选择一本书开始创作")}
              <span className="sin-dot" aria-hidden="true" />
              小说创作 · 书即工作空间
            </span>
          </div>
          <div className="sin-head-actions">
            <Button
              size="small"
              type="text"
              icon={<PlusOutlined />}
              onClick={() => void handleNewSession()}
              disabled={!engineReady || state.running}
              title="在当前书下新建会话"
            >
              新会话
            </Button>
            <Button
              size="small"
              type="text"
              icon={<ExportOutlined />}
              disabled
              title="导出（刀3 接入）"
            >
              导出
            </Button>
          </div>
        </header>

        <div className="sin-body">
          {/* ── 左：书架工作空间 ── */}
          <SinShelf
            books={books}
            activeBookPath={bookPath}
            activeBookTitle={bookTitle}
            sessions={sessions}
            onOpenBook={(p) => void handleOpenBook(p)}
            onNewSession={() => void handleNewSession()}
            onResumeSession={(p) => void handleResumeSession(p)}
          />

          {/* ── 中：主区页签（故事/轨迹/上下文/阅读/设定）── */}
          <main className="sin-main">
            <ChatTabs
              active={sinTab}
              onChange={(id) => setSinTab(id as SinTab)}
              tabs={["chat", "trajectory", "context", "reading", "setting"]}
              labelOverrides={{ chat: "故事" }}
              className="sin-tabs"
            />
            <div className={`sin-chat-pane${sinTab === "chat" ? "" : " is-hidden"}`}>
              <div className="sin-stream-host">
                <SinTranscript running={state.running} onFeedback={false} onPrompt={() => {}} />
              </div>
              <div className="sin-composer">
                <Composer
                  running={state.running}
                  onSend={(display, submit) => ctl.send(display, submit)}
                  onCancel={ctl.cancel}
                  onPickFolder={async () => ""}
                  disabled={!engineReady}
                  onSteer={ctl.steer}
                />
              </div>
            </div>
            {sinTab === "trajectory" && (
              <div className="sin-insight-host">
                <TrajectoryView
                  running={state.running}
                  fetchTrajectory={fetchSinTrajectory}
                />
              </div>
            )}
            {sinTab === "context" && (
              <div className="sin-insight-host">
                <ContextView
                  running={state.running}
                  fetchTimeline={fetchSinContext}
                  fetchNodeDetail={fetchSinNodeDetail}
                  sessionName={bookTitle || "当前书"}
                  spaceOverride="娱乐空间"
                  showAgentNetwork={false}
                />
              </div>
            )}
            {sinTab === "reading" && (
              <div className="sin-reading-host">
                <SinReadingPane active={sinTab === "reading"} chapterTick={chapterTick} hasBook={engineReady} />
              </div>
            )}
            {sinTab === "setting" && (
              <div className="sin-setting-host">
                <React.Suspense fallback={<div className="novel-tab-skeleton" aria-hidden />}>
                  <SettingHub active={sinTab === "setting"} />
                </React.Suspense>
              </div>
            )}
          </main>

          {/* ── 右：创作面板（刀6 接入 pane 注册表；占位保留布局位）── */}
          <div className="sin-side-host is-hidden">
            <div />
          </div>
        </div>

        {/* ── 审批 / 提问卡（引擎交互闸；输入岛上沿非模态）── */}
        {state.approval && (
          <ApprovalModal
            approval={state.approval}
            onAnswer={(decision) => ctl.approve(state.approval!.id, decision)}
          />
        )}
        {state.ask && (
          <AskCard
            ask={state.ask}
            onAnswer={(id, answers) => ctl.answerQuestion(id, answers)}
            onDismiss={() => ctl.answerQuestion(state.ask!.id, [])}
          />
        )}
      </div>
    </LocaleProvider>
  )
}

export default AppBoard
