import React, { useEffect, useState } from 'react'
import { Button, Tooltip } from 'antd'
import {
  ControlOutlined, MenuFoldOutlined,
  FileTextOutlined, BookOutlined, ThunderboltOutlined, UserOutlined,
  ReadOutlined, InfoCircleOutlined, SafetyCertificateOutlined,
} from '@ant-design/icons'
import { RunChapterGate } from '../../../wailsjs/go/app/NovelB'
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
  chapterNum?: number
}

/** RunChapterGate 合并报告（四路，单路失败/缺省时为 null——诚实降级） */
type GateReport = Record<string, unknown> | null

/** 窄化体检报告的某一路 → 数值/字符串列表，供紧凑展示。 */
function gateRouteOf(report: GateReport | null, key: string): Record<string, unknown> | null {
  if (!report) return null
  const v = (report as Record<string, unknown>)[key]
  return typeof v === 'object' && v !== null ? v as Record<string, unknown> : null
}

function gateNum(route: Record<string, unknown> | null, key: string): number | undefined {
  const v = route?.[key]
  return typeof v === 'number' ? v : undefined
}

/** 取一路报告里第一条字符串建议（weaknesses / improvement_tips），无则空。 */
function gateFirstTip(route: Record<string, unknown> | null, key: string): string {
  const v = route?.[key]
  if (Array.isArray(v)) {
    const first = v.find((x) => typeof x === 'string' && x.trim())
    return typeof first === 'string' ? first : ''
  }
  return ''
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
  // 章节体检（RunChapterGate 单章合并报告；手动触发，非每章盖章向导）
  const [gate, setGate] = useState<GateReport>(null)
  const [gateBusy, setGateBusy] = useState(false)
  const [gateErr, setGateErr] = useState('')

  // 订阅阅读页上报的当前章节属性
  useEffect(() => {
    const handler = (e: Event) => {
      const data = (e as CustomEvent<ChapterActivePayload>).detail
      if (data) {
        setChapter(data)
        setGate(null) // 换章即弃旧报告
        setGateErr('')
      }
    }
    window.addEventListener('novel:chapter-active', handler)
    return () => window.removeEventListener('novel:chapter-active', handler)
  }, [])

  const runGate = async () => {
    const num = chapter?.chapterNum
    if (!num || num < 1 || gateBusy) return
    setGateBusy(true)
    setGateErr('')
    try {
      setGate(await RunChapterGate(num))
    } catch (e) {
      setGateErr(e instanceof Error ? e.message : '体检运行失败')
    } finally {
      setGateBusy(false)
    }
  }

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

  const renderChapterSection = () => {
    const review = gateNum(gateRouteOf(gate, 'review'), 'score')
    const analysis = gateNum(gateRouteOf(gate, 'analysis'), 'quality_score')
    const consistency = gateNum(gateRouteOf(gate, 'consistency'), 'total_issues')
    const aiTaste = gateNum(gateRouteOf(gate, 'aiTaste'), 'score')
    const reviewTip = gateFirstTip(gateRouteOf(gate, 'review'), 'weaknesses')
    const analysisTip = gateFirstTip(gateRouteOf(gate, 'analysis'), 'improvement_tips')
    return (
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
              <span className={`novel-tag-tone ${chapter.saved ? 'is-success' : 'is-warning'}`}>
                {chapter.saved ? '已保存' : '未保存'}
              </span>
            </span>
          </div>
          {/* 章节体检：单章四路合并报告，手动触发（分析/审查/一致性/AI味） */}
          {chapter.chapterNum != null && chapter.chapterNum >= 1 && (
            <div className="novel-inspector-item" style={{ display: 'block' }}>
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 4 }}>
                <span className="novel-inspector-item-label"><SafetyCertificateOutlined /> 章节体检</span>
                <Button size="small" icon={<ThunderboltOutlined />} loading={gateBusy}
                  onClick={() => void runGate()} style={{ fontSize: 11 }}>
                  {gate ? '重跑' : '运行'}
                </Button>
              </div>
              {gateErr ? (
                <span className="novel-inspector-hint">{gateErr}</span>
              ) : gate ? (
                <div style={{ display: 'grid', gap: 2, fontSize: 11 }}>
                  <span>章节质量：<b>{review ?? '未启用'}</b>{review != null ? ' /10' : ''}</span>
                  <span>情节质量：<b>{analysis ?? '未启用'}</b>{analysis != null ? ' /100' : ''}</span>
                  <span>一致性：{consistency != null ? <b>{consistency}</b> : '未启用'}{consistency != null ? ' 处疑点' : ''}</span>
                  <span>AI 味：<b>{aiTaste ?? '未启用'}</b>{aiTaste != null ? ' 分' : ''}</span>
                  {(reviewTip || analysisTip) ? (
                    <span className="novel-inspector-hint" style={{ marginTop: 2 }}>
                      {reviewTip || analysisTip}
                    </span>
                  ) : null}
                </div>
              ) : (
                <span className="novel-inspector-hint">合并分析/审查/一致性/AI 味四路，出一份单章报告。</span>
              )}
            </div>
          )}
        </>
      ) : (
        <div className="novel-inspector-hint">从左侧大纲选择章节后，这里会显示该章节的属性。</div>
      )}
    </section>
    )
  }

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
          <div className="novel-inspector-section-title"><BookOutlined />本书</div>
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
              世界观设定支持编辑 / 分屏 / 渲染 / 维度化四种模式；创作生成时会自动注入最新设定。
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
