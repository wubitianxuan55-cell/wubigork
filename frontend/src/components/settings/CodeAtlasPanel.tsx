import React, { useMemo, useState } from 'react'
import { Button, Input, Tag, Typography } from 'antd'
import { ClusterOutlined, ReloadOutlined } from '@ant-design/icons'
import SettingsSection from './SettingsSection'
import { useT } from '../../gaea/lib/i18n'
import raw from '../../data/codeAtlas.json'

/** CodeAtlasPanel — 设置中心「代码图谱」标签页（2026-11-10 建图批；同晚神经网络形态重设计）。
 *  数据=frontend/src/data/codeAtlas.json，由 `go run scripts/code_atlas_doc.go -json`
 *  单源再生成（勿手改）。
 *
 *  神经网络形态不是装饰：Go 的包依赖是无环 DAG，按「到基础库的最长路径」做拓扑
 *  分层（height=0 无内部依赖=输入层；(main)=唯一输出神经元），每条依赖边都严格
 *  从左往右流动——信号沿「基础库 → 入口」的供给方向传导，长跨度边即跨层连接。
 *  颜色仍标建筑学层（十层口径），列标 L0..Ln 为拓扑深度。动效三样：环境信号
 *  （固定子集匀速流动）、选中点火（扩散环）、神经元辉光微呼吸，全部在
 *  prefers-reduced-motion 下停用。零新依赖，静态坐标一次算定。 */

interface RawNode { id: string; layer: string; srcLoc: number; testLoc: number }
interface RawEdge { from: string; to: string; layer: string }
interface RawAtlas { meta: { generated: string; packages: number; edges: number; srcLoc: number; testLoc: number }; nodes: RawNode[]; edges: RawEdge[] }

const atlas = raw as unknown as RawAtlas

interface SimNode extends RawNode { in: number; out: number; x: number; y: number; ambient: boolean }
interface SynEdge { from: string; to: string; d: string; ambient: boolean }

const LAYERS: { id: string; name: string; color: string }[] = [
  { id: 'entry', name: '入口', color: 'var(--gaea-glow)' },
  { id: 'shell', name: '壳层·板块装配', color: 'var(--md-sys-color-primary)' },
  { id: 'kernel-rt', name: '内核·装配执行', color: 'var(--md-sys-color-info)' },
  { id: 'kernel-lib', name: '内核·共享库', color: 'color-mix(in srgb, var(--md-sys-color-info) 70%, var(--md-sys-color-primary))' },
  { id: 'office', name: '办公域', color: 'var(--md-sys-color-success)' },
  { id: 'novel', name: '小说域', color: 'color-mix(in srgb, var(--md-sys-color-success) 70%, var(--gaea-glow))' },
  { id: 'engine', name: '引擎域', color: 'var(--md-sys-color-warning)' },
  { id: 'pantheon', name: '众神殿', color: 'color-mix(in srgb, var(--md-sys-color-warning) 65%, var(--md-sys-color-error))' },
  { id: 'domain', name: '领域库', color: 'color-mix(in srgb, var(--md-sys-color-primary) 75%, var(--md-sys-color-text-secondary))' },
  { id: 'devtools', name: 'scripts 工具带', color: 'var(--md-sys-color-text-tertiary)' },
]
const LAYER_BY_ID: Record<string, { name: string; color: string; index: number }> = {}
LAYERS.forEach((l, i) => { LAYER_BY_ID[l.id] = { ...l, index: i } })

const VW = 960
const VH = 560
const PAD_T = 16
const PAD_B = 38
const MAX_PER_SUB = 26

const fmt = (n: number): string => n.toLocaleString()

/** 亮态减分项打磨（视觉复验 2026-11-10）：层色直接作线色在浅底上发白——
 *  混入 15% 文字色后暗态微提亮、亮态自动压暗，两态细线都保住对比。 */
const inkColor = (layerColor: string): string =>
  `color-mix(in srgb, ${layerColor} 85%, var(--md-sys-color-text))`

/** 前馈分层布局：height=最长路径到基础库；列=height，宽列拆双子列锯齿排布；
 *  列内按建筑学层分带+重心（下层邻居均值）排序压 crossings。确定性可复算。 */
