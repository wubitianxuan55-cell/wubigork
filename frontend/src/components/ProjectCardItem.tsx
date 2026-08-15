import React from 'react'
import { Button, Popconfirm, Tooltip } from 'antd'
import {
  BookOutlined, ReadOutlined, ClockCircleOutlined,
  DeleteOutlined, RightOutlined,
} from '@ant-design/icons'
import type { ProjectCard } from '../stores/appStore'
import { formatRelativeTime } from '../utils/time'

interface ProjectCardItemProps {
  card: ProjectCard
  isActive: boolean
  isHero: boolean
  isMobile?: boolean
  readingChapter?: string
  /** 阅读进度（0-1，undefined = 无进度） */
  readingProgress?: number
  onOpen: (card: ProjectCard) => void
  onContinueReading?: (card: ProjectCard) => void
  onDelete: (card: ProjectCard) => void
}

/** 题材 → 封面 tone 类（零硬编码色值，令牌派生渐变） */
function coverToneFor(genre: string): string {
  const g = (genre || '').trim()
  if (!g || g === '未分类') return 'novel-cover-tone-primary'
  if (/玄幻|奇幻|仙侠|魔幻|魔法|神话|东方幻想/.test(g)) return 'novel-cover-tone-fantasy'
  if (/科幻|赛博|未来|末世|星际|机甲|末日/.test(g)) return 'novel-cover-tone-scifi'
  if (/悬疑|推理|惊悚|恐怖|侦探|犯罪/.test(g)) return 'novel-cover-tone-mystery'
  if (/言情|爱情|都市情缘|耽美|百合|婚恋/.test(g)) return 'novel-cover-tone-romance'
  if (/历史|穿越|古代|架空|军事|战争/.test(g)) return 'novel-cover-tone-history'
  if (/都市|现实|职场|校园|生活|青春/.test(g)) return 'novel-cover-tone-urban'
  return 'novel-cover-tone-primary'
}

/** ProjectCardItem — 书架卡：封面渐变条 + 题材标签 + 进度条 + 继续阅读主操作 */
const ProjectCardItem: React.FC<ProjectCardItemProps> = ({
  card, isActive, isHero, readingChapter, readingProgress, onOpen, onContinueReading, onDelete,
}) => {
  const pct = readingProgress != null && readingProgress > 0
    ? Math.min(100, Math.round(readingProgress * 100))
    : 0
  const tone = coverToneFor(card.genre)

  return (
    <div
      key={card.path}
      role="button"
      tabIndex={0}
      className={`novel-shelf-card${isActive ? ' is-active' : ''}${isHero ? ' novel-shelf-card-hero' : ''}`}
      onClick={() => onOpen(card)}
      onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onOpen(card) } }}
      aria-label={`打开小说「${card.title}」`}
    >
      {/* 封面渐变条 */}
      <div className={`novel-cover ${tone}`}>
        <BookOutlined aria-hidden className="novel-cover-icon" />
        <span className="novel-cover-title">{card.title}</span>
      </div>

      <div className="novel-shelf-body">
        {/* 题材 / 风格标签 */}
        <div className="novel-shelf-tags">
          {card.genre && card.genre !== '未分类' && (
            <span className="novel-tag-tone is-primary">{card.genre}</span>
          )}
          {card.style && card.style !== '默认' && (
            <span className="novel-tag-tone is-neutral">{card.style}</span>
          )}
        </div>

        {/* 统计 + 最近打开 */}
        <div className="novel-shelf-meta">
          <span>{card.chapter_count > 0
            ? <><b>{card.word_count.toLocaleString()}</b> 字 · <b>{card.chapter_count}</b> 章</>
            : '尚未开始写作'}</span>
          <span aria-hidden style={{ opacity: 0.6 }}>·</span>
          <span><ClockCircleOutlined aria-hidden /> {formatRelativeTime(card.last_opened_at)}</span>
        </div>

        {readingChapter && (
          <div className="novel-shelf-continue" title={readingChapter}>
            <ReadOutlined aria-hidden />
            <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{readingChapter}</span>
          </div>
        )}

        {/* 阅读进度条（主色，章节进度） */}
        {pct > 0 && (
          <div className="novel-progress-track" role="progressbar" aria-valuenow={pct} aria-valuemin={0} aria-valuemax={100} aria-label={`阅读进度 ${pct}%`}>
            <i className="novel-progress-fill" style={{ width: `${pct}%` }} />
          </div>
        )}

        {/* 操作区 */}
        <div className="novel-shelf-actions">
          {readingChapter ? (
            <Tooltip title="跳转到阅读页继续上次章节">
              <Button size="small" type="primary" icon={<ReadOutlined />} onClick={(e) => { e.stopPropagation(); onContinueReading ? onContinueReading(card) : onOpen(card) }}>
                继续阅读
              </Button>
            </Tooltip>
          ) : (
            <Button size="small" icon={<RightOutlined />} onClick={(e) => { e.stopPropagation(); onOpen(card) }}>
              打开
            </Button>
          )}
          <span className="novel-shelf-spacer" />
          <Popconfirm
            key="delete"
            title="确定删除？"
            description={`「${card.title}」的所有数据将被永久删除`}
            onConfirm={(e) => { e?.stopPropagation(); onDelete(card) }}
            onCancel={(e) => e?.stopPropagation()}
            okText="删除"
            cancelText="取消"
            okButtonProps={{ danger: true }}
          >
            <Button
              type="text" size="small" danger
              icon={<DeleteOutlined />}
              aria-label={`删除「${card.title}」`}
              onClick={(e) => e.stopPropagation()}
            />
          </Popconfirm>
        </div>
      </div>
    </div>
  )
}

export default ProjectCardItem
