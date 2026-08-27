import React from 'react'
import { Button, Tooltip } from 'antd'
import {
  GlobalOutlined, MenuFoldOutlined,
  ReadOutlined, BookOutlined, BarChartOutlined,
} from '@ant-design/icons'
import OutlinePanel from './OutlinePanel'
import { C } from '../../utils/theme'
import type { OutlineNode } from '../../types'

interface NovelSidebarProps {
  /** 当前书目大纲（已排序） */
  outlines: OutlineNode[]
  /** 当前激活章节 id（阅读页通过 novel:chapter-active 事件上报） */
  activeKey: string
  /** 是否折叠（44px 微条） */
  collapsed: boolean
  onToggleCollapse: () => void
  /** 点击大纲节点 → 打开阅读 tab 并定位章节 */
  onOpenChapter: (node: OutlineNode) => void
  /** 跳回书架 tab */
  onGoBookshelf: () => void
  projectTitle: string
  projectPath: string
  stats: { totalWords: number; chapterCount: number } | null
}

/**
 * NovelSidebar — 世界构建工作台「左 = 侧栏 zone」
 * 当前书目卡（书架/大纲树上下文）+ 可交互大纲树导航。
 */
const NovelSidebar: React.FC<NovelSidebarProps> = ({
  outlines, activeKey, collapsed, onToggleCollapse,
  onOpenChapter, onGoBookshelf, projectTitle, projectPath, stats,
}) => {
  if (collapsed) {
    return (
      <aside className="v3-panel novel-zone novel-side-zone is-collapsed" aria-label="世界侧栏（已折叠）">
        <div className="novel-zone-head">
          <Button type="text" size="small" icon={<GlobalOutlined />}
            onClick={onToggleCollapse} aria-label="展开世界侧栏" title="展开世界侧栏"
            style={{ color: C('color-text-secondary') }} />
        </div>
      </aside>
    )
  }

  return (
    <aside className="v3-panel novel-zone novel-side-zone" aria-label="世界侧栏">
      <div className="novel-zone-head">
        <span className="novel-zone-title"><GlobalOutlined />世界大纲</span>
        <div className="novel-zone-spacer" />
        <Tooltip title="折叠侧栏">
          <Button type="text" size="small" icon={<MenuFoldOutlined />}
            onClick={onToggleCollapse} aria-label="折叠世界侧栏"
            style={{ color: C('color-text-secondary'), fontSize: 11 }} />
        </Tooltip>
      </div>

      <div className="novel-zone-body">
        {/* 当前书目卡 */}
        {projectPath ? (
          <div className="novel-book-card">
            <span className="novel-book-title"><ReadOutlined aria-hidden style={{ marginRight: 4 }} />{projectTitle || '未命名小说'}</span>
            <span className="novel-book-path" title={projectPath}>{projectPath}</span>
            <span className="novel-book-meta">
              <span>章节 <b>{stats?.chapterCount ?? 0}</b></span>
              <span>字数 <b>{(stats?.totalWords ?? 0).toLocaleString()}</b></span>
            </span>
          </div>
        ) : (
          <div className="novel-book-card">
            <span className="novel-book-title"><BookOutlined aria-hidden style={{ marginRight: 4 }} />未打开小说</span>
            <span className="novel-book-path">去书架打开或新建一部小说</span>
            <Button size="small" type="primary" ghost icon={<BookOutlined />} onClick={onGoBookshelf}>
              去书架
            </Button>
          </div>
        )}

        {/* 大纲树导航（内部折叠按钮在 shell 中隐藏，由 zone 折叠接管） */}
        <div className="novel-outline-shell">
          <OutlinePanel
            outlines={outlines}
            activeKey={activeKey}
            onSelectNode={onOpenChapter}
            collapsed={false}
            onToggleCollapse={() => {}}
          />
        </div>

        {outlines.length > 0 && (
          <div className="novel-action-row">
            <Button size="small" icon={<BarChartOutlined />} onClick={onGoBookshelf} style={{ fontSize: 11 }}>
              书架
            </Button>
          </div>
        )}
      </div>
    </aside>
  )
}

export default NovelSidebar
