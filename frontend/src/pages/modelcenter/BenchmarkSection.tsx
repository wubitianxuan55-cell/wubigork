import React, { useCallback, useEffect, useMemo, useState } from 'react'
import { Button, Input, InputNumber, message, Select, Space } from 'antd'
import { ExperimentOutlined, ExportOutlined, PlayCircleOutlined, ReloadOutlined } from '@ant-design/icons'
import { EmptyState, KpiTile, SectionHead, StatusChip } from './ui'
import {
  exportBenchmark, getBenchmarkList, getHerdsmanCatalog, startBenchmark,
  type BenchmarkPromptRequest, type BenchmarkRequest, type BenchmarkRunSummary,
} from '../../api/engines'

/**
 * BenchmarkSection — Herdsman 受控测评（D3-3 测评产品化）
 *
 * 复用 herdsman /api/benchmarks 异步测评（多模型 × 变体 × 上下文长度，
 * 逐 case 采集 TTFT/TPS/token）：发起受控测评 / 运行列表 / 导出 Markdown 报告。
 * 上下文长度与并发选项同时覆盖 D3-4 的长上下文（>4K）与并发专项。
 */

// 受控任务集（蒸馏自 120 组对照测评方法学，2026-08-12 报告）
const TASK_PRESETS: { key: string; label: string; prompt: string }[] = [
  { key: 'common', label: 'T1 常识解释', prompt: '用 50-200 字通俗解释什么是「大语言模型」，要求条理清晰、举例恰当。' },
  { key: 'doc', label: 'T2 公文写作', prompt: '以正式公文格式撰写一份「关于加强项目施工现场安全管理工作的通知」，不少于 120 字，要素齐全。' },
  { key: 'translate', label: 'T3 翻译（英→中）', prompt: '将以下英文翻译成地道中文：「The contractor shall complete all site works in accordance with the approved drawings and specifications within the stipulated timeframe.」' },
  { key: 'code', label: 'T4 代码生成', prompt: '用 Python 写一个函数 find_most_frequent(words: list[str]) -> str，返回列表中出现次数最多的元素；并列时返回先出现的。' },
  { key: 'logic', label: 'T5 逻辑推理', prompt: '有三只箱子分别贴有「苹果」「橘子」「苹果和橘子」的标签，但所有标签都是错的。你只能从一只箱子中拿一个水果，如何判断每只箱子里装的是什么？' },
  { key: 'math', label: 'T6 工程计算', prompt: '某设备每天工作 8 小时，每小时消耗柴油 5.2 升，柴油单价 7.8 元/升。计算该设备一个月（22 个工作日）的柴油费用，并写出计算过程。' },
  { key: 'summary', label: 'T7 长文摘要', prompt: '请用 50-100 字概括以下段落的核心信息：城市地下综合管廊是指在城市地下建造一个隧道空间，将电力、通信、燃气、供热、给排水等各种工程管线集于一体，设有专门的检修口、吊装口和监测系统，实施统一规划、统一设计、统一建设和管理，是保障城市运行的重要基础设施和「生命线」。' },
  { key: 'json', label: 'T8 JSON 抽取', prompt: '从以下文本中提取所有成本条目，输出 JSON 数组（字段：name/unit/price/source）：「台班费：挖掘机 3200 元/台班（来源：市场询价）；柴油 7.8 元/升（来源：当地加油站）；钢筋 4100 元/吨（来源：供应商报价单）。」只输出 JSON。' },
  { key: 'creative', label: 'T11 创意写作', prompt: '写一篇 200 字以上的微小说，必须包含「雨」和「电话亭」两个元素，结尾要有反转。' },
  { key: 'long', label: 'T14 长文输出', prompt: '请撰写一份「冬季施工方案」的技术说明，不少于 600 字，结构完整（背景、措施、安全、附录）。' },
]

const CONTEXT_OPTIONS = [
  { value: 4096, label: '4K（常规）' },
  { value: 8192, label: '8K（长上下文）' },
  { value: 16384, label: '16K（长上下文）' },
  { value: 32768, label: '32K（长上下文）' },
]

const CONCURRENCY_OPTIONS = [
  { value: 1, label: '1（串行基准）' },
  { value: 2, label: '2（并发）' },
  { value: 4, label: '4（并发压力）' },
]

const statusTone = (s: string): 'ok' | 'warn' | 'danger' | 'neutral' => {
  if (s === 'succeeded') return 'ok'
  if (s === 'running' || s === 'pending') return 'warn'
  if (s === 'failed') return 'danger'
  return 'neutral'
}

const fmtTime = (s: string) => (s ? s.replace('T', ' ').slice(0, 16) : '—')

