/**
 * EditChrome — 编辑模式顶部 chrome（章节 Tab / 保存状态 / 章节导航 / 朗读 /
 * 阅读模式 / 专注模式 / 配图 / 保存），自 ChapterPage 原样搬移：纯受控展示组件，
 * 全部状态与回调经 props 传入；classNames/data-testid 与拆分前逐字节一致。
 */
import React from 'react'
import { Button, Tabs, Tooltip } from 'antd'
import type { TabsProps } from 'antd'
import {
  SaveOutlined, LeftOutlined, RightOutlined,
  ReadOutlined, ShrinkOutlined, ExpandOutlined, PictureOutlined,
} from '@ant-design/icons'
import TTSPlayer from '../../components/TTSPlayer'

export interface EditChromeProps {
  tabItems: TabsProps['items']
  activeKey: string
  onTabChange: (key: string) => void
  onTabRemove: (key: string) => void
  saved: boolean
  onPrev: () => void
  onNext: () => void
  atFirst: boolean
  atLast: boolean
  getText: () => string
  onRead: () => void
  focusMode: boolean
  onToggleFocus: () => void
  canIllustrate: boolean
  onIllustrate: () => void
  onSave: () => void
  canSave: boolean
}

const EditChrome: React.FC<EditChromeProps> = ({
  tabItems,
  activeKey,
  onTabChange,
  onTabRemove,
  saved,
  onPrev,
  onNext,
  atFirst,
  atLast,
  getText,
  onRead,
  focusMode,
  onToggleFocus,
  canIllustrate,
  onIllustrate,
  onSave,
  canSave,
}) => (
  <>
    <Tabs
      className="novel-editor-tabs"
      activeKey={activeKey}
      onChange={onTabChange}
      onEdit={(key, action) => { if (action === 'remove' && typeof key === 'string') onTabRemove(key) }}
      items={tabItems}
      type="editable-card"
      size="small"
      hideAdd
      style={{ flex: 1, minWidth: 0, marginBottom: 0 }}
      tabBarStyle={{ marginBottom: 0 }}
    />
    <span className="novel-chrome-spacer" />
    <span className="novel-chrome-save-state">
      <i className={`novel-dot ${saved ? 'ok' : 'dirty'}`} aria-hidden />
      <span style={{ color: saved ? 'var(--color-success)' : 'var(--color-warning)', fontSize: 11 }}>
        {saved ? '已保存' : '未保存'}
      </span>
    </span>
    <div className="novel-chrome-nav">
      <Tooltip title="上一章"><Button size="small" icon={<LeftOutlined />} onClick={onPrev} type="text" disabled={atFirst} aria-label="上一章" /></Tooltip>
      <Tooltip title="下一章"><Button size="small" icon={<RightOutlined />} onClick={onNext} type="text" disabled={atLast} aria-label="下一章" /></Tooltip>
    </div>
    <TTSPlayer getText={getText} />
    <Tooltip title="阅读模式（沉浸排版）">
      <Button size="small" icon={<ReadOutlined />} onClick={onRead} className="is-readmode" aria-label="进入阅读模式" />
    </Tooltip>
    <Tooltip title={focusMode ? '退出专注模式' : '专注模式 F11'}>
      <Button size="small" icon={focusMode ? <ShrinkOutlined /> : <ExpandOutlined />} onClick={onToggleFocus} type="text" aria-label="专注模式" />
    </Tooltip>
    <Tooltip title="为当前章节生成配图">
      <Button
        size="small"
        icon={<PictureOutlined />}
        onClick={onIllustrate}
        disabled={!canIllustrate}
        aria-label="生成配图"
      >
        配图
      </Button>
    </Tooltip>
    <Tooltip title="Ctrl+S">
      <Button size="small" icon={<SaveOutlined />} onClick={onSave} disabled={!canSave}>保存</Button>
    </Tooltip>
  </>
)

export default EditChrome