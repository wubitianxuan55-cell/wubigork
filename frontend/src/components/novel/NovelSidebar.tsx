import React from 'react'
import { Button, Tooltip } from 'antd'
import {
  MenuFoldOutlined, MenuUnfoldOutlined,
  BookOutlined,
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
  projectPath: string
}

/**
 * NovelSidebar — 书房工坊「左 = 目录 zone」
 * 可交互大纲树；未开书时给去书架引导。仅在阅读页展开（外壳按 tab 显隐）。
 */
const NovelSidebar: React.FC<NovelSidebarProps> = ({
  outlines, activeKey, collapsed, onToggleCollapse,
  onOpenChapter, onGoBookshelf, projectPath,
}) => {
  if (collapsed) {
    return (
      <aside className="v3-panel novel-zone novel-side-zone is-collapsed" aria-label="目录（已折叠）">
        <div className="novel-zone-head">
          <Button type="text" size="small" icon={<MenuUnfoldOutlined />}
            onClick={onToggleCollapse} aria-label="展开目录" title="展开目录"
            style={{ color: C('color-text-secondary') }} />
        </div>
      </aside>
    )
  }

  return (
    <aside className="v3-panel novel-zone novel-side-zone" aria-label="目录">
      <div className="novel-zone-head">
        <span className="novel-zone-title">目录</span>
        <div className="novel-zone-spacer" />
        <Tooltip title="折叠目录">
          <Button type="text" size="small" icon={<MenuFoldOutlined />}
            onClick={onToggleCollapse} aria-label="折叠目录"
            style={{ color: C('color-text-secondary') }} />
        </Tooltip>
      </div>

      <div className="novel-zone-body">
        {projectPath ? null : (
          <div className="novel-book-card">
            <span className="novel-book-kicker">尚未打开</span>
            <span className="novel-book-title">去书架选一本书</span>
            <Button size="small" type="primary" ghost icon={<BookOutlined aria-hidden />} onClick={onGoBookshelf}>
              去书架
            </Button>
          </div>
        )}

        <div className="novel-outline-shell">
          <OutlinePanel
            outlines={outlines}
            activeKey={activeKey}
            onSelectNode={onOpenChapter}
            collapsed={false}
            onToggleCollapse={() => {}}
          />
        </div>
      </div>
    </aside>
  )
}

export default NovelSidebar
