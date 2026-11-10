import React, { useMemo, useState } from 'react'
import { Button, Input, Tag, Typography } from 'antd'
import { ClusterOutlined, ReloadOutlined } from '@ant-design/icons'
import SettingsSection from './SettingsSection'
import { useT } from '../../gaea/lib/i18n'
import raw from '../../data/codeAtlas.json'

/** CodeAtlasPanel — 设置中心「代码图谱」标签页（2026-11-10 建图批）：
 *  Go 包十层结构与内部依赖的快照可视化。数据=frontend/src/data/codeAtlas.json，
 *  由 `go run scripts/code_atlas_doc.go -json` 单源再生成（勿手改）；布局为组件内
 *  确定性 mini force（层锚点+斥力+弹簧，挂载时算 300 步出静态坐标，零新依赖）。 */

interface RawNode { id: string; layer: string; srcLoc: number; testLoc: number }
interface RawEdge { from: string; to: string; layer: string }
interface RawAtlas { meta: { generated: string; packages: number; edges: number; srcLoc: number; testLoc: number }; nodes: RawNode[]; edges: RawEdge[] }

const atlas = raw as unknown as RawAtlas

interface SimNode extends RawNode { in: number; out: number; x: number; y: number }

/** 层定义（与 docs/gaea-code-atlas-2026-11.md / scripts/code_atlas_doc.go 同口径；
 *  色板用主题 token 组合，明暗两态自适应）。 */
const LAYERS: { id: string; name: string; color: string }[] = [
  { id: 'entry', name: '入口', color: 'var(--gaea-glow)' },
  { id: 'shell', name: '壳层·板块装配', color: 'var(--md-sys-color-primary)' },
  { id: 'kernel-rt', name: '内核·装配执行', color: 'var(--md-sys-color-info)' },
  { id: 'kernel-lib', name: '内核·共享库', color: 'color-mix(in srgb, var(--md-sys-color-info) 45%, var(--md-sys-color-primary))' },
  { id: 'office', name: '办公域', color: 'var(--md-sys-color-success)' },
  { id: 'novel', name: '小说域', color: 'color-mix(in srgb, var(--md-sys-color-success) 45%, var(--gaea-glow))' },
  { id: 'engine', name: '引擎域', color: 'var(--md-sys-color-warning)' },
  { id: 'pantheon', name: '众神殿', color: 'color-mix(in srgb, var(--md-sys-color-warning) 45%, var(--md-sys-color-error))' },
  { id: 'domain', name: '领域库', color: 'color-mix(in srgb, var(--md-sys-color-primary) 55%, var(--md-sys-color-text-secondary))' },
  { id: 'devtools', name: 'scripts 工具带', color: 'var(--md-sys-color-text-tertiary)' },
]
const LAYER_BY_ID: Record<string, { name: string; color: string }> = {}
LAYERS.forEach((l) => { LAYER_BY_ID[l.id] = l })

const VW = 900
const VH = 560

/** 确定性伪随机（挂载即稳定，测试可复算）。 */
const rnd = (i: number): number => {
  const x = Math.sin(i * 127.1 + 311.7) * 43758.5453
  return x - Math.floor(x)
}

const radiusOf = (loc: number): number => Math.min(24, 3 + 4 * Math.sqrt(Math.max(0, loc) / 1000))

