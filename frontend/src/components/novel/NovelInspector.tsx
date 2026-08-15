import React, { useEffect, useState } from 'react'
import { Button, Tooltip, Tag } from 'antd'
import {
  ControlOutlined, MenuFoldOutlined, MenuUnfoldOutlined,
  FileTextOutlined, BookOutlined, ThunderboltOutlined, UserOutlined,
  ExportOutlined, ReadOutlined, InfoCircleOutlined,
} from '@ant-design/icons'
import { useAppStore } from '../../stores/appStore'
import { C } from '../../utils/theme'
import type { NovelTab } from '../../pages/NovelPage'

/** novel:chapter-active 事件载荷（阅读页上报当前章节属性） */
interface ChapterActivePayload {
  id?: string
  title?: string
  words?: number
  saved?: boolean
  status?: string
}

interface NovelInspectorProps {
  activeTab: NovelTab
  collapsed: boolean
  onToggleCollapse: () => void
  onNavigate: (tab: NovelTab) => void
  stats: { totalWords: number; chapterCount: number } | null
}

/**
 * NovelInspector — 世界构建工作台「右 = 属性 inspector zone」（可折叠）
 * 按 tab 呈现上下文属性：书架=项目统计；阅读=当前章节属性；其余=世界统计 + 快捷入口。
 */
const NovelInspector: React.FC<NovelInspectorProps> = ({
  activeTab, collapsed, onToggleCollapse, onNavigate, stats,
}) => {
  const projectTitle = useAppStore((s) => s.projectTitle)
  const projectPath = useAppStore((s) => s.projectPath)
  const [chapter, setChapter] = useState<ChapterActivePayload | null>(null)

  // 订阅阅读页上报的当前章节属性
  useEffect(() => {
    const handler = (e: Event) => {
      const data = (e as CustomEvent<ChapterActivePayload>).detail
      if (data) setChapter(data)
    }
    window.addEventListener('novel:chapter-active', handler)
    return () => window.removeEventListener('novel:chapter-active', handler)
  }, [])

  const quickActions: Array<{ key: NovelTab; label: string; icon: React.ReactNode }> = [
    { key: 'create', label: '创作', icon: <ThunderboltOutlined /> },
    { key: 'novelsetting', label: '设定', icon: <FileTextOutlined /> },
    { key: 'character', label: '角色', icon: <UserOutlined /> },
  ]

  if (collapsed) {
    return (
      <aside className="v3-panel novel-zone novel-inspector-zone is-collapsed" aria-label="属性检查器（已折叠）">
        <div className="novel-zone-head">
          <Button type="text" size="small" icon={<ControlOutlined />}
            onClick={onToggleCollapse} aria-label="展开属性检查器" title="展开属性检查器"
            style={{ color: C('color-text-secondary') }} />
        </div>
      </aside>
    )
  }

  const renderChapterSection = () => (
    <section className="novel-inspector-section">
      <div className="novel-inspector-section-title"><ReadOutlined />当前章节</div>
      {chapter?.id ? (
        <>
          <div className="novel-inspector-item">
            <span className="novel-inspector-item-label">标题</span>
            <span className="novel-inspector-item-value">{chapter.title || '未命名章节'}</span>
          </div>
          <div className="novel-inspector-item">
            <span className="novel-inspector-item-label">字数</span>
            <span className="novel-inspector-item-value">{(chapter.words ?? 0).toLocaleString()} 字</span>
          </div>
          <div className="novel-inspector-item">
            <span className="novel-inspector-item-label">保存状态</span>
            <span className="novel-inspector-item-value">
              <Tag color={chapter.saved ? 'success' : 'warning'} style={{ marginInlineEnd: 0, fontSize: 11 }}>
                {chapter.saved ? '已保存' : '未保存'}
              </Tag>
            </span>
          </div>
        </>
      ) : (
        <div className="novel-inspector-hint">从左侧大纲选择章节后，这里会显示该章节的属性。</div>
      )}
    </section>
  )

  return (
    <aside className="v3-panel novel-zone novel-inspector-zone" aria-label="属性检查器">
      <div className="novel-zone-head">
        <span className="novel-zone-title"><ControlOutlined />属性</span>
        <div className="novel-zone-spacer" />
        <Tooltip title="折叠检查器">
          <Button type="text" size="small" icon={<MenuFoldOutlined />}
            onClick={onToggleCollapse} aria-label="折叠属性检查器"
            style={{ color: C('color-text-secondary'), fontSize: 11 }} />
        </Tooltip>
      </div>

      <div className="novel-zone-body">
        {/* 项目统计（全 tab 通用） */}
        <section className="novel-inspector-section">
          <div className="novel-inspector-section-title"><BookOutlined />世界统计</div>
          <div className="novel-inspector-item">
            <span className="novel-inspector-item-label">当前小说</span>
            <span className="novel-inspector-item-value">{projectTitle || '（未打开）'}</span>
          </div>
          <div className="novel-inspector-item">
            <span className="novel-inspector-item-label">章节数 / 总字数</span>
            <span className="novel-inspector-item-value">
              {stats?.chapterCount ?? 0} 章 · {(stats?.totalWords ?? 0).toLocaleString()} 字
            </span>
          </div>
        </section>

        {/* 阅读 tab：当前章节属性 */}
        {activeTab === 'chapter' && renderChapterSection()}

        {/* 设定 tab：设定预览提示 */}
        {activeTab === 'novelsetting' && (
          <section className="novel-inspector-section">
            <div className="novel-inspector-section-title"><InfoCircleOutlined />提示</div>
            <div className="novel-inspector-hint">
              世界观设定为纯 Markdown 文本，支持编辑 / 分屏 / 渲染三种模式；创作生成时会自动注入最新设定。
            </div>
          </section>
        )}

        {/* 导出 tab：说明 */}
        {activeTab === 'export' && (
          <section className="novel-inspector-section">
            <div className="novel-inspector-section-title"><ExportOutlined />导出</div>
            <div className="novel-inspector-hint">
              一键导出全部格式到小说目录下的 export/ 文件夹（TXT + Markdown + EPUB）。
            </div>
          </section>
        )}

        {/* 快捷入口 */}
        {projectPath && activeTab !== 'chapter' && (
          <section className="novel-inspector-section">
            <div className="novel-inspector-section-title"><ThunderboltOutlined />快捷入口</div>
            <div className="novel-action-row">
              {quickActions.map((a) => (
                <Button key={a.key} size="small" icon={a.icon} onClick={() => onNavigate(a.key)} style={{ fontSize: 11 }}>
                  {a.label}
                </Button>
              ))}
            </div>
          </section>
        )}
      </div>
    </aside>
  )
}

export default NovelInspector
