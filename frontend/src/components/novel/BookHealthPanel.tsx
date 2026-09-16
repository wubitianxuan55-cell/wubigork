// BookHealthPanel.tsx — 全书体检面板（GenerationGate 闭环收口，规格
// 进度计划/gaea-gen-gate-closure-20260916.md）：触发式全量编译，纯确定性
// 零 LLM，作者点按跑完出报告（无常驻无定时）。消费 RunBookHealthCheck：
// 头部聚合卡（总章/契约问题章/质量问题章/最差 AI 味/V2 覆盖/伏笔 findings）
// + 逐章表（章号/字数/契约/质量/AI 味，越线红标）+ 伏笔 findings 列表。
import React, { useCallback, useEffect, useState } from 'react'
import { Alert, Button, Empty, Modal, Spin, Table, Tag, Typography, message } from 'antd'
import { app } from '../../gaea/lib/bridge'
import type { BookHealthReportView } from '../../gaea/lib/bridge/novel'

const softTextStyle: React.CSSProperties = { fontSize: 12, color: 'var(--v3-fg-soft, #6b7280)' }

/** 逐章体检行（JSX 泛型不认索引访问，抽别名）。 */
type HealthRow = BookHealthReportView['chapters'][number]

export default function BookHealthPanel({ open, onClose }: {
  open: boolean
  onClose: () => void
}) {
  const [loading, setLoading] = useState(false)
  const [report, setReport] = useState<BookHealthReportView | null>(null)

  const run = useCallback(async () => {
    setLoading(true)
    try {
      setReport(await app.RunBookHealthCheck())
    } catch (e) {
      message.error(`全书体检失败：${e instanceof Error ? e.message : String(e)}`)
      setReport(null)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (open) void run()
  }, [open, run])

  const fs = report?.foreshadow
  const findings = fs?.findings ?? []

  return (
    <Modal open={open} title="全书体检（确定性 · 零模型调用）" onCancel={onClose} width={780} destroyOnHidden
      footer={[
        <Button key="rerun" size="small" loading={loading} onClick={() => void run()}>重新体检</Button>,
        <Button key="close" size="small" onClick={onClose}>关闭</Button>,
      ]}>
      {loading ? (
        <div style={{ padding: '48px 0', textAlign: 'center' }}><Spin /><div style={{ ...softTextStyle, marginTop: 8 }}>全量编译中（逐章写前契约 / 写后质量 / AI 味 / 伏笔登记）…</div></div>
      ) : !report ? (
        <Empty description="没有体检结果" style={{ padding: '32px 0' }} />
      ) : (
        <div data-testid="book-health-body" style={{ maxHeight: '66vh', overflowY: 'auto' }}>
          {/* 聚合卡 */}
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 12 }}>
            <Tag style={{ marginRight: 0 }}>共 {report.totalChapters} 章</Tag>
            <Tag color={report.contractIssueChapters ? 'orange' : undefined} style={{ marginRight: 0 }}>契约问题章 {report.contractIssueChapters}</Tag>
            <Tag color={report.qualityIssueChapters ? 'orange' : undefined} style={{ marginRight: 0 }}>质量问题章 {report.qualityIssueChapters}</Tag>
            {report.worstAiTaste && (
              <Tag color="red" style={{ marginRight: 0 }}>最差 AI 味 {report.worstAiTaste.aiTasteScore}（第 {report.worstAiTaste.chapterNum} 章）</Tag>
            )}
            <Tag style={{ marginRight: 0 }}>分析覆盖 {report.analyzedChapters}/{report.totalChapters}</Tag>
            {fs && <Tag style={{ marginRight: 0 }}>伏笔 {fs.items} 条{findings.length ? ` · ${findings.length} 项告警` : ''}</Tag>}
          </div>
          {report.worstAiTaste && report.worstAiTaste.aiTasteScore >= 60 && (
            <Alert type="warning" showIcon style={{ marginBottom: 12 }}
              message={`第 ${report.worstAiTaste.chapterNum} 章 AI 味 ${report.worstAiTaste.aiTasteScore} 分`}
              description="建议用「一键去味 / 高级去味」处理后复检（60 分以上视为明显 AI 味）。" />
          )}
          {/* 逐章表 */}
          <Table<HealthRow> size="small" rowKey="chapterNum"
            dataSource={report.chapters ?? []} pagination={false}
            locale={{ emptyText: '还没有已写章节' }}
            columns={[
              { title: '章', dataIndex: 'chapterNum', width: 56 },
              { title: '字数', dataIndex: 'words', width: 80, render: (v: number) => <span style={{ fontVariantNumeric: 'tabular-nums' }}>{v}</span> },
              {
                title: '写前契约', width: 100, render: (_v, row) =>
                  row.outlineIssues > 0
                    ? <span style={{ color: row.outlineErrors > 0 ? 'var(--color-error, var(--md-sys-color-error, #cf1322))' : undefined }}>{row.outlineIssues} 项{row.outlineErrors > 0 ? `（${row.outlineErrors} 阻断级）` : ''}</span>
                    : <span style={softTextStyle}>—</span>,
              },
              {
                title: '写后质量', width: 100, render: (_v, row) =>
                  row.qualityIssues > 0
                    ? <span style={{ color: row.qualityErrors > 0 ? 'var(--color-error, var(--md-sys-color-error, #cf1322))' : undefined }}>{row.qualityIssues} 项{row.qualityErrors > 0 ? `（${row.qualityErrors} 阻断级）` : ''}</span>
                    : <span style={softTextStyle}>—</span>,
              },
              {
                title: 'AI 味', dataIndex: 'aiTasteScore', width: 80, render: (v: number) =>
                  v < 0 ? <span style={softTextStyle}>—</span> : (
                    <span style={{ color: v >= 60 ? 'var(--color-error, var(--md-sys-color-error, #cf1322))' : undefined, fontVariantNumeric: 'tabular-nums' }}>{v}</span>
                  ),
              },
            ]} />
          {/* 伏笔 findings */}
          {findings.length > 0 && (
            <div style={{ marginTop: 14 }}>
              <div style={{ fontSize: 13, fontWeight: 600, marginBottom: 6 }}>伏笔登记告警（{findings.length}）</div>
              {findings.map((f, i) => (
                <div key={i} style={{ padding: '3px 0', fontSize: 12.5, display: 'flex', gap: 6 }}>
                  <Tag color="purple" style={{ marginRight: 0, flexShrink: 0 }}>{String(f.code ?? 'lint')}</Tag>
                  <span>{String(f.message ?? '')}</span>
                </div>
              ))}
            </div>
          )}
          <Typography.Text type="secondary" style={{ fontSize: 12, display: 'block', marginTop: 12 }}>
            体检为纯确定性检查（写前契约齐备性 / 段落句式标点 / AI 味高频词 / 伏笔登记自洽），零模型调用；
            逐章 AI 深检（情节分析/审阅/一致性）在章节页「章节体检」单章触发。
          </Typography.Text>
        </div>
      )}
    </Modal>
  )
}