function prepare(): { nodes: SimNode[]; edges: SynEdge[]; depthCount: number } {
  const byIdR: Record<string, RawNode> = {}
  atlas.nodes.forEach((n) => { byIdR[n.id] = n })
  const outs: Record<string, string[]> = {}
  const ins: Record<string, string[]> = {}
  atlas.nodes.forEach((n) => { outs[n.id] = []; ins[n.id] = [] })
  const edgesRaw = atlas.edges.filter((e) => byIdR[e.from] && byIdR[e.to])
  edgesRaw.forEach((e) => { outs[e.from].push(e.to); ins[e.to].push(e.from) })

  const heightMemo: Record<string, number> = {}
  const height = (id: string, seen: Set<string>): number => {
    if (heightMemo[id] !== undefined) return heightMemo[id]
    if (seen.has(id)) return 0 // 防御：Go 编译器保证无环，此行不可达
    seen.add(id)
    let m = -1
    for (const t of outs[id]) m = Math.max(m, height(t, seen))
    seen.delete(id)
    return (heightMemo[id] = m + 1)
  }
  atlas.nodes.forEach((n) => height(n.id, new Set()))

  const depthCount = Math.max(...Object.values(heightMemo)) + 1
  const colX = (h: number): number => 30 + (h * (VW - 60)) / (depthCount - 1)
  const usableY = VH - PAD_T - PAD_B

  const nodes: SimNode[] = atlas.nodes.map((n, i) => ({
    ...n,
    in: ins[n.id].length, out: outs[n.id].length,
    x: 0, y: 0,
    ambient: i % 11 === 0, // 固定子集微呼吸（确定性）
  }))
  const pos: Record<string, { x: number; y: number }> = {}
  const simById: Record<string, SimNode> = {}
  nodes.forEach((n) => { simById[n.id] = n })

  for (let h = 0; h < depthCount; h++) {
    const col = nodes.filter((n) => heightMemo[n.id] === h)
    col.sort((a, b) => {
      const la = LAYER_BY_ID[a.layer]?.index ?? 99
      const lb = LAYER_BY_ID[b.layer]?.index ?? 99
      if (la !== lb) return la - lb
      const bary = (n: SimNode): number => {
        const ys = outs[n.id].map((t) => pos[t]?.y).filter((v) => v !== undefined) as number[]
        return ys.length ? ys.reduce((s, v) => s + v, 0) / ys.length : -1
      }
      const ba = bary(a), bb = bary(b)
      if (ba !== bb && ba >= 0 && bb >= 0) return ba - bb
      return a.id < b.id ? -1 : 1
    })
    const subs = Math.ceil(col.length / MAX_PER_SUB)
    col.forEach((n, i) => {
      const sub = Math.min(Math.floor(i / MAX_PER_SUB), subs - 1)
      const inSub = i - sub * MAX_PER_SUB
      const subCount = sub < subs - 1 ? MAX_PER_SUB : col.length - (subs - 1) * MAX_PER_SUB
      const pitch = usableY / (subCount + 1)
      n.x = colX(h) + (sub - (subs - 1) / 2) * 34
      n.y = PAD_T + pitch * (inSub + 1)
      pos[n.id] = { x: n.x, y: n.y }
    })
  }

  const edges: SynEdge[] = edgesRaw.map((e, i) => {
    const a = pos[e.to], b = pos[e.from] // 信号方向：被依赖（左）→ 依赖方（右）
    const dx = (b.x - a.x) * 0.42
    return {
      from: e.from, to: e.to, ambient: i % 6 === 0,
      d: `M ${a.x.toFixed(1)} ${a.y.toFixed(1)} C ${(a.x + dx).toFixed(1)} ${a.y.toFixed(1)}, ${(b.x - dx).toFixed(1)} ${b.y.toFixed(1)}, ${b.x.toFixed(1)} ${b.y.toFixed(1)}`,
    }
  })
  return { nodes, edges, depthCount }
}

const radiusOf = (loc: number): number => Math.min(14, Math.max(3.2, 2.8 + 2.2 * Math.sqrt(loc / 1000)))

