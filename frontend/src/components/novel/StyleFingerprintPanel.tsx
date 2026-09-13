// StyleFingerprintPanel.tsx — 文风指纹面板（小说创作间 CreatePage rail 弹窗）
// 受控 presentational Modal（形态参照 CreatePage「叙事状态账本」弹窗）：
//   1. 参考档状态——未构建给空态引导；已构建给元信息行 + 摘要格子 + 口头禅 Tag；
//   2. 章节体检——AI 味大数字 + 语义分档 + 基线距离 Δ + 命中问题列表；
//   3. 底部小字说明（参考档不随写作漂移，去味/手改后可重建）。
// 数据获取全部由 CreatePage 持有（open 时拉 Status，onBuild/onScore 触发动作），
// 本组件纯展示、props 驱动直测（见 StyleFingerprintPanel.test.tsx）。
// 契约：NovelFingerprint* 三绑定（gaea/lib/bridge/novel.ts，Go NovelB 门面）；
// 后端 omitempty 字段可能缺省，数值/列表展示一律 ?. 与 ?? 防御。
import React from 'react'
import { Button, Modal, Tag } from 'antd'
import type { FingerprintScorePayload, FingerprintStatusPayload, FingerprintSummary } from '../../gaea/lib/bridge/novel'

/** FingerprintSummary 中的数值字段键（排除 topBigrams/topTrigrams/authorSignWords 等 string[] 源）。 */
type FingerprintNumberKey = {
  [K in keyof FingerprintSummary]: FingerprintSummary[K] extends number ? K : never
}[keyof FingerprintSummary]

/** 摘要格子指标（顺序即展示序；label 全中文，对齐后端 FingerprintSummary 字段）。 */
const SUMMARY_METRICS: Array<{ key: FingerprintNumberKey; label: string }> = [
  { key: 'sentenceMean', label: '句长均值' },
  { key: 'sentenceSd', label: '句长标准差' },
  { key: 'paraMean', label: '段长均值' },
  { key: 'ttr1000', label: '词汇多样 TTR' },
  { key: 'dialogRatio', label: '对话占比' },
  { key: 'fourCharRatio', label: '四字格密度' },
  { key: 'connectiveDensity', label: '连接词密度' },
  { key: 'adjAdvDensity', label: '形副密度' },
]

/** severity → antd Tag 色（low=default / medium=blue / high=orange / blocker=red）。 */
const SEVERITY_COLORS: Record<string, string> = {
  low: 'default',
  medium: 'blue',
  high: 'orange',
  blocker: 'red',
}

/** severity → 中文档位标签（未知档透传原文）。 */
const SEVERITY_LABELS: Record<string, string> = {
  low: '轻',
  medium: '中',
  high: '重',
  blocker: '严重',
}

/** score 语义分档（阈值与 CreatePage aiTaste 一致：<35 顺眼 / 35~59 有 AI 痕迹 / ≥60 AI 味偏重）。 */
function scoreSemantics(score: number): { label: string; color: string } {
  if (score >= 60) return { label: 'AI 味偏重', color: 'var(--color-warning)' }
  if (score >= 35) return { label: '有 AI 痕迹', color: 'var(--color-info)' }
  return { label: '顺眼', color: 'var(--color-success)' }
}

/** 数值展示：1~2 位小数；缺省/非法值兜底占位（后端 omitempty 可能不给）。 */
function fmtNum(v?: number): string {
  return typeof v === 'number' && Number.isFinite(v) ? v.toFixed(2) : '—'
}

/** 构建时间本地化展示（截断为本地可读串；解析失败回退原文）。 */
function fmtBuiltAt(builtAt?: string): string {
  if (!builtAt) return ''
  const d = new Date(builtAt)
  return Number.isNaN(d.getTime()) ? builtAt : d.toLocaleString()
}

interface StyleFingerprintPanelProps {
  open: boolean
  onClose: () => void
  /** 任一后端动作（构建/体检）进行中；两按钮共享 loading 态 */
  busy: boolean
  /** 动作结果/错误中文提示（CreatePage fpMsg） */
  msg: string
  status: FingerprintStatusPayload | null
  score: FingerprintScorePayload | null
  onBuild: () => void
  onScore: () => void
  /** 是否存在可体检的当前章（无章时「体检当前章」禁用） */
  hasChapter: boolean
}

const StyleFingerprintPanel: React.FC<StyleFingerprintPanelProps> = ({
  open, onClose, busy, msg, status, score, onBuild, onScore, hasChapter,
}) => {
  const exists = status?.exists ?? false
  const summary = status?.summary
  // 口头禅 = 双字组合 + 三字组合 + 签名词平铺；三源全空则不显区块
  const signWords = [
    ...(summary?.topBigrams ?? []),
    ...(summary?.topTrigrams ?? []),
    ...(summary?.authorSignWords ?? []),
  ]
  const issues = score?.issues ?? []
  const sem = scoreSemantics(score?.score ?? 0)

  return (
    <Modal
      title="文风指纹"
      open={open}
      onCancel={onClose}
      width={640}
      footer={[
        <Button key="build" type="primary" size="small" loading={busy} onClick={onBuild}>构建/重建参考档</Button>,
        <Button key="score" size="small" loading={busy} disabled={!hasChapter} onClick={onScore}>体检当前章</Button>,
      ]}
    >
      {msg ? (
        <div style={{ fontSize: 12, marginBottom: 8, color: 'var(--color-text-secondary)' }}>{msg}</div>
      ) : null}

      {/* 1. 参考档状态 */}
      <div style={{ marginBottom: 14 }}>
        <div style={{ fontWeight: 600, marginBottom: 6 }}>参考档状态</div>
        {!exists ? (
          <div style={{ fontSize: 12, color: 'var(--color-text-secondary)', padding: '8px 0' }}>
            用已有章节构建你自己的文风基线；构建后 AI 味体检将对照你的风格打分。
          </div>
        ) : (
          <>
            <div style={{ fontSize: 12, color: 'var(--color-text-secondary)', marginBottom: 6 }}>
              {status?.chapters ?? 0} 章 · {(status?.chars ?? 0).toLocaleString()} 字 · 构建于 {fmtBuiltAt(status?.builtAt) || '—'}
            </div>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, minmax(0, 1fr))', gap: 6, marginBottom: 8 }}>
              {SUMMARY_METRICS.map(({ key, label }) => (
                <div key={key} style={{ border: '1px solid var(--color-border)', borderRadius: 4, padding: '4px 8px' }}>
                  <div style={{ fontSize: 11, color: 'var(--color-text-secondary)' }}>{label}</div>
                  <div style={{ fontSize: 14 }}>{fmtNum(summary?.[key])}</div>
                </div>
              ))}
            </div>
            {signWords.length > 0 && (
              <div style={{ fontSize: 12 }}>
                口头禅：
                {signWords.map((w, i) => (
                  <Tag key={`${w}-${i}`} style={{ marginInlineEnd: 4 }}>{w}</Tag>
                ))}
              </div>
            )}
          </>
        )}
      </div>

      {/* 2. 章节体检 */}
      <div style={{ marginBottom: 14 }}>
        <div style={{ fontWeight: 600, marginBottom: 6 }}>章节体检</div>
        {!score ? (
          <div style={{ fontSize: 12, color: 'var(--color-text-secondary)', padding: '8px 0' }}>
            点击「体检当前章」获取本章 AI 味评分；构建参考档后同时给出与基线的距离。
          </div>
        ) : (
          <>
            <div style={{ display: 'flex', alignItems: 'baseline', gap: 8, marginBottom: 4 }}>
              <span style={{ fontSize: 32, fontWeight: 700, color: sem.color }}>{score.score}</span>
              <span style={{ color: sem.color }}>AI 味 {sem.label}</span>
            </div>
            {typeof score.delta === 'number' ? (
              <div style={{ fontSize: 12, marginBottom: 4 }}>与你的基线距离 Δ {score.delta.toFixed(2)}（越小越像你）</div>
            ) : (
              <div style={{ fontSize: 12, marginBottom: 4, color: 'var(--color-text-secondary)' }}>未构建参考档，按通用阈值打分</div>
            )}
            {issues.length === 0 ? (
              <div style={{ fontSize: 12, color: 'var(--color-text-secondary)', padding: '4px 0' }}>未命中明显 AI 套路</div>
            ) : (
              <div style={{ maxHeight: 220, overflow: 'auto' }}>
                {issues.map((iss, idx) => (
                  <div key={`${iss.start}-${iss.end}-${idx}`} style={{ borderBottom: '1px solid var(--border-subtle)', padding: '6px 0' }}>
                    <div>
                      <Tag color={SEVERITY_COLORS[iss.severity] ?? 'default'} style={{ marginInlineEnd: 6 }}>
                        {SEVERITY_LABELS[iss.severity] ?? iss.severity}
                      </Tag>
                      <span>{iss.reason}</span>
                    </div>
                    {iss.excerpt ? (
                      <div style={{ borderLeft: '2px solid var(--color-border)', paddingLeft: 8, margin: '4px 0', fontSize: 12, color: 'var(--color-text-secondary)' }}>
                        「{iss.excerpt}」
                      </div>
                    ) : null}
                    {iss.suggestion ? (
                      <div style={{ fontSize: 12, color: 'var(--color-text-secondary)' }}>建议：{iss.suggestion}</div>
                    ) : null}
                  </div>
                ))}
              </div>
            )}
          </>
        )}
      </div>

      {/* 3. 底部说明 */}
      <div style={{ fontSize: 12, color: 'var(--color-text-secondary)', borderTop: '1px solid var(--border-subtle)', paddingTop: 8 }}>
        参考档 = 构建时点全部已写章节的风格基线，不会随写作自动漂移；去味或手改后可重建更新基线。
      </div>
    </Modal>
  )
}

export default StyleFingerprintPanel
