// ChapterReviewPanel.tsx — 平台质量评审面板（小说创作间 CreatePage rail 弹窗）
// 受控 presentational Modal（形态参照 StyleFingerprintPanel）：
//   1. 档位选择——通用 / 番茄小说 / 起点中文网 / 知乎盐言故事（Go 侧 rubric 数据资产）；
//   2. 结论——APPROVE / CONCERNS / REJECT + S1~S4 计数 + 字数；
//   3. 逐维结果——pass/warn/fail/skip 四态 + severity Tag + 说明 + 改法 + 原文证据摘录；
//   4. 黄金三问（写手自检）固定说明。
// 数据获取全部由 CreatePage 持有（open 时拉档位清单，onReview 触发评审），
// 本组件纯展示、props 驱动直测（见 ChapterReviewPanel.test.tsx）。
// 契约：NovelReviewPlatforms / NovelChapterReview 两绑定（gaea/lib/bridge/novel.ts，Go NovelB 门面）；
// 后端 omitempty 字段可能缺省，数值/列表展示一律 ?. 与 ?? 防御。
import React from 'react'
import { Button, Modal, Select, Tag } from 'antd'
import type { ChapterReviewPayload, ReviewPlatform } from '../../gaea/lib/bridge/novel'

/** verdict → antd Tag 色（pass=green / warn=gold / fail=red / skip=default）。 */
const VERDICT_COLORS: Record<string, string> = {
  pass: 'green',
  warn: 'gold',
  fail: 'red',
  skip: 'default',
}

/** verdict → 中文档位（未知档透传原文）。 */
const VERDICT_LABELS: Record<string, string> = {
  pass: '通过',
  warn: '提醒',
  fail: '不达标',
  skip: '跳过',
}

/** severity → Tag 色（S1 打回 / S2 顾虑 / S3 局部 / S4 建议）。 */
const SEVERITY_COLORS: Record<string, string> = {
  S1: 'red',
  S2: 'orange',
  S3: 'blue',
  S4: 'default',
}

/** 结论语义（阈值与后端 rubric verdict 同口径：无 S1/S2 → 可发；有 S2 → 有顾虑；有 S1 → 打回）。 */
function verdictSemantics(verdict?: string): { label: string; color: string; hint: string } {
  switch (verdict) {
    case 'APPROVE':
      return { label: '可发', color: 'var(--color-success)', hint: '无 S1/S2，S3 可快速处理' }
    case 'CONCERNS':
      return { label: '有顾虑', color: 'var(--color-warning)', hint: '有 S2，或 S3 数量影响阅读' }
    case 'REJECT':
      return { label: '打回', color: 'var(--color-destructive)', hint: '有 S1，或核心卖点/动机/规则崩坏' }
    default:
      return { label: verdict ?? '—', color: 'var(--color-text-secondary)', hint: '' }
  }
}

interface ChapterReviewPanelProps {
  open: boolean
  onClose: () => void
  /** 评审动作进行中 */
  busy: boolean
  /** 动作结果/错误中文提示（CreatePage reviewMsg） */
  msg: string
  /** 档位清单（CreatePage open 时拉取；空表=选择器只显示当前档） */
  platforms: ReviewPlatform[]
  /** 当前档位 id（默认 general） */
  platformId: string
  onPlatformChange: (id: string) => void
  /** 评审报告（未评审为 null） */
  report: ChapterReviewPayload | null
  onReview: () => void
  /** 是否存在可评审的当前章（无章时按钮禁用） */
  hasChapter: boolean
}

