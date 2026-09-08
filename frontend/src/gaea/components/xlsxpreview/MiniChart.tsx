/**
 * xlsxpreview/MiniChart — 自 XlsxPreview 原样搬移的 SVG 迷你图（原生图表已嵌入
 * xlsx，这里是即时视觉反馈）：bar 柱状、line 折线、pie 扇形，其余类型归一为柱状。
 * 调色板为数据非 UI chrome（// hex-exempt）。
 */

const PIE_COLORS = ["#5B8DEF", "#7BC47F", "#F2A65A", "#E57373", "#9B7EDE", "#4FC3C3", "#D4A5E5", "#8D9CAD"]; // hex-exempt 图表调色板（xlsx 原生图表）
const CHART_BLUE = "#5B8DEF"; // hex-exempt 图表调色板
const MAX_MINI_POINTS = 12;

export function MiniChart({ chartType, labels, values }: { chartType: string; labels: string[]; values: number[] }) {
  const vals = values.slice(0, MAX_MINI_POINTS);
  const lbls = labels.slice(0, MAX_MINI_POINTS);
  if (vals.length === 0) return null;
  const W = 560;
  const H = 150;
  const trunc = (s: string) => (s.length > 6 ? s.slice(0, 5) + "…" : s);

  if (chartType === "pie") {
    const cx = 75;
    const cy = H / 2;
    const r = H / 2 - 8;
    const total = vals.reduce((s, v) => s + Math.abs(v), 0) || 1;
    let acc = -Math.PI / 2;
    const segs = vals.map((v, i) => {
      const a0 = acc;
      acc += (Math.abs(v) / total) * Math.PI * 2;
      const a1 = acc;
      const large = a1 - a0 > Math.PI ? 1 : 0;
      const x0 = cx + r * Math.cos(a0);
      const y0 = cy + r * Math.sin(a0);
      const x1 = cx + r * Math.cos(a1);
      const y1 = cy + r * Math.sin(a1);
      return {
        d: `M ${cx} ${cy} L ${x0.toFixed(2)} ${y0.toFixed(2)} A ${r} ${r} 0 ${large} 1 ${x1.toFixed(2)} ${y1.toFixed(2)} Z`,
        color: PIE_COLORS[i % PIE_COLORS.length],
        label: trunc(lbls[i] ?? String(i + 1)),
        pct: Math.round((Math.abs(v) / total) * 100),
      };
    });
    return (
      <svg viewBox={`0 0 ${W} ${H}`} className="w-full max-w-[560px] mt-1.5" data-testid="mini-chart" role="img">
        {segs.map((s, i) => (
          <path key={i} d={s.d} fill={s.color} stroke="var(--bg)" strokeWidth="1" />
        ))}
        {segs.map((s, i) => (
          <text
            key={`t${i}`}
            x={170 + (i % 2) * 190}
            y={28 + Math.floor(i / 2) * 22}
            fontSize="11"
            fill="var(--fg-dim, #9aa2ad)"
          >
            <tspan fill={s.color}>■</tspan> {s.label} {s.pct}%
          </text>
        ))}
      </svg>
    );
  }

  const padL = 8;
  const padT = 14;
  const padB = 22;
  const plotW = W - padL - 8;
  const plotH = H - padB - padT;
  const max = Math.max(...vals.map(Math.abs), 1e-9);
  const step = plotW / vals.length;

  if (chartType === "line") {
    const pts = vals.map((v, i) => ({
      x: padL + step * (i + 0.5),
      y: padT + plotH - (Math.abs(v) / max) * plotH,
    }));
    return (
      <svg viewBox={`0 0 ${W} ${H}`} className="w-full max-w-[560px] mt-1.5" data-testid="mini-chart" role="img">
        <polyline
          points={pts.map((p) => `${p.x.toFixed(1)},${p.y.toFixed(1)}`).join(" ")}
          fill="none"
          stroke={CHART_BLUE}
          strokeWidth="2"
        />
        {pts.map((p, i) => (
          <g key={i}>
            <circle cx={p.x.toFixed(1)} cy={p.y.toFixed(1)} r="3" fill={CHART_BLUE} />
            <text x={p.x.toFixed(1)} y={(p.y - 6).toFixed(1)} fontSize="9" textAnchor="middle" fill="var(--fg-dim, #9aa2ad)">
              {vals[i]}
            </text>
            {lbls[i] !== undefined && (
              <text x={p.x.toFixed(1)} y={H - 6} fontSize="9" textAnchor="middle" fill="var(--fg-faint, #6b7280)">
                {trunc(lbls[i])}
              </text>
            )}
          </g>
        ))}
      </svg>
    );
  }

  // bar（默认；scatter 归一为柱状展示）
  return (
    <svg viewBox={`0 0 ${W} ${H}`} className="w-full max-w-[560px] mt-1.5" data-testid="mini-chart" role="img">
      {vals.map((v, i) => {
        const h = Math.max(1, (Math.abs(v) / max) * plotH);
        const x = padL + step * i + step * 0.15;
        const w = step * 0.7;
        const y = padT + plotH - h;
        return (
          <g key={i}>
            <rect x={x.toFixed(1)} y={y.toFixed(1)} width={w.toFixed(1)} height={h.toFixed(1)} rx="2" fill={CHART_BLUE} />
            <text x={(x + w / 2).toFixed(1)} y={(y - 4).toFixed(1)} fontSize="9" textAnchor="middle" fill="var(--fg-dim, #9aa2ad)">
              {v}
            </text>
            {lbls[i] !== undefined && (
              <text x={(x + w / 2).toFixed(1)} y={H - 6} fontSize="9" textAnchor="middle" fill="var(--fg-faint, #6b7280)">
                {trunc(lbls[i])}
              </text>
            )}
          </g>
        );
      })}
    </svg>
  );
}