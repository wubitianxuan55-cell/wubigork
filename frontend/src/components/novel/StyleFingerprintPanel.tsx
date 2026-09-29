// StyleFingerprintPanel.tsx — 文风指纹面板（小说创作间 CreatePage rail 弹窗）
// 受控 presentational Modal（形态参照 CreatePage「叙事状态账本」弹窗）：
//   1. 参考档状态——未构建给空态引导；已构建给元信息行 + 摘要格子 + 口头禅 Tag；
//   2. 章节体检——AI 味大数字 + 语义分档 + 基线距离 Δ + 命中问题列表；
//   3. 底部小字说明（参考档不随写作漂移，去味/手改后可重建）。
// 数据获取全部由 CreatePage 持有（open 时拉 Status，onBuild/onScore 触发动作），
// 本组件纯展示、props 驱动直测（见 StyleFingerprintPanel.test.tsx）。
// 契约：NovelFingerprint* 三绑定（gaea/lib/bridge/novel.ts，Go NovelB 门面）；
// 后端 omitempty 字段可能缺省，数值/列表展示一律 ?. 与 ?? 防御。
import React, { useCallback, useEffect, useRef, useState } from 'react'
import { Button, Modal, Tag, Typography, message } from 'antd'
import { useAppStore } from '../../stores/appStore'
import { app } from '../../gaea/lib/bridge'
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
  // v4.425：参考档 / 章节体检报告都不含书名，切书后整屏仍是上一本的分数且看不出归属。
  // 数据由 CreatePage 持有（本组件纯展示），父层清得掉；这里再按 projectPath 兜一道：
  // 报告绑在切换前那本书的上下文里，切书即视为失效。用 ref 记录「这份报告出自哪本书」，
  // 并在 effect 里随本次提交更新——切书的下一次渲染当帧就看到不一致，不经过 setState
  // 的二次渲染，避免一帧旧报告闪现。
  const projectPath = useAppStore((s) => s.projectPath)
  const reportPathRef = useRef(projectPath)
  useEffect(() => { reportPathRef.current = projectPath }, [projectPath])
  const status1 = reportPathRef.current === projectPath ? status : null
  const score1 = reportPathRef.current === projectPath ? score : null
  const exists = status1?.exists ?? false
  const summary = status1?.summary
  // 口头禅 = 双字组合 + 三字组合 + 签名词平铺；三源全空则不显区块
  const signWords = [
    ...(summary?.topBigrams ?? []),
    ...(summary?.topTrigrams ?? []),
    ...(summary?.authorSignWords ?? []),
  ]
  const issues = score1?.issues ?? []
  const sem = scoreSemantics(score1?.score ?? 0)

  // ── 长篇刀5：风格学习回灌（自治 state：构建/预览/清除，生成时自动注入）──
  const [digest, setDigest] = useState<{ exists?: boolean; builtAt?: string; chapters?: number; chars?: number; instructions?: string } | null>(null)
  const [digestBusy, setDigestBusy] = useState(false)
  const [digestOpen, setDigestOpen] = useState(false)
  const loadDigest = useCallback(async () => {
    try { setDigest((await app.NovelStyleDigestGet()) as never) } catch { /* 留旧态 */ }
  }, [])
  useEffect(() => { if (open) void loadDigest() }, [open, loadDigest])
  const buildDigest = async () => {
    setDigestBusy(true)
    try {
      const r = (await app.NovelStyleDigestBuild()) as { instructions?: string } | null
      message.success('已学习你的成稿风格：生成（整章/逐场景）将自动按此口径约束')
      setDigest((prev) => ({ ...prev, exists: true, builtAt: new Date().toISOString(), instructions: r?.instructions ?? prev?.instructions }))
    } catch (e) {
      message.error(e instanceof Error ? e.message : '学习失败')
    } finally { setDigestBusy(false) }
  }
  const clearDigest = async () => {
    setDigestBusy(true)
    try {
      await app.NovelStyleDigestClear()
      message.info('已清除风格回灌（文风指纹体检档不受影响）')
      setDigest({ exists: false })
    } catch (e) {
      message.error(e instanceof Error ? e.message : '清除失败')
    } finally { setDigestBusy(false) }
  }

  return (
    <Modal
      title="文风指纹"
      open={open}
      onCancel={onClose}
      width={640}
      footer={[
        <Button key="digest" size="small" loading={digestBusy} onClick={() => void buildDigest()} data-testid="digest-build">
          {digest?.exists ? '重建风格回灌' : '学习我的成稿（生成回灌）'}
        </Button>,
        digest?.exists ? <Button key="digest-clear" size="small" danger disabled={digestBusy} onClick={() => void clearDigest()}>停用回灌</Button> : null,
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
              {status1?.chapters ?? 0} 章 · {(status1?.chars ?? 0).toLocaleString()} 字 · 构建于 {fmtBuiltAt(status1?.builtAt) || '—'}
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
        {!score1 ? (
          <div style={{ fontSize: 12, color: 'var(--color-text-secondary)', padding: '8px 0' }}>
            点击「体检当前章」获取本章 AI 味评分；构建参考档后同时给出与基线的距离。
          </div>
        ) : (
          <>
            <div style={{ display: 'flex', alignItems: 'baseline', gap: 8, marginBottom: 4 }}>
              <span style={{ fontSize: 32, fontWeight: 700, color: sem.color }}>{score1.score}</span>
              <span style={{ color: sem.color }}>AI 味 {sem.label}</span>
            </div>
            {typeof score1.delta === 'number' ? (
              <div style={{ fontSize: 12, marginBottom: 4 }}>与你的基线距离 Δ {score1.delta.toFixed(2)}（越小越像你）</div>
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

      {/* 2.5 风格学习回灌（长篇刀5）：已学习给状态+指令预览；未学习给引导 */}
      <div style={{ marginTop: 12, borderTop: '1px solid var(--border-subtle)', paddingTop: 8 }} data-testid="digest-section">
        {digest?.exists ? (
          <>
            <div style={{ display: 'flex', gap: 6, alignItems: 'center', flexWrap: 'wrap' }}>
              <Tag color="green">风格回灌已启用</Tag>
              <Typography.Text style={{ fontSize: 12 }}>
                {digest.chapters ?? 0} 章 · {(digest.chars ?? 0).toLocaleString()} 字 · 学习于 {fmtBuiltAt(digest.builtAt) || '—'}
              </Typography.Text>
              <Button size="small" type="link" style={{ padding: 0, fontSize: 12 }} onClick={() => setDigestOpen((v) => !v)} data-testid="digest-toggle">
                {digestOpen ? '收起指令' : '查看注入的写作指令'}
              </Button>
            </div>
            <Typography.Text style={{ fontSize: 12, color: 'var(--color-text-secondary)' }}>
              生成（整章/逐场景）会自动带上这些从你成稿学到的表达习惯；不想用时点「停用回灌」。
            </Typography.Text>
            {digestOpen && (
              <pre style={{ fontSize: 12, whiteSpace: 'pre-wrap', margin: '6px 0 0', padding: 8, background: 'var(--bg-subtle)', borderRadius: 6 }} data-testid="digest-instructions">
                {digest.instructions ?? ''}
              </pre>
            )}
          </>
        ) : (
          <Typography.Text style={{ fontSize: 12, color: 'var(--color-text-secondary)' }} data-testid="digest-empty">
            风格回灌未启用：点「学习我的成稿」从已写章节提取你的句长节奏/对话占比/惯用词等表达习惯，生成时自动约束（与文风指纹体检相互独立）。
          </Typography.Text>
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