/** mini force 布局：层锚点成环（色簇可读）+ 全对斥力 + 依赖弹簧 + 中心/锚点重力。 */
function prepare(): { nodes: SimNode[]; edges: RawEdge[] } {
  const nodes: SimNode[] = atlas.nodes.map((n) => ({ ...n, in: 0, out: 0, x: 0, y: 0 }))
  const byId: Record<string, SimNode> = {}
  const idx: Record<string, number> = {}
  nodes.forEach((n, i) => { byId[n.id] = n; idx[n.id] = i })
  const edges = atlas.edges.filter((e) => byId[e.from] && byId[e.to])
  edges.forEach((e) => { byId[e.from].out++; byId[e.to].in++ })

  const anchors: Record<string, { x: number; y: number }> = {}
  LAYERS.forEach((l, i) => {
    const a = (i / LAYERS.length) * Math.PI * 2 - Math.PI / 2
    anchors[l.id] = { x: VW / 2 + Math.cos(a) * 205, y: VH / 2 + Math.sin(a) * 190 }
  })
  nodes.forEach((n, i) => {
    const an = anchors[n.layer] || { x: VW / 2, y: VH / 2 }
    n.x = an.x + (rnd(i) - 0.5) * 90
    n.y = an.y + (rnd(i + 5000) - 0.5) * 90
  })

  const vx = new Float64Array(nodes.length)
  const vy = new Float64Array(nodes.length)
  for (let step = 0; step < 300; step++) {
    for (let i = 0; i < nodes.length; i++) {
      const a = nodes[i]
      for (let j = i + 1; j < nodes.length; j++) {
        const b = nodes[j]
        let dx = a.x - b.x, dy = a.y - b.y
        let d2 = dx * dx + dy * dy
        if (d2 < 1) { d2 = 1; dx = rnd(i * 31 + j) - 0.5; dy = rnd(j * 17 + i) - 0.5 }
        const d = Math.sqrt(d2)
        if (d > 170) continue
        const f = 1500 / d2
        const fx = (dx / d) * f, fy = (dy / d) * f
        vx[i] += fx; vy[i] += fy
        vx[j] -= fx; vy[j] -= fy
      }
    }
    edges.forEach((e) => {
      const a = byId[e.from], b = byId[e.to]
      const dx = b.x - a.x, dy = b.y - a.y
      const d = Math.sqrt(dx * dx + dy * dy) || 1
      const f = (d - 44) * 0.03
      const fx = (dx / d) * f, fy = (dy / d) * f
      const ia = idx[e.from], ib = idx[e.to]
      vx[ia] += fx; vy[ia] += fy
      vx[ib] -= fx; vy[ib] -= fy
    })
    nodes.forEach((n, i) => {
      const an = anchors[n.layer] || { x: VW / 2, y: VH / 2 }
      vx[i] += (an.x - n.x) * 0.012 + (VW / 2 - n.x) * 0.002
      vy[i] += (an.y - n.y) * 0.012 + (VH / 2 - n.y) * 0.002
      vx[i] *= 0.55; vy[i] *= 0.55
      const s = Math.hypot(vx[i], vy[i])
      const cap = 9
      if (s > cap) { vx[i] = (vx[i] / s) * cap; vy[i] = (vy[i] / s) * cap }
      n.x = Math.min(VW - 18, Math.max(18, n.x + vx[i]))
      n.y = Math.min(VH - 18, Math.max(18, n.y + vy[i]))
    })
  }
  return { nodes, edges }
}

