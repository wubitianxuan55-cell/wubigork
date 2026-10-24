import React, { useEffect, useRef, useState } from 'react'
import { Modal, Button, InputNumber, Checkbox, Space, Tag, Typography, Alert } from 'antd'
import { C } from '../../utils/theme'
import { app } from '../../gaea/lib/bridge'
import type { NovelDissectEntry, NovelDissectReport } from '../../gaea/lib/bridge/novel'

interface DissectPanelProps {
  open: boolean
  onClose: () => void
}

/**
 * 对标拆解面板（oh-story story-long-analyze 机制重推导蒸馏，学习导向——
 * 与「章节分析」的修改导向分面）：
 *  - 逐章拆解走 NovelDissectChapter（前端串行循环：进度可见、可中断、
 *    天然断点续跑——skipExisting 开启时已拆章直接跳过不发 LLM）；
 *  - 清单/聚合报告（剧情线/钩子分布/写法公式/情绪曲线）零 LLM。
 * 产物落 <项目根>/dissections/，属于拆书参考，不写书稿数据。
 */
const DissectPanel: React.FC<DissectPanelProps> = ({ open, onClose }) => {
  const [from, setFrom] = useState<number | null>(1)
  const [to, setTo] = useState<number | null>(10)
  const [skipExisting, setSkipExisting] = useState(true)
  const [running, setRunning] = useState(false)
  const [progress, setProgress] = useState('')
  const [doneNums, setDoneNums] = useState<number[]>([])
  const [report, setReport] = useState<NovelDissectReport | null>(null)
  const [err, setErr] = useState('')
  const cancelRef = useRef(false)

  const loadList = async () => {
    try {
      const res = await app.NovelDissectList()
      setDoneNums((res.entries ?? []).map((e: NovelDissectEntry) => e.chapter).sort((a, b) => a - b))
    } catch {
      setDoneNums([]) // 清单失败不阻断面板（拆解时逐章如实报错）
    }
  }

  useEffect(() => {
    if (open) {
      cancelRef.current = false
      setRunning(false)
      setProgress('')
      setErr('')
      setReport(null)
      void loadList()
    }
  }, [open])

  const runDissect = async () => {
    if (running) return
    const start = from ?? 0
    const end = to ?? 0
    if (start <= 0 || end < start) {
      setErr('章号范围不合法：起止章号需为正数且起始 ≤ 结束')
      return
    }
    setRunning(true)
    setErr('')
    cancelRef.current = false
    let done = 0
    let total = 0
    try {
      for (let n = start; n <= end; n++) {
        if (skipExisting && doneNums.includes(n)) continue
        total++
      }
      for (let n = start; n <= end; n++) {
        if (cancelRef.current) break
        if (skipExisting && doneNums.includes(n)) continue
        setProgress(`正在拆解第 ${n} 章…（${done}/${total}）`)
        try {
          await app.NovelDissectChapter(n, !skipExisting)
          done++
        } catch (e) {
          // 单章失败不中断批次（如实报错，作者可修章后重跑——断点续跑跳过已拆）
          setErr(`第 ${n} 章拆解失败：${e instanceof Error ? e.message : String(e)}`)
          break
        }
      }
      setProgress(done > 0 ? `本轮拆解完成 ${done}/${total} 章` : cancelRef.current ? '已中断' : '没有需要拆解的章（全部已拆解）')
      await loadList()
    } finally {
      setRunning(false)
    }
  }

  const runReport = async () => {
    setErr('')
    try {
      setReport(await app.NovelDissectReport())
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e))
    }
  }

  return (
    <Modal
      title={<span style={{ color: C('color-text') }}>对标拆解（写法学习）</span>}
      open={open}
      onCancel={() => { cancelRef.current = true; onClose() }}
      footer={null}
      width={760}
      destroyOnHidden
      transitionName=""
      maskTransitionName=""
      styles={{ body: { background: 'transparent' }, header: { background: 'transparent' } }}
    >
      <Space direction="vertical" size={12} style={{ width: '100%' }}>
        <Typography.Text type="secondary" style={{ fontSize: 12, display: 'block' }}>
          拆「这本书的写法哪里值得学」——付息/钩子/情绪曲线/写法公式；偷机制不偷设定。拆解产物落项目 dissections/ 目录，不写书稿。
        </Typography.Text>

        <Space wrap size={8}>
          <Typography.Text style={{ color: C('color-text'), fontSize: 13 }}>章号</Typography.Text>
          <InputNumber min={1} value={from} onChange={(v) => setFrom(v)} style={{ width: 80 }} data-testid="dissect-from" />
          <Typography.Text type="secondary">至</Typography.Text>
          <InputNumber min={1} value={to} onChange={(v) => setTo(v)} style={{ width: 80 }} data-testid="dissect-to" />
          <Checkbox checked={skipExisting} onChange={(e) => setSkipExisting(e.target.checked)}>
            跳过已拆解（断点续跑）
          </Checkbox>
          {running ? (
            <Button danger size="small" onClick={() => { cancelRef.current = true }}>停止</Button>
          ) : (
            <Button size="small" type="primary" data-testid="dissect-run" onClick={() => void runDissect()}>开始拆解</Button>
          )}
          <Button size="small" data-testid="dissect-report" onClick={() => void runReport()}>生成拆解报告</Button>
        </Space>

        {progress && (
          <Typography.Text data-testid="dissect-progress" style={{ fontSize: 12, color: C('color-text-secondary') }}>
            {progress}
          </Typography.Text>
        )}
        {err && <Alert type="warning" showIcon data-testid="dissect-error" message={err} />}
        {doneNums.length > 0 && (
          <div>
            <Typography.Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 4 }}>
              已拆解 {doneNums.length} 章
            </Typography.Text>
            <Space wrap size={4}>
              {doneNums.map((n) => (
                <Tag key={n}>{String(n).padStart(3, '0')}</Tag>
              ))}
            </Space>
          </div>
        )}

        {report && (
          <div data-testid="dissect-report-view" style={{ borderTop: `1px solid ${C('color-border')}`, paddingTop: 8 }}>
            <Typography.Text strong style={{ color: C('color-text'), fontSize: 13 }}>
              拆解报告（{report.chapters_dissected} 章，纯本地聚合零 LLM）
            </Typography.Text>

            {Object.keys(report.hook_dist ?? {}).length > 0 && (
              <div style={{ marginTop: 6 }}>
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>钩子类型分布：</Typography.Text>
                <Space wrap size={4} style={{ marginTop: 2 }}>
                  {Object.entries(report.hook_dist).map(([k, v]) => (
                    <Tag key={k} color={k === '危机' || k === '反转' ? 'orange' : 'blue'}>{k}×{v}</Tag>
                  ))}
                </Space>
              </div>
            )}

            {(report.timeline ?? []).length > 0 && (
              <div style={{ marginTop: 8 }}>
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>剧情线（付息链）：</Typography.Text>
                <div style={{ maxHeight: 160, overflow: 'auto', marginTop: 2 }}>
                  {report.timeline.map((t) => (
                    <div key={t.chapter} style={{ fontSize: 12, color: C('color-text'), padding: '2px 0' }}>
                      第{String(t.chapter).padStart(3, '0')}章 [{t.hook_type}] {t.payoff || '—'}
                    </div>
                  ))}
                </div>
              </div>
            )}

            {(report.formulas ?? []).length > 0 && (
              <div style={{ marginTop: 8 }}>
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>写法公式（可迁移，去重合并）：</Typography.Text>
                <div style={{ maxHeight: 160, overflow: 'auto', marginTop: 2 }}>
                  {report.formulas.map((f, i) => (
                    <div key={i} style={{ fontSize: 12, color: C('color-text'), padding: '2px 0' }}>
                      {i + 1}. {f}
                    </div>
                  ))}
                </div>
              </div>
            )}

            {(report.emotion_arc ?? []).length > 0 && (
              <div style={{ marginTop: 8 }}>
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>情绪曲线：</Typography.Text>
                <div style={{ maxHeight: 120, overflow: 'auto', marginTop: 2 }}>
                  {report.emotion_arc.map((e) => (
                    <div key={e.chapter} style={{ fontSize: 12, color: C('color-text'), padding: '2px 0' }}>
                      第{String(e.chapter).padStart(3, '0')}章 {e.emotion_arc || '—'}
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}
      </Space>
    </Modal>
  )
}

export default DissectPanel