const ChapterReviewPanel: React.FC<ChapterReviewPanelProps> = ({
  open, onClose, busy, msg, platforms, platformId, onPlatformChange, report, onReview, hasChapter,
}) => {
  const sem = verdictSemantics(report?.verdict)
  const counts = report?.counts ?? {}
  const dims = report?.dimensions ?? []
  // 先列 fail/warn（按 S1→S4），再列 pass/skip——S1/S2 优先可见。
  const rank = (d: { verdict?: string; severity?: string }): number => {
    if (d.verdict === 'fail') return d.severity === 'S1' ? 0 : 1
    if (d.verdict === 'warn') return d.severity === 'S1' ? 0 : 2
    return 3
  }
  const ordered = [...dims].sort((a, b) => rank(a) - rank(b))
  const problemCount = (counts.S1 ?? 0) + (counts.S2 ?? 0) + (counts.S3 ?? 0) + (counts.S4 ?? 0)

  return (
    <Modal
      title="平台评审（质量 rubric）"
      open={open}
      onCancel={onClose}
      width={680}
      footer={[
        <Button key="review" type="primary" size="small" loading={busy} disabled={!hasChapter} onClick={onReview}>
          评审当前章
        </Button>,
        <Button key="close" size="small" onClick={onClose}>关闭</Button>,
      ]}
    >
      {msg ? (
        <div style={{ fontSize: 12, marginBottom: 8, color: 'var(--color-text-secondary)' }}>{msg}</div>
      ) : null}

      {/* 1. 档位选择 */}
      <div style={{ marginBottom: 12, display: 'flex', alignItems: 'center', gap: 8 }}>
        <span style={{ fontSize: 12, color: 'var(--color-text-secondary)' }}>目标平台</span>
        <Select
          size="small"
          style={{ minWidth: 180 }}
          value={platformId}
          onChange={onPlatformChange}
          options={(platforms.length > 0 ? platforms : [{ id: 'general', label: '通用', form: 'chapter' }]).map((p) => ({
            value: p.id,
            label: p.form === 'story' ? `${p.label}（整篇口径）` : p.label,
          }))}
        />
        <span style={{ fontSize: 12, color: 'var(--color-text-secondary)' }}>
          确定性评审，零模型调用；命中只提示，不自动改稿
        </span>
      </div>

      {/* 2. 结论 */}
      <div style={{ marginBottom: 14 }}>
        {!report ? (
          <div style={{ fontSize: 12, color: 'var(--color-text-secondary)', padding: '8px 0' }}>
            点「评审当前章」按所选平台档位逐维体检；每条发现都带原文证据，可对照修改。
          </div>
        ) : (
          <>
            <div style={{ display: 'flex', alignItems: 'baseline', gap: 8, marginBottom: 4 }}>
              <span style={{ fontSize: 26, fontWeight: 700, color: sem.color }}>{sem.label}</span>
              <span style={{ fontSize: 12, color: 'var(--color-text-secondary)' }}>
                {report.platformLabel} · 第 {report.chapterNum} 章 · {(report.words ?? 0).toLocaleString()} 字 · 命中 {problemCount} 项
              </span>
            </div>
            {sem.hint ? (
              <div style={{ fontSize: 12, color: 'var(--color-text-secondary)', marginBottom: 4 }}>{sem.hint}</div>
            ) : null}
            <div style={{ display: 'flex', gap: 6, marginBottom: 4 }}>
              {(['S1', 'S2', 'S3', 'S4'] as const).map((sev) => (
                <Tag key={sev} color={counts[sev] ? SEVERITY_COLORS[sev] : 'default'} style={{ marginInlineEnd: 0 }}>
                  {sev} {counts[sev] ?? 0}
                </Tag>
              ))}
            </div>
          </>
        )}
      </div>

      {/* 3. 逐维结果 */}
      {report ? (
        <div style={{ marginBottom: 14 }}>
          <div style={{ fontWeight: 600, marginBottom: 6 }}>逐维结果</div>
          <div style={{ maxHeight: 300, overflow: 'auto' }}>
            {ordered.map((d) => (
              <div key={d.id} style={{ borderBottom: '1px solid var(--border-subtle)', padding: '6px 0' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 6, flexWrap: 'wrap' }}>
                  <Tag color={VERDICT_COLORS[d.verdict] ?? 'default'} style={{ marginInlineEnd: 0 }}>
                    {VERDICT_LABELS[d.verdict] ?? d.verdict}
                  </Tag>
                  {d.severity ? (
                    <Tag color={SEVERITY_COLORS[d.severity] ?? 'default'} style={{ marginInlineEnd: 0 }}>{d.severity}</Tag>
                  ) : null}
                  <span style={{ fontWeight: 500 }}>{d.label}</span>
                  <span style={{ fontSize: 12, color: 'var(--color-text-secondary)' }}>{d.detail}</span>
                </div>
                {(d.evidence ?? []).map((ev, i) => (
                  <div
                    key={`${d.id}-ev-${i}`}
                    style={{ borderLeft: '2px solid var(--color-border)', paddingLeft: 8, margin: '4px 0', fontSize: 12, color: 'var(--color-text-secondary)' }}
                  >
                    {ev.paragraph > 0 ? `第 ${ev.paragraph} 段：` : ''}「{ev.excerpt}」
                  </div>
                ))}
                {d.advice ? (
                  <div style={{ fontSize: 12, color: 'var(--color-text-secondary)' }}>改法：{d.advice}</div>
                ) : null}
              </div>
            ))}
          </div>
        </div>
      ) : null}

      {/* 4. 黄金三问 + 底部说明 */}
      {(report?.advisories ?? []).length > 0 ? (
        <div style={{ fontSize: 12, marginBottom: 8 }}>
          <div style={{ fontWeight: 600, marginBottom: 4 }}>黄金三问（逐问作答）</div>
          <ol style={{ margin: 0, paddingInlineStart: 18, color: 'var(--color-text-secondary)' }}>
            {(report?.advisories ?? []).map((q, i) => (
              <li key={`adv-${i}`}>{q}</li>
            ))}
          </ol>
        </div>
      ) : null}
      <div style={{ fontSize: 12, color: 'var(--color-text-secondary)', borderTop: '1px solid var(--border-subtle)', paddingTop: 8 }}>
        口径：通用 18 维 + 确定性扩展（字数区间/对话占比/预告式收尾/破折号/格式/人称/主角/金手指）。
        语义维度（卖点·动机·伏笔）由评审技能按原文证据判定；引擎对缺数据的维度显式跳过，不静默给通过。
      </div>
    </Modal>
  )
}

export default ChapterReviewPanel
