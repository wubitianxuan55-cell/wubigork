import React from 'react'
import { Button, Popconfirm, Space } from 'antd'
import {
  BookOutlined, RightOutlined,
  ReloadOutlined, PlusOutlined, DeleteOutlined, ShareAltOutlined,
} from '@ant-design/icons'
import { C } from '../../../utils/theme'
import { chapterLabel, type TreeNode } from './outlineTree'
import type { OutlineNode } from '../../../types'

interface ChapterTreePanelProps {
  flatNodes: TreeNode[]
  activeId: string
  nextChapterNum: number
  onSelect: (node: OutlineNode) => void
  onRegenerate: (node: OutlineNode) => void
  onDelete: (node: OutlineNode) => void
  /** 在节点后生成下一章/分支（含覆盖确认，由页面层实现并打开向导） */
  onAddNext: (node: OutlineNode) => void
  /** 底部「生成第N章」入口（主线程末尾追加） */
  onGenerateNext: () => void
}

/** 章节树面板（T6-7.5 从 CreatePage 拆分） */
const ChapterTreePanel: React.FC<ChapterTreePanelProps> = ({
  flatNodes, activeId, nextChapterNum,
  onSelect, onRegenerate, onDelete, onAddNext, onGenerateNext,
}) => (
  <aside className="novel-panel novel-workspace-col novel-tree-col">
    <div className="novel-panel-head">
      <span className="novel-panel-title"><ShareAltOutlined />章节</span>
      <div style={{ flex: 1 }} />
      <span className="novel-setting-meta">{flatNodes.length} 章</span>
    </div>
    <div className="novel-tree" style={{ flex: 1, overflow: 'auto' }}>
      {flatNodes.length === 0 ? (
        <div className="novel-tree-empty">
          <BookOutlined />
          <span>还没有章节，从「生成第 1 章」开始</span>
        </div>
      ) : flatNodes.map(tn => {
        const n = tn.node
        const isActive = activeId === n.id
        const chapNum = n.order_index || 1
        const isBranch = !!n.parent_id
        const statusClass = n.status === 'writing' ? 'is-writing' : n.status === 'done' ? 'is-done' : 'is-todo'
        return (
          <div
            key={n.id}
            className={`novel-tree-item${isActive ? ' is-active' : ''}`}
            style={{ paddingLeft: 8 + tn.depth * 14, borderLeft: isBranch ? '2px solid var(--md-sys-color-outline-variant)' : undefined }}
          >
            <span className={`novel-tree-status ${statusClass}`} title={n.status || '未写'} />
            <div onClick={() => onSelect(n)}
              title={n.summary ? `摘要: ${n.summary.slice(0, 100)}${n.summary.length > 100 ? '...' : ''}` : `ID: ${n.id}`}
              style={{ flex: 1, cursor: 'pointer', fontSize: 12, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', minWidth: 0 }}>
              {isBranch ? <ShareAltOutlined style={{ fontSize: 9, marginRight: 4, opacity: 0.5 }} /> : <RightOutlined style={{ fontSize: 10, marginRight: 4, opacity: 0.4 }} />}
              {n.title || chapterLabel(chapNum)}
            </div>
            <Space size={0}>
              <Button type="text" size="small" icon={<ReloadOutlined />}
                onClick={() => onRegenerate(n)}
                style={{ color: C('color-text-secondary'), fontSize: 11, padding: '0 3px', height: 22 }} title="重新生成" />
              <Popconfirm title="删除此章节？" onConfirm={() => onDelete(n)} okText="删除" cancelText="取消">
                <Button type="text" size="small" danger icon={<DeleteOutlined />}
                  style={{ fontSize: 11, padding: '0 3px', height: 22 }} />
              </Popconfirm>
              <Button type="text" size="small" icon={<PlusOutlined />}
                onClick={() => onAddNext(n)}
                style={{ color: 'var(--md-sys-color-primary)', fontSize: 11, padding: '0 3px', height: 22 }}
                title={tn.children.length > 0 ? '添加子分支' : '生成下一章'} />
            </Space>
          </div>
        )
      })}
    </div>
    <div className="novel-tree-footer">
      <Button block type="dashed" icon={<PlusOutlined />} onClick={onGenerateNext}>
        生成第{nextChapterNum}章
      </Button>
    </div>
  </aside>
)

export default ChapterTreePanel