export function BenchmarkSection() {
  const [runs, setRuns] = useState<BenchmarkRunSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [starting, setStarting] = useState(false)
  const [installed, setInstalled] = useState<{ name: string; displayName: string }[]>([])
  const [form, setForm] = useState<{
    models: string[]
    preset: string
    customPrompt: string
    temperature: number
    maxTokens: number
    contextSize: number
    concurrency: number
  }>({ models: [], preset: 'common', customPrompt: '', temperature: 0.3, maxTokens: 512, contextSize: 4096, concurrency: 1 })

  const load = useCallback(async () => {
    try {
      const [list, catalog] = await Promise.all([getBenchmarkList(), getHerdsmanCatalog().catch(() => null)])
      setRuns(list || [])
      if (catalog?.models) {
        setInstalled(
          catalog.models
            .filter((m) => m.installed)
            .map((m) => ({ name: m.name, displayName: m.display_name || m.name })),
        )
      }
    } catch (err: any) {
      message.warning(err?.message || '无法连接 Herdsman 测评接口')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { void load() }, [load])

  // 有运行中的任务时每 8s 轮询刷新。
  const anyRunning = runs.some((r) => r.status === 'running' || r.status === 'pending')
  useEffect(() => {
    if (!anyRunning) return
    const t = setInterval(() => { void load() }, 8000)
    return () => clearInterval(t)
  }, [anyRunning, load])

  const promptText = form.customPrompt.trim() || TASK_PRESETS.find((p) => p.key === form.preset)?.prompt || ''

  const handleStart = async () => {
    if (form.models.length === 0) {
      message.warning('请至少选择一个已安装模型')
      return
    }
    if (!promptText) {
      message.warning('请选择任务或填写提示词')
      return
    }
    setStarting(true)
    const req: BenchmarkRequest = {
      model_names: form.models,
      variants: ['standard'],
      context_sizes: [form.contextSize],
      cache_reuse_mode: 'same_prompt_second',
      warmup_count: 1,
      repeat_count: 1,
      concurrency: form.concurrency,
      request: {
        user_prompt: promptText,
        temperature: form.temperature,
        top_p: 0.9,
        top_k: 40,
        repeat_penalty: 1.1,
        max_tokens: form.maxTokens,
        stream: true,
        timeout_seconds: 1800,
      } as BenchmarkPromptRequest,
    }
    try {
      const id = await startBenchmark(req)
      message.success(`测评已发起（运行 ${id.slice(0, 8)}…），完成后将出现在下方列表`)
      setForm((f) => ({ ...f, models: [] }))
      void load()
    } catch (err: any) {
      message.error(err?.message || '发起测评失败')
    } finally {
      setStarting(false)
    }
  }

  const handleExport = async (id: string) => {
    try {
      const go = (window as any).go?.app?.App
      let dir = ''
      if (typeof go?.PickDirectory === 'function') dir = (await go.PickDirectory()) || ''
      if (!dir) {
        message.info('已取消导出（未选择目录）')
        return
      }
      const path = await exportBenchmark(id, dir)
      message.success(`报告已导出：${path}`)
    } catch (err: any) {
      message.error(err?.message || '导出失败')
    }
  }

  const modelOptions = useMemo(
    () => installed.map((m) => ({ value: m.name, label: m.displayName || m.name })),
    [installed],
  )

  return (
    <div className="mc-drawer-body">
      <SectionHead
        icon={<ExperimentOutlined />}
        title="Herdsman 受控测评"
        desc="复用 herdsman /api/benchmarks 异步测评：多模型 × 上下文长度 × 并发，逐 case 采集 TTFT / TPS / token；方法学蒸馏自 120 组对照测评（2026-08-12）。"
        extra={
          <Button size="small" icon={<ReloadOutlined />} loading={loading} onClick={() => void load()}>
            刷新
          </Button>
        }
      />

      {/* 发起新测评 */}
      <div className="mc-panel" style={{ marginTop: 12 }}>
        <div className="mc-panel-title">发起新测评</div>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', gap: '0 16px', marginTop: 8 }}>
          <div style={{ marginBottom: 12 }}>
            <div className="mc-field-label">被测模型（可多选）</div>
            <Select
              mode="multiple"
              size="small"
              placeholder="选择已安装模型（模型库 → 启动）"
              value={form.models}
              onChange={(v) => setForm({ ...form, models: v })}
              options={modelOptions}
              style={{ width: '100%' }}
              showSearch
              optionFilterProp="label"
            />
            {installed.length === 0 && (
              <div style={{ fontSize: 11, color: 'var(--mc-muted)', marginTop: 2 }}>
                未发现已安装模型——请先在「模型库」下载并启动模型
              </div>
            )}
          </div>
          <div style={{ marginBottom: 12 }}>
            <div className="mc-field-label">受控任务</div>
            <Select
              size="small"
              value={form.preset}
              onChange={(v) => setForm({ ...form, preset: v })}
              options={TASK_PRESETS.map((p) => ({ value: p.key, label: p.label }))}
              style={{ width: '100%' }}
            />
          </div>
          <div style={{ marginBottom: 12 }}>
            <div className="mc-field-label">提示词（自定义覆盖任务预设）</div>
            <Input.TextArea
              rows={2}
              size="small"
              value={form.customPrompt}
              onChange={(e) => setForm({ ...form, customPrompt: e.target.value })}
              placeholder={TASK_PRESETS.find((p) => p.key === form.preset)?.prompt}
              style={{ background: 'var(--md-sys-color-surface-container)', border: '1px solid var(--md-sys-color-outline-variant)' }}
            />
          </div>
          <div style={{ marginBottom: 12 }}>
            <div className="mc-field-label">参数</div>
            <Space size={8} wrap>
              <InputNumber size="small" min={0} max={1} step={0.1} value={form.temperature}
                onChange={(v) => setForm({ ...form, temperature: v ?? 0.3 })} addonBefore="温度" />
              <InputNumber size="small" min={64} max={4096} step={64} value={form.maxTokens}
                onChange={(v) => setForm({ ...form, maxTokens: v ?? 512 })} addonBefore="max_tokens" />
              <Select size="small" value={form.contextSize}
                onChange={(v) => setForm({ ...form, contextSize: v })}
                options={CONTEXT_OPTIONS} style={{ width: 150 }} />
              <Select size="small" value={form.concurrency}
                onChange={(v) => setForm({ ...form, concurrency: v })}
                options={CONCURRENCY_OPTIONS} style={{ width: 130 }} />
            </Space>
          </div>
        </div>
        <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
          <Button type="primary" size="small" icon={<PlayCircleOutlined />} loading={starting} onClick={() => void handleStart()}>
            发起受控测评
          </Button>
        </div>
      </div>

      {/* 运行列表 */}
      <div className="mc-panel" style={{ marginTop: 12 }}>
        <div className="mc-panel-title">历史运行</div>
        {loading ? (
          <EmptyState title="加载中…" compact />
        ) : runs.length === 0 ? (
          <EmptyState
            icon={<ExperimentOutlined />}
            title="暂无测评记录"
            hint="发起一次受控测评后，结果将在这里汇总（TTFT / TPS / token）"
            compact
          />
        ) : (
          runs.map((r) => (
            <div key={r.id} className="mc-catalog-stats" style={{ marginBottom: 10 }}>
              <div className="mc-catalog-stats-head">
                <span className="mc-catalog-stats-model" style={{ fontWeight: 600 }}>
                  {r.model_names.join(' + ')}
                </span>
                <span style={{ flex: 1 }} />
                <span style={{ color: 'var(--mc-muted)', fontSize: 11, marginRight: 8 }}>
                  {fmtTime(r.created_at)} · ctx {r.context_sizes.join('/')}
                </span>
                <StatusChip tone={statusTone(r.status)} dot>{r.status}</StatusChip>
                <Button size="small" type="text" icon={<ExportOutlined />}
                  title="导出 Markdown 报告" onClick={() => void handleExport(r.id)}>
                  导出
                </Button>
              </div>
              {r.summary && (
                <div className="mc-overview-grid" style={{ gridTemplateColumns: 'repeat(4, minmax(0,1fr))', marginTop: 8 }}>
                  <KpiTile label="用例" value={r.summary.total_cases}
                    hint={`成功 ${r.summary.succeeded} · 失败 ${r.summary.failed}`} />
                  <KpiTile label="平均 TPS" value={r.summary.avg_tps ? r.summary.avg_tps.toFixed(1) : '—'} />
                  <KpiTile label="平均 TTFT" value={r.summary.avg_ttft_ms ? `${r.summary.avg_ttft_ms.toFixed(0)} ms` : '—'} />
                  <KpiTile label="平均耗时" value={r.summary.avg_duration_ms ? `${r.summary.avg_duration_ms.toFixed(0)} ms` : '—'} />
                </div>
              )}
            </div>
          ))
        )}
      </div>

      <div style={{ color: 'var(--mc-muted)', fontSize: 11, marginTop: 8 }}>
        口径说明：温度 0.3 默认、同一提示词、cache 复用 same_prompt_second（二次请求测缓存命中）；
        报告导出为 Markdown（含逐用例 TTFT avg/p95、入/出/总 token、缓存 token）。
      </div>
    </div>
  )
}

export default BenchmarkSection
