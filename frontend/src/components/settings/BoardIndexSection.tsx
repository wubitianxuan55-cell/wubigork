import React from 'react'
import { Button } from 'antd'
import { RightOutlined } from '@ant-design/icons'
import { emitFrontendEvent, FRONTEND_EVENTS } from '../../events'
import type { BoardId } from '../../boards/manifests'

/**
 * BoardIndexSection — 板块域索引卡（设置中枢批 7 拍板的挂账收口）。
 *
 * 「使用中即时调」型的板块显示偏好（进度计划网络图样式/记忆中枢星图显示设置）
 * 按**边界留域**拍板不收编设置页——偏好面留驻板块（单一事实源，避免双处副本
 * 漂移），设置页只给索引卡：说明 + 可调项清单 + 一键跳板块页（NAVIGATE 白名单
 * 直达，keepAlive 板块往返不丢现场）。
 */
export interface BoardIndexSectionProps {
  /** 卡题（如「进度计划偏好（板块内即时调）」） */
  title: string
  /** 留域说明：为何设置页不复制该偏好面 */
  desc: string
  /** 可调项清单（胶囊呈现，锚板块内真实入口文案） */
  items: string[]
  /** 目标板块 id（navigate 白名单=manifest 派生，非法 id 事件被壳层忽略） */
  page: BoardId
  /** 跳转按钮文案 */
  openLabel: string
}

const BoardIndexSection: React.FC<BoardIndexSectionProps> = ({ title, desc, items, page, openLabel }) => (
  <section className="v3-card board-index-card" aria-label={title}>
    <div className="board-index-card__head">
      <strong className="board-index-card__title">{title}</strong>
      <Button
        size="small"
        icon={<RightOutlined />}
        onClick={() => emitFrontendEvent(FRONTEND_EVENTS.NAVIGATE, { page })}
      >
        {openLabel}
      </Button>
    </div>
    <p className="board-index-card__desc">{desc}</p>
    <div className="board-index-card__items">
      {items.map((it) => (
        <span key={it} className="board-index-card__item">{it}</span>
      ))}
    </div>
  </section>
)

export default BoardIndexSection