const CodeAtlasPanel: React.FC = () => {
  const t = useT()
  const graph = useMemo(prepare, [])
  const [query, setQuery] = useState('')
  const [selected, setSelected] = useState<string | null>(null)
  const [hidden, setHidden] = useState<Record<string, boolean>>({})

  const neighbor: Record<string, boolean> = useMemo(() => {
    const set: Record<string, boolean> = {}
    if (!selected) return set
    set[selected] = true
    graph.edges.forEach((e) => {
      if (e.from === selected) set[e.to] = true
      if (e.to === selected) set[e.from] = true
    })
    return set
  }, [selected, graph])

  const q = query.trim()

  const nodeOpacity = (n: SimNode): number => {
    if (selected) return neighbor[n.id] ? 0.95 : 0.07
    if (q) return n.id.includes(q) ? 0.95 : 0.07
    return 0.8
  }
  const edgeOpacity = (e: RawEdge): number => {
    if (selected) return neighbor[e.from] && neighbor[e.to] ? 0.55 : 0.04
    if (q) return (e.from.includes(q) || e.to.includes(q)) ? 0.5 : 0.04
    return 0.3
  }

  const sel = selected ? graph.nodes.find((n) => n.id === selected) : null
  const hits = q ? graph.nodes.filter((n) => n.id.includes(q) && !hidden[n.layer]).length : 0
  const bigLabels = graph.nodes.filter((n) => n.srcLoc > 8000)

  return (
    <SettingsSection
      icon={<ClusterOutlined />}
      title={t('settings.codeatlas.title')}
      desc={t('settings.codeatlas.desc')}
    >
      {/* 快照口径 + 再生成方式（如实展示数据来源，防「以为是实时」） */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap', marginBottom: 10 }}>
        <Tag style={{ margin: 0 }}>{atlas.meta.packages} {t('settings.codeatlas.packages')}</Tag>
        <Tag style={{ margin: 0 }}>{atlas.meta.edges} {t('settings.codeatlas.edges')}</Tag>
        <Tag style={{ margin: 0 }}>{t('settings.codeatlas.snapshot')} {atlas.meta.generated}</Tag>
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          {t('settings.codeatlas.regen')}
        </Typography.Text>
      </div>

      {/* 层过滤 + 搜索 + 重置（一个状态一个机制） */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap', marginBottom: 10 }}>
        {LAYERS.map((l) => (
          <Tag.CheckableTag
            key={l.id}
            checked={!hidden[l.id]}
            onChange={(v) => setHidden((h) => ({ ...h, [l.id]: !v }))}
          >
            <span style={{ display: 'inline-block', width: 8, height: 8, borderRadius: 2, marginRight: 5, background: l.color }} aria-hidden="true" />
            {l.name}
          </Tag.CheckableTag>
        ))}
        <span style={{ flex: 1 }} />
        <Input
          allowClear
          size="small"
          placeholder={t('settings.codeatlas.searchPlaceholder')}
          value={query}
          onChange={(e) => { setQuery(e.target.value); if (e.target.value) setSelected(null) }}
          style={{ width: 220 }}
          aria-label={t('settings.codeatlas.searchPlaceholder')}
        />
        <Button size="small" icon={<ReloadOutlined />} onClick={() => { setHidden({}); setQuery(''); setSelected(null) }}>
          {t('settings.codeatlas.reset')}
        </Button>
      </div>

      {/* 图主体：静态快照布局，点选高亮一跳邻居 */}
      <div
        style={{
          borderRadius: 'var(--md-sys-radius-md)',
          border: '1px solid var(--md-sys-color-outline-variant)',
          background: 'var(--md-sys-color-surface-container)',
          padding: 6,
        }}
      >
        <svg
          data-atlas-svg="1"
          viewBox={`0 0 ${VW} ${VH}`}
          role="img"
          aria-label={t('settings.codeatlas.title')}
          style={{ width: '100%', height: 'auto', display: 'block' }}
          onClick={(e) => { if (e.target === e.currentTarget) setSelected(null) }}
        >
          {graph.edges.map((e, i) => {
            const a = graph.nodes.find((n) => n.id === e.from)!
            const b = graph.nodes.find((n) => n.id === e.to)!
            const off = hidden[a.layer] || hidden[b.layer]
            return (
              <line
                key={i}
                x1={a.x} y1={a.y} x2={b.x} y2={b.y}
                stroke="var(--md-sys-color-outline-variant)"
                strokeWidth={edgeOpacity(e) > 0.4 ? 1.4 : 1}
                strokeOpacity={off ? 0 : edgeOpacity(e)}
                display={off ? 'none' : undefined}
              />
            )
          })}
          {graph.nodes.map((n) => (
            <circle
              key={n.id}
              data-node-id={n.id}
              cx={n.x} cy={n.y} r={radiusOf(n.srcLoc)}
              fill={LAYER_BY_ID[n.layer]?.color || 'var(--md-sys-color-primary)'}
              fillOpacity={nodeOpacity(n)}
              stroke={selected === n.id ? 'var(--md-sys-color-text)' : 'transparent'}
              strokeWidth={selected === n.id ? 1.5 : 0}
              display={hidden[n.layer] ? 'none' : undefined}
              style={{ cursor: 'pointer' }}
              onClick={(e) => { e.stopPropagation(); setSelected(n.id) }}
            >
              <title>{`${n.id} · ${LAYER_BY_ID[n.layer]?.name || n.layer} · 源码 ${n.srcLoc.toLocaleString()} 行 · 测试 ${n.testLoc.toLocaleString()} 行 · 被引 ${n.in} / 出边 ${n.out}`}</title>
            </circle>
          ))}
          {bigLabels.map((n) => (
            <text
              key={`l-${n.id}`}
              x={n.x} y={n.y - radiusOf(n.srcLoc) - 4}
              textAnchor="middle"
              fontSize={11}
              fill="var(--md-sys-color-text-secondary)"
              pointerEvents="none"
              display={hidden[n.layer] ? 'none' : undefined}
            >
              {n.id.replace('internal/gaea/', 'gaea/').replace('internal/', '')}
            </text>
          ))}
        </svg>
      </div>

      {/* 选中/搜索态单行说明 */}
      <Typography.Text
        type="secondary"
        style={{ display: 'block', marginTop: 8, minHeight: '1.4em', fontSize: 12 }}
        aria-live="polite"
      >
        {sel
          ? `${sel.id} · ${LAYER_BY_ID[sel.layer]?.name || sel.layer} · 源码 ${sel.srcLoc.toLocaleString()} 行 · 测试 ${sel.testLoc.toLocaleString()} 行 · 被引 ${sel.in} / 出边 ${sel.out}`
          : q
            ? `「${q}」：${hits} ${t('settings.codeatlas.hits')}`
            : t('settings.codeatlas.hint')}
      </Typography.Text>
    </SettingsSection>
  )
}

export default CodeAtlasPanel
