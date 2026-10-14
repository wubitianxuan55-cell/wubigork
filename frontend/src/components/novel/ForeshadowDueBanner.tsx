// ForeshadowDueBanner.tsx — 伏笔到期预警条（阶段八 8.6）
// 判定全部在后端（types.UrgencyLevel 单源）：urgent=急需回收+已超期（红），
// soon=需关注（黄）。挂载点=章节计划卡（写计划正是回收决策点）+ 阅读页（编辑
// chrome 下方薄条）。零预警零渲染（不占版面）。
import React, { useEffect, useState } from 'react'
import { Alert, Tag, Tooltip } from 'antd'
import { app } from '../../gaea/lib/bridge'

export interface ForeshadowDueItemView {
  id: string
  title: string
  status: string
  plantedIn?: string
  targetIn?: string
  urgency: number
  overdueBy?: number
  longTerm?: boolean
}

export interface ForeshadowDueReportView {
  chapterNum: number
  urgent: ForeshadowDueItemView[]
  soon: ForeshadowDueItemView[]
}

interface ForeshadowDueBannerProps {
  /** 当前章号；<=0 时后端按全书已写章数回退。 */
  chapterNum: number | null
  /** 刷新键（父层切换章节/保存后自增触发重拉）。 */
  refreshKey?: number
}

const dueStatusLabel = (s: string) =>
  s === 'revealed' || s === 'resolved' ? '已回收'
    : s === 'partial' || s === 'partially_resolved' ? '部分回收'
      : s === 'hinted' ? '已暗示'
        : s === 'pending' ? '待埋入'
          : s || '已埋入'

const itemText = (it: ForeshadowDueItemView) => {
  const target = it.targetIn ? `计划第 ${it.targetIn.replace(/\.md$/, '')} 章回收` : '未排回收章'
  const over = it.overdueBy ? `，已超期 ${it.overdueBy} 章` : ''
  return `${it.title || it.id}（${dueStatusLabel(it.status)}，${target}${over}）`
}

/**
 * 伏笔到期预警条：urgent>0 红色（急需/超期），否则 soon>0 黄色（需关注），
 * 两者皆空不渲染。数据失败静默（预警是增强面，不打断写作主链路）。
 */
const ForeshadowDueBanner: React.FC<ForeshadowDueBannerProps> = ({ chapterNum, refreshKey = 0 }) => {
  const [rep, setRep] = useState<ForeshadowDueReportView | null>(null)

  useEffect(() => {
    let alive = true
    app.ForeshadowDue(chapterNum ?? 0)
      .then((raw) => {
        if (!alive) return
        const r = raw as Record<string, unknown>
        setRep({
          chapterNum: (r.chapterNum as number) ?? 0,
          urgent: (r.urgent as ForeshadowDueItemView[]) || [],
          soon: (r.soon as ForeshadowDueItemView[]) || [],
        })
      })
      .catch(() => { /* 预警失败静默：增强面不报错打扰 */ })
    return () => { alive = false }
  }, [chapterNum, refreshKey])

  if (!rep) return null
  const all = [...rep.urgent, ...rep.soon]
  if (all.length === 0) return null
  const over = rep.urgent.filter((it) => it.overdueBy)
  const mustNow = rep.urgent.filter((it) => !it.overdueBy)

  const message = over.length > 0
    ? `伏笔回收预警：${over.length} 条已超期${mustNow.length ? `、${mustNow.length} 条本章应回收` : ''}`
    : `伏笔回收提醒：${mustNow.length} 条本章应回收`

  return (
    <div className="novel-create-alert" data-testid="foreshadow-due-banner">
      <Alert
        type={over.length > 0 ? 'error' : mustNow.length > 0 ? 'warning' : 'info'}
        showIcon
        message={message}
        description={
          <div>
            {all.slice(0, 4).map((it) => (
              <div key={it.id}>
                {it.urgency >= 2 ? <Tag color={it.overdueBy ? 'red' : 'orange'}>{it.overdueBy ? `超期${it.overdueBy}章` : '本章回收'}</Tag> : <Tag>临近</Tag>}
                <Tooltip title={itemText(it)}>
                  <span>{itemText(it)}</span>
                </Tooltip>
                {it.longTerm ? <Tag>长线</Tag> : null}
              </div>
            ))}
            {all.length > 4 && <div style={{ opacity: 0.75 }}>…等共 {all.length} 条（详情见设定页伏笔面板）</div>}
          </div>
        }
      />
    </div>
  )
}

export default ForeshadowDueBanner
