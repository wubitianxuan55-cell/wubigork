// SinMemoryCard.tsx — 书级记忆卡（刀4）：facts play 空间的管理面
// （AI 侧经 remember/memory_get 工具读写同一存储；这里是人工查看/增删/固化）。
import React, { useCallback, useEffect, useState } from "react"
import { Button, Empty, Input, Popconfirm, Tag } from "antd"
import { DeleteOutlined, PushpinFilled, PushpinOutlined, SearchOutlined } from "@ant-design/icons"
import { app } from "../../gaea/lib/bridge"
import type { SinMemoryEntry } from "../../gaea/lib/bridge/sin"

interface SinMemoryCardProps {
  /** 引擎是否就绪（未绑书=空态引导）。 */
  enabled: boolean
}

const SinMemoryCard: React.FC<SinMemoryCardProps> = ({ enabled }) => {
  const [entries, setEntries] = useState<SinMemoryEntry[]>([])
  const [query, setQuery] = useState("")
  const [loading, setLoading] = useState(false)
  const [adding, setAdding] = useState(false)
  const [draftTitle, setDraftTitle] = useState("")
  const [draftBody, setDraftBody] = useState("")
  const [error, setError] = useState("")

  const reload = useCallback(async (q = query) => {
    setLoading(true)
    setError("")
    try {
      setEntries(await app.SinMemoryList(q))
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setLoading(false)
    }
  }, [query])

  useEffect(() => {
    if (enabled) void reload()
  }, [enabled, reload])

  const handleSave = useCallback(async () => {
    if (!draftTitle.trim() || !draftBody.trim()) return
    try {
      await app.SinMemorySave("", draftTitle.trim(), draftBody.trim())
      setDraftTitle("")
      setDraftBody("")
      setAdding(false)
      await reload()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    }
  }, [draftTitle, draftBody, reload])

  const handleDelete = useCallback(async (name: string) => {
    try {
      await app.SinMemoryDelete(name)
      await reload()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    }
  }, [reload])

  const handlePin = useCallback(async (entry: SinMemoryEntry) => {
    try {
      await app.SinMemoryPinToggle(entry.name, !entry.pinned)
      await reload()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    }
  }, [reload])

  return (
    <div className="sin-memory-card" data-testid="sin-memory-card">
      <div className="sin-memory-toolbar">
        <Input
          size="small"
          allowClear
          prefix={<SearchOutlined />}
          placeholder="搜记忆（键/标题/正文）"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onPressEnter={() => void reload()}
          disabled={!enabled}
          aria-label="记忆搜索"
        />
        <Button
          size="small"
          type={adding ? "default" : "primary"}
          onClick={() => setAdding((v) => !v)}
          disabled={!enabled}
        >
          {adding ? "收起" : "记一笔"}
        </Button>
      </div>
      {adding && enabled && (
        <div className="sin-memory-form">
          <Input
            size="small"
            placeholder="标题（如：林晚的身份）"
            value={draftTitle}
            onChange={(e) => setDraftTitle(e.target.value)}
            maxLength={60}
          />
          <Input.TextArea
            size="small"
            rows={3}
            placeholder="内容（剧情事实/角色状态/伏笔——AI 也会读这里）"
            value={draftBody}
            onChange={(e) => setDraftBody(e.target.value)}
            maxLength={2000}
            showCount
          />
          <Button size="small" type="primary" onClick={() => void handleSave()} disabled={!draftTitle.trim() || !draftBody.trim()}>
            保存
          </Button>
        </div>
      )}
      {error && <div className="sin-memory-error" role="alert">{error}</div>}
      <div className="sin-memory-list">
        {!enabled && (
          <Empty description="先在书架选择一本书——记忆随书存放（play 空间）" image={Empty.PRESENTED_IMAGE_SIMPLE} />
        )}
        {enabled && !loading && entries.length === 0 && (
          <Empty description="本书还没有记忆——剧情事实/伏笔记在这里，跨会话都在" image={Empty.PRESENTED_IMAGE_SIMPLE} />
        )}
        {entries.map((m) => (
          <div key={m.name} className="sin-memory-item" data-testid="sin-memory-item">
            <div className="sin-memory-item-head">
              {m.pinned && <Tag color="gold" style={{ marginInlineEnd: 4 }}>固</Tag>}
              <span className="sin-memory-item-title" title={m.name}>{m.title || m.name}</span>
              <span className="sin-memory-item-time">{m.updatedAt}</span>
              <Button
                size="small"
                type="text"
                icon={m.pinned ? <PushpinFilled /> : <PushpinOutlined />}
                aria-label={m.pinned ? `取消固化 ${m.title}` : `固化 ${m.title}`}
                onClick={() => void handlePin(m)}
              />
              <Popconfirm
                title="删除这条记忆？"
                okText="删除"
                cancelText="取消"
                onConfirm={() => void handleDelete(m.name)}
              >
                <Button size="small" type="text" danger icon={<DeleteOutlined />} aria-label={`删除记忆 ${m.title}`} />
              </Popconfirm>
            </div>
            <div className="sin-memory-item-body">{m.body}</div>
          </div>
        ))}
        {loading && <div className="sin-memory-error">读取中…</div>}
      </div>
    </div>
  )
}

export default SinMemoryCard