const CodeAtlasPanel: React.FC = () => {
  const t = useT()
  const graph = useMemo(prepare, [])
  const byId = useMemo(() => {
    const m: Record<string, SimNode> = {}
    graph.nodes.forEach((n) => { m[n.id] = n })
    return m
  }, [graph])
  const layerCounts = useMemo(() => {
    const c: Record<string, number> = {}
    atlas.nodes.forEach((n) => { c[n.layer] = (c[n.layer] || 0) + 1 })
    return c
  }, [])
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
    return 0.95
  }
  const edgeOpacity = (e: SynEdge): number => {
    if (selected) return neighbor[e.from] && neighbor[e.to] ? 0.75 : 0.04
    if (q) return (e.from.includes(q) || e.to.includes(q)) ? 0.55 : 0.04
    return 0.34
  }
  const isFocusEdge = (e: SynEdge): boolean =>
    !!selected && neighbor[e.from] && neighbor[e.to]

  const sel = selected ? graph.nodes.find((n) => n.id === selected) : null
  const hits = q ? graph.nodes.filter((n) => n.id.includes(q) && !hidden[n.layer]).length : 0
  const bigLabels = useMemo(() => graph.nodes.filter((n) => n.srcLoc > 8000).map((n) => n.id), [graph])
  const focusLabels = useMemo(() => {
    if (!selected) return [] as string[]
    return graph.nodes
      .filter((n) => neighbor[n.id] && n.srcLoc > 1200)
      .sort((a, b) => b.srcLoc - a.srcLoc)
      .slice(0, 12)
      .map((n) => n.id)
  }, [selected, neighbor, graph])
  const labelIds = new Set([...bigLabels, ...focusLabels])
  const allHidden = LAYERS.every((l) => hidden[l.id])

  const stats: { value: string; label: string }[] = [
    { value: fmt(atlas.meta.packages), label: t('settings.codeatlas.packages') },
    { value: fmt(atlas.meta.edges), label: t('settings.codeatlas.edges') },
    { value: fmt(atlas.meta.srcLoc), label: t('settings.codeatlas.statSrc') },
    { value: fmt(atlas.meta.testLoc), label: t('settings.codeatlas.statTest') },
  ]

  return (
    <SettingsSection
      icon={<ClusterOutlined />}
      title={t('settings.codeatlas.title')}
      desc={t('settings.codeatlas.desc')}
    >
      <style>{`
        .codeatlas-node:hover { stroke: var(--md-sys-color-text); stroke-width: 1.5; }
        .atlas-signal { stroke-dasharray: 2 15; animation: atlas-flow 2.4s linear infinite; }
        @keyframes atlas-flow { to { stroke-dashoffset: -119; } }
        .atlas-fire { transform-box: fill-box; transform-origin: center; animation: atlas-fire-kf 1.15s ease-out infinite; }
        @keyframes atlas-fire-kf { 0% { transform: scale(1); opacity: 0.75; } 100% { transform: scale(2.4); opacity: 0; } }
        .atlas-ambient { animation: atlas-ambient-kf 3.2s ease-in-out infinite; }
        @keyframes atlas-ambient-kf { 0%, 100% { opacity: 0.1; } 50% { opacity: 0.32; } }
        @media (prefers-reduced-motion: reduce) {
          .atlas-signal, .atlas-fire, .atlas-ambient { animation: none; }
          .atlas-signal { display: none; }
        }
      `}</style>

      {/* 快照口径 stat 条 */}
      <div style={{ display: 'flex', alignItems: 'stretch', gap: 8, flexWrap: 'wrap', marginBottom: 12 }}>
        {stats.map((s) => (
          <div
            key={s.label}
            style={{
              flex: '1 1 100px', minWidth: 100, padding: '8px 12px',
              borderRadius: 'var(--md-sys-radius-md)',
              background: 'var(--md-sys-color-surface-container)',
              border: '1px solid var(--md-sys-color-outline-variant)',
            }}
          >
            <div className="tabular-nums" style={{ fontSize: 16, fontWeight: 600, color: 'var(--md-sys-color-text)', lineHeight: 1.2 }}>{s.value}</div>
            <div style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)', marginTop: 2 }}>{s.label}</div>
          </div>
        ))}
        <div style={{ display: 'flex', alignItems: 'center' }}>
          <Tag style={{ margin: 0, borderRadius: 8 }}>{t('settings.codeatlas.snapshot')} {atlas.meta.generated}</Tag>
        </div>
      </div>

      {/* 层 chips=图例+过滤（色点+名称+计数）｜右侧搜索+重置 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 6, flexWrap: 'wrap', marginBottom: 10 }}>
        {LAYERS.map((l) => (
          <Tag.CheckableTag
            key={l.id}
            checked={!hidden[l.id]}
            onChange={(v) => setHidden((h) => ({ ...h, [l.id]: !v }))}
          >
            <span style={{ display: 'inline-block', width: 8, height: 8, borderRadius: 2, marginRight: 5, background: l.color }} aria-hidden="true" />
            {l.name}
            <span style={{ opacity: 0.65, marginLeft: 4 }} className="tabular-nums">{layerCounts[l.id] || 0}</span>
          </Tag.CheckableTag>
        ))}
        <span style={{ flex: 1, minWidth: 8 }} />
        <Input
          allowClear
          size="small"
          placeholder={t('settings.codeatlas.searchPlaceholder')}
          value={query}
          onChange={(e) => { setQuery(e.target.value); if (e.target.value) setSelected(null) }}
          style={{ width: 210 }}
          aria-label={t('settings.codeatlas.searchPlaceholder')}
        />
        <Button size="small" icon={<ReloadOutlined />} onClick={() => { setHidden({}); setQuery(''); setSelected(null) }}>
          {t('settings.codeatlas.reset')}
        </Button>
      </div>

      {/* 神经网络主体：输入层=基础库（左）… 输出神经元=(main)（右）；信号沿依赖供给方向流动 */}
      <div
        style={{
          position: 'relative',
          borderRadius: 'var(--md-sys-radius-lg)',
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
          {/* 列参考线 + 深度标号 */}
          {Array.from({ length: graph.depthCount }, (_, h) => {
            const x = 30 + (h * (VW - 60)) / (graph.depthCount - 1)
            return (
              <g key={`col-${h}`} pointerEvents="none">
                <line x1={x} y1={PAD_T} x2={x} y2={VH - PAD_B} stroke="var(--md-sys-color-outline-variant)" strokeOpacity={0.25} strokeDasharray="2 6" />
                <text x={x} y={VH - 12} textAnchor="middle" fontSize={10} fill="var(--md-sys-color-text-secondary)">L{h}</text>
              </g>
            )
          })}
          <text x={8} y={VH - 12} fontSize={10} fill="var(--md-sys-color-text-secondary)" textAnchor="start">
            {t('settings.codeatlas.axisInput')}
          </text>
          <text x={VW - 8} y={VH - 12} fontSize={10} fill="var(--md-sys-color-text-secondary)" textAnchor="end">
            {t('settings.codeatlas.axisOutput')}
          </text>

          {/* 突触（贝塞尔）：信号方向=被依赖→依赖方（左→右） */}
          {graph.edges.map((e, i) => {
            const a = byId[e.to], b = byId[e.from]
            const off = !a || !b || hidden[a.layer] || hidden[b.layer]
            if (off) return <path key={i} d={e.d} fill="none" display="none" />
            const focus = isFocusEdge(e)
            return (
              <path
                key={i}
                data-edge={i}
                data-from={e.from}
                data-to={e.to}
                d={e.d}
                fill="none"
                stroke={focus ? 'var(--gaea-glow)' : inkColor(LAYER_BY_ID[a.layer]?.color || 'var(--md-sys-color-primary)')}
                strokeWidth={focus ? 1.6 : 1}
                strokeOpacity={edgeOpacity(e)}
              />
            )
          })}
          {/* 流动信号：环境固定子集 + 聚焦边（reduced-motion 下隐藏） */}
          {graph.edges.map((e, i) => {
            const a = byId[e.to], b = byId[e.from]
            const off = !a || !b || hidden[a.layer] || hidden[b.layer]
            const active = !off && (e.ambient || isFocusEdge(e)) && (!q || e.from.includes(q) || e.to.includes(q))
            if (!active) return null
            return (
              <path
                key={`sig-${i}`}
                className="atlas-signal"
                d={e.d}
                fill="none"
                stroke={isFocusEdge(e) ? 'var(--gaea-glow)' : inkColor(LAYER_BY_ID[a!.layer]?.color || 'var(--gaea-glow)')}
                strokeWidth={isFocusEdge(e) ? 2 : 1.2}
                strokeOpacity={isFocusEdge(e) ? 0.9 : 0.55}
                style={{ animationDelay: `${-((i % 7) * 0.34)}s`, animationDuration: `${2 + (i % 5) * 0.28}s` }}
                pointerEvents="none"
              />
            )
          })}

          {/* 神经元：辉光（微呼吸子集）→ 胞体 → 核 */}
          {graph.nodes.map((n) => {
            const r = radiusOf(n.srcLoc)
            const color = LAYER_BY_ID[n.layer]?.color || 'var(--md-sys-color-primary)'
            const off = hidden[n.layer]
            const dim = nodeOpacity(n) < 0.5
            return (
              <g key={n.id} display={off ? 'none' : undefined}>
                <circle
                  className={n.ambient && !dim ? 'atlas-ambient' : undefined}
                  cx={n.x} cy={n.y} r={r * 2.05}
                  fill={color}
                  fillOpacity={dim ? 0.03 : 0.2}
                  pointerEvents="none"
                />
                <circle
                  data-node-id={n.id}
                  className="codeatlas-node"
                  cx={n.x} cy={n.y} r={r}
                  fill={color}
                  fillOpacity={nodeOpacity(n)}
                  stroke="var(--md-sys-color-surface-container)"
                  strokeWidth={1}
                  display={off ? 'none' : undefined}
                  style={{ cursor: 'pointer' }}
                  onClick={(ev) => { ev.stopPropagation(); setSelected(n.id) }}
                >
                  <title>{`${n.id} · ${LAYER_BY_ID[n.layer]?.name || n.layer} · 源码 ${fmt(n.srcLoc)} 行 · 测试 ${fmt(n.testLoc)} 行 · 被引 ${n.in} / 出边 ${n.out}`}</title>
                </circle>
                <circle cx={n.x} cy={n.y} r={Math.max(1, r * 0.26)} fill="var(--md-sys-color-text)" fillOpacity={dim ? 0.05 : 0.5} pointerEvents="none" />
                {selected === n.id && (
                  <circle
                    data-node-ring={n.id}
                    className="atlas-fire"
                    cx={n.x} cy={n.y} r={r + 3}
                    fill="none"
                    stroke="var(--md-sys-color-text)"
                    strokeWidth={1.4}
                    pointerEvents="none"
                  />
                )}
              </g>
            )
          })}

          {/* 标签：巨型包常驻 + 聚焦邻居补显（表面色光晕保压线可读） */}
          {graph.nodes.filter((n) => labelIds.has(n.id)).map((n) => (
            <text
              key={`l-${n.id}`}
              x={n.x} y={n.y - radiusOf(n.srcLoc) - 5}
              textAnchor="middle"
              fontSize={11}
              fill="var(--md-sys-color-text-secondary)"
              stroke="var(--md-sys-color-surface-container)"
              strokeWidth={2.5}
              paintOrder="stroke"
              pointerEvents="none"
              display={hidden[n.layer] ? 'none' : undefined}
            >
              {n.id.replace('internal/gaea/', 'gaea/').replace('internal/', '')}
            </text>
          ))}
          {allHidden && (
            <text x={VW / 2} y={VH / 2} textAnchor="middle" fontSize={13} fill="var(--md-sys-color-text-secondary)">
              {t('settings.codeatlas.emptyLayers')}
            </text>
          )}
        </svg>
      </div>

      {/* 检视条：默认提示｜搜索命中｜选中神经元四项底数（aria-live 播报） */}
      <div
        aria-live="polite"
        style={{
          display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap',
          marginTop: 8, minHeight: 26, fontSize: 12,
          color: 'var(--md-sys-color-text-secondary)',
        }}
      >
        {sel ? (
          <>
            <span
              style={{
                display: 'inline-flex', alignItems: 'center', gap: 5,
                padding: '2px 8px', borderRadius: 999,
                background: 'color-mix(in srgb, var(--gaea-glow) 10%, transparent)',
                border: '1px solid color-mix(in srgb, var(--gaea-glow) 30%, transparent)',
                color: 'var(--md-sys-color-text)', fontSize: 11,
              }}
            >
              <span style={{ width: 7, height: 7, borderRadius: 2, background: LAYER_BY_ID[sel.layer]?.color }} aria-hidden="true" />
              {LAYER_BY_ID[sel.layer]?.name || sel.layer}
            </span>
            <code style={{ color: 'var(--md-sys-color-text)' }}>{sel.id}</code>
            <span className="tabular-nums">{t('settings.codeatlas.statSrc')} {fmt(sel.srcLoc)}</span>
            <span className="tabular-nums">{t('settings.codeatlas.statTest')} {fmt(sel.testLoc)}</span>
            <span className="tabular-nums">{t('settings.codeatlas.statIn')} {sel.in}</span>
            <span className="tabular-nums">{t('settings.codeatlas.statOut')} {sel.out}</span>
          </>
        ) : q ? (
          <span>「{q}」：{hits} {t('settings.codeatlas.hits')}</span>
        ) : (
          <span>{t('settings.codeatlas.hint')}</span>
        )}
      </div>

      {/* 快照再生成口径（如实展示数据来源，防「以为是实时」） */}
      <Typography.Text type="secondary" style={{ display: 'block', marginTop: 6, fontSize: 11 }}>
        <code>{t('settings.codeatlas.regen')}</code>
      </Typography.Text>
    </SettingsSection>
  )
}

export default CodeAtlasPanel
