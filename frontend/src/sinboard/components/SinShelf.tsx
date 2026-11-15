// SinShelf.tsx — 原罪左栏书架（刀2）：书=工作空间，书下挂引擎会话。
// 数据源：ListProjects（书架）+ SinEngineState（当前书）+ SinEngineListSessions
// （当前书的会话）。切书=切工作空间（SinEngineOpenBook→新会话）。
import React from "react"
import { Button } from "antd"
import { BookOutlined, MessageOutlined, PlusOutlined } from "@ant-design/icons"

export interface SinShelfBook {
  path: string
  title: string
}

export interface SinShelfSession {
  path: string
  title: string
  preview: string
  current: boolean
  modTime: number
}

interface SinShelfProps {
  books: SinShelfBook[]
  activeBookPath: string
  activeBookTitle: string
  sessions: SinShelfSession[]
  onOpenBook: (path: string) => void
  onNewSession: () => void
  onResumeSession: (path: string) => void
}

function sessionTitle(s: SinShelfSession): string {
  if (s.title) return s.title
  return s.preview ? s.preview.slice(0, 24) : "会话"
}

const SinShelf: React.FC<SinShelfProps> = ({
  books, activeBookPath, activeBookTitle, sessions, onOpenBook, onNewSession, onResumeSession,
}) => {
  return (
    <aside className="sin-shelf" data-testid="sin-shelf">
      <div className="sin-shelf-head">
        <span>书架</span>
        <Button
          size="small"
          type="text"
          icon={<PlusOutlined />}
          onClick={onNewSession}
          disabled={!activeBookPath}
          title="在当前书下新建会话"
        >
          新会话
        </Button>
      </div>

      {/* 书列表（书=工作空间；当前书展开会话列表） */}
      <div className="sin-shelf-list">
        {books.map((b) => {
          const active = b.path === activeBookPath
          return (
            <div key={b.path}>
              <div
                className={`sin-shelf-item${active ? " is-active" : ""}`}
                onClick={() => { if (!active) onOpenBook(b.path) }}
                role="button"
                tabIndex={0}
                onKeyDown={(e) => {
                  if (e.key === "Enter" || e.key === " ") {
                    e.preventDefault()
                    if (!active) onOpenBook(b.path)
                  }
                }}
              >
                <BookOutlined className="sin-shelf-icon" />
                <div className="sin-shelf-text">
                  <div className="sin-shelf-title" title={b.title}>{b.title}</div>
                  {active && <div className="sin-shelf-preview">当前工作空间 · {activeBookTitle || b.title}</div>}
                </div>
              </div>
              {active && (
                <div className="sin-shelf-sessions" data-testid="sin-shelf-sessions">
                  {sessions.map((s) => (
                    <div
                      key={s.path}
                      className={`sin-shelf-item sin-shelf-session${s.current ? " is-active" : ""}`}
                      onClick={() => { if (!s.current) onResumeSession(s.path) }}
                      role="button"
                      tabIndex={0}
                      onKeyDown={(e) => {
                        if (e.key === "Enter" || e.key === " ") {
                          e.preventDefault()
                          if (!s.current) onResumeSession(s.path)
                        }
                      }}
                    >
                      <MessageOutlined className="sin-shelf-icon" />
                      <div className="sin-shelf-text">
                        <div className="sin-shelf-title" title={sessionTitle(s)}>{sessionTitle(s)}</div>
                        <div className="sin-shelf-preview" title={s.preview}>{s.preview || "（还没有内容）"}</div>
                      </div>
                    </div>
                  ))}
                  {sessions.length === 0 && (
                    <div className="sin-shelf-empty">本书还没有会话</div>
                  )}
                </div>
              )}
            </div>
          )
        })}
        {books.length === 0 && (
          <div className="sin-shelf-empty">书架是空的——先在小说板块建一本书</div>
        )}
      </div>
    </aside>
  )
}

export default SinShelf
