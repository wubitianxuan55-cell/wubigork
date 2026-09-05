// GenUI 组件渲染：白名单节点 → React 元素。
// 安全边界：本文件只消费 guard 修复后的节点；不做任何 raw HTML。
/* eslint-disable react-refresh/only-export-components -- 渲染分发树按族集中维护 */
import { memo, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import {
  type GenuiAvatar,
  type GenuiBadge,
  type GenuiButton,
  type GenuiCallout,
  type GenuiCard,
  type GenuiChart,
  type GenuiCheckbox,
  type GenuiCode,
  type GenuiCol,
  type GenuiCopy,
  type GenuiDiff,
  type GenuiGrid,
  type GenuiInput,
  type GenuiJson,
  type GenuiKeyValue,
  type GenuiList,
  type GenuiNode,
  type GenuiProgress,
  type GenuiQuiz,
  type GenuiRadio,
  type GenuiRow,
  type GenuiSelect,
  type GenuiSlider,
  type GenuiStat,
  type GenuiSteps,
  type GenuiSubmit,
  type GenuiSwitch,
  type GenuiTable,
  type GenuiTabs,
  type GenuiText,
  type GenuiTextarea,
  type GenuiTimeline,
} from "./spec";
import { useBlockApi } from "./blocks/state";

function cx(...parts: Array<string | false | undefined>): string {
  return parts.filter(Boolean).join(" ");
}

function renderItems(items: GenuiNode[], depth: number): ReactNode[] {
  return items.map((node, i) => (
    <div className="gui-item" key={`${i}-${depth}`}>
      {renderNode(node, depth + 1)}
    </div>
  ));
}

// ─── 布局 ───────────────────────────────────────────────

function TextNode({ node }: { node: GenuiText }) {
  return (
    <div
      className={cx(
        "gui-text",
        node.size ? `gui-text-${node.size}` : "gui-text-body",
        node.center === true && "gui-center",
      )}
    >
      {node.content}
    </div>
  );
}

function RowNode({ node, depth }: { node: GenuiRow; depth: number }) {
  return (
    <div
      className={cx("gui-row", node.wrap === true && "gui-wrap")}
      style={node.gap !== undefined ? { gap: node.gap } : undefined}
    >
      {renderItems(node.items, depth)}
    </div>
  );
}

function ColNode({ node, depth }: { node: GenuiCol; depth: number }) {
  return (
    <div
      className={cx("gui-col", node.wrap === true && "gui-wrap")}
      style={node.gap !== undefined ? { gap: node.gap } : undefined}
    >
      {renderItems(node.items, depth)}
    </div>
  );
}

function GridNode({ node, depth }: { node: GenuiGrid; depth: number }) {
  return (
    <div
      className="gui-grid"
      style={{ gridTemplateColumns: `repeat(${Math.max(1, Math.min(12, node.cols))}, minmax(0, 1fr))` }}
    >
      {renderItems(node.items, depth)}
    </div>
  );
}

function CardNode({ node, depth }: { node: GenuiCard; depth: number }) {
  return (
    <div className="gui-card">
      {node.title !== undefined && <div className="gui-card-title">{node.title}</div>}
      <div className="gui-card-body">{renderItems(node.items, depth)}</div>
    </div>
  );
}

// ─── 展示 ───────────────────────────────────────────────

function StatNode({ node }: { node: GenuiStat }) {
  const deltaClass =
    node.delta !== undefined && node.delta.startsWith("-")
      ? "gui-delta-down"
      : node.delta !== undefined && node.delta.startsWith("+")
        ? "gui-delta-up"
        : undefined;
  return (
    <div className="gui-stat">
      <div className="gui-stat-label">{node.label}</div>
      <div className="gui-stat-value">{node.value}</div>
      {node.delta !== undefined && (
        <div className={cx("gui-stat-delta", deltaClass)}>{node.delta}</div>
      )}
    </div>
  );
}

const BADGE_TONES = new Set(["success", "warn", "danger", "accent"]);

function BadgeNode({ node }: { node: GenuiBadge }) {
  return (
    <span className={cx("gui-badge", BADGE_TONES.has(node.tone ?? "") && `gui-tone-${node.tone}`)}>
      {node.icon !== undefined && <span className="gui-badge-icon">{node.icon}</span>}
      {node.label}
    </span>
  );
}

function ProgressNode({ node }: { node: GenuiProgress }) {
  return (
    <div className="gui-progress">
      {node.label !== undefined && (
        <div className="gui-progress-head">
          <span>{node.label}</span>
          {node.valueLabel !== undefined && <span className="gui-muted">{node.valueLabel}</span>}
        </div>
      )}
      <div
        className="gui-progress-track"
        role="progressbar"
        aria-valuenow={Math.round(node.value)}
        aria-valuemin={0}
        aria-valuemax={100}
      >
        <div className="gui-progress-bar" style={{ width: `${node.value}%` }} />
      </div>
    </div>
  );
}

function KeyValueNode({ node }: { node: GenuiKeyValue }) {
  return (
    <div className="gui-kv">
      {node.pairs.map((p, i) => (
        <div className="gui-kv-row" key={`${p.key}-${i}`}>
          <span className="gui-kv-key">{p.key}</span>
          <span className="gui-kv-value">{p.value}</span>
        </div>
      ))}
    </div>
  );
}

function ListNode({ node, depth }: { node: GenuiList; depth: number }) {
  return (
    <ul className="gui-list">
      {node.items.map((item, i) => (
        <li key={i} className="gui-list-item">
          {typeof item === "string" ? item : renderNode(item, depth + 1)}
        </li>
      ))}
    </ul>
  );
}

function numericValue(text: string): number | null {
  const t = text.trim();
  if (t === "") return null;
  const cleaned = t.replace(/,/g, "");
  const m = /^[^\d\-+]*(-?\d+(?:\.\d+)?)\s*([kKmMbB万亿%]?)[^\d]*$/.exec(cleaned);
  if (!m) return null;
  let value = parseFloat(m[1]);
  if (!Number.isFinite(value)) return null;
  const suffix = m[2];
  if (suffix === "k" || suffix === "K") value *= 1e3;
  else if (suffix === "m" || suffix === "M") value *= 1e6;
  else if (suffix === "b" || suffix === "B") value *= 1e9;
  else if (suffix === "万") value *= 1e4;
  else if (suffix === "亿") value *= 1e8;
  return value;
}

type SortDir = "asc" | "desc" | null;

const SortableTable = memo(function SortableTable({ node }: { node: GenuiTable }) {
  const [sort, setSort] = useState<{ col: number; dir: SortDir } | null>(null);
  const sorted = useMemo(() => {
    if (!sort) return node.rows;
    const rows = [...node.rows];
    const col = sort.col;
    rows.sort((a, b) => {
      const an = numericValue(String(a[col] ?? ""));
      const bn = numericValue(String(b[col] ?? ""));
      let cmp: number;
      if (an !== null && bn !== null) cmp = an - bn;
      else cmp = String(a[col] ?? "").localeCompare(String(b[col] ?? ""), "zh-Hans-CN");
      return sort.dir === "desc" ? -cmp : cmp;
    });
    return rows;
  }, [node.rows, sort]);

  const toggle = (col: number): void => {
    setSort((prev) => {
      if (!prev || prev.col !== col) return { col, dir: "asc" };
      if (prev.dir === "asc") return { col, dir: "desc" };
      return null;
    });
  };

  return (
    <div className="gui-table-wrap">
      <table className="gui-table">
        <thead>
          <tr>
            {node.columns.map((label, i) => (
              <th key={i} className={cx("gui-th", sort?.col === i && "gui-th-sorted")}>
                <button type="button" className="gui-th-btn" onClick={() => toggle(i)}>
                  <span>{label}</span>
                  <span className="gui-sort-arrow" aria-hidden>
                    {sort?.col === i ? (sort.dir === "asc" ? "↑" : "↓") : ""}
                  </span>
                </button>
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {sorted.map((row, r) => (
            <tr key={r}>
              {row.map((cell, c) => (
                <td key={c} className={cx("gui-td", numericValue(String(cell)) !== null && "gui-num")}>
                  {String(cell)}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
});

function TimelineNode({ node }: { node: GenuiTimeline }) {
  return (
    <ol className="gui-timeline">
      {node.items.map((item, i) => (
        <li key={i} className="gui-timeline-item">
          <div className="gui-timeline-dot" />
          <div className="gui-timeline-body">
            <div className="gui-timeline-head">
              <span className="gui-timeline-title">{item.title}</span>
              {item.time !== undefined && <span className="gui-muted gui-timeline-time">{item.time}</span>}
            </div>
            {item.desc !== undefined && <div className="gui-timeline-desc">{item.desc}</div>}
          </div>
        </li>
      ))}
    </ol>
  );
}

const CALLOUT_TONES = new Set(["info", "success", "warning", "error"]);

function CalloutNode({ node }: { node: GenuiCallout }) {
  return (
    <div className={cx("gui-callout", CALLOUT_TONES.has(node.tone ?? "") && `gui-tone-${node.tone}`)}>
      {node.title !== undefined && <div className="gui-callout-title">{node.title}</div>}
      <div className="gui-callout-body">{node.content}</div>
    </div>
  );
}

function StepsNode({ node }: { node: GenuiSteps }) {
  const current = node.current ?? 1;
  return (
    <ol className="gui-steps">
      {node.steps.map((step, i) => {
        const n = i + 1;
        return (
          <li key={i} className={cx("gui-step", n === current && "gui-step-current")}>
            <div className="gui-step-head">
              <span className="gui-step-no">{n <= current ? "✓" : n}</span>
              <span className="gui-step-title">{step.title}</span>
            </div>
            {step.desc !== undefined && <div className="gui-step-desc">{step.desc}</div>}
          </li>
        );
      })}
    </ol>
  );
}

function AvatarNode({ node }: { node: GenuiAvatar }) {
  const first = [...node.name.trim()][0] ?? "?";
  return (
    <span
      className="gui-avatar"
      style={node.color !== undefined ? { background: node.color } : undefined}
      aria-hidden
    >
      {first.toUpperCase()}
    </span>
  );
}

function CopyNode({ node }: { node: GenuiCopy }) {
  const [copied, setCopied] = useState(false);
  const copy = (): void => {
    const text = node.text;
    if (navigator.clipboard?.writeText) {
      void navigator.clipboard.writeText(text).then(() => {
        setCopied(true);
        window.setTimeout(() => setCopied(false), 1200);
      }).catch(() => fallbackCopy(text));
    } else {
      fallbackCopy(text);
    }
  };
  return (
    <button type="button" className="gui-btn gui-btn-ghost gui-btn-small gui-copy" onClick={copy}>
      {copied ? "已复制" : (node.label ?? "复制")}
    </button>
  );
}

function fallbackCopy(text: string): void {
  try {
    const ta = document.createElement("textarea");
    ta.value = text;
    ta.style.position = "fixed";
    ta.style.opacity = "0";
    document.body.appendChild(ta);
    ta.select();
    document.execCommand("copy");
    document.body.removeChild(ta);
  } catch {
    // noop
  }
}

// ─── 轻图表（手写 SVG） ─────────────────────────────────

const DEFAULT_SERIES = ["var(--gaea-glow)", "var(--color-primary)", "var(--color-success)", "var(--color-warning)"];

function seriesColor(chart: GenuiChart, s: { color?: string }, i: number): string {
  return s.color ?? DEFAULT_SERIES[i % DEFAULT_SERIES.length] ?? "var(--gaea-glow)";
}

function ChartNode({ node }: { node: GenuiChart }) {
  const kind = node.kind ?? "bars";
  const series = node.series && node.series.length > 0
    ? node.series
    : [{ label: "data", data: node.data }];
  const hasSeries = (node.series?.length ?? 0) > 0;
  if (kind === "donut") {
    return <Donut chart={node} />;
  }
  const allValues = series.flatMap((s) => s.data.map((d) => d.value));
  const maxAbs = Math.max(1, ...allValues.map((v) => Math.abs(v)));
  const W = 560;
  const H = 260;
  const padL = 38;
  const padR = 12;
  const padT = 16;
  const padB = 34;
  const plotW = W - padL - padR;
  const plotH = H - padT - padB;
  const baseline = (v: number): number => padT + plotH - ((v + maxAbs) / (2 * maxAbs)) * plotH;
  const x = (i: number, n: number): number =>
    n <= 1 ? padL + plotW / 2 : padL + (i / (n - 1)) * plotW;
  return (
    <div className="gui-chart">
      {series.length > 1 && (
        <div className="gui-chart-legend">
          {series.map((s, i) => (
            <span key={s.label} className="gui-chart-legend-item">
              <span className="gui-chart-legend-swatch" style={{ background: seriesColor(node, s, i) }} />
              {s.label}
            </span>
          ))}
        </div>
      )}
      <svg viewBox={`0 0 ${W} ${H}`} className="gui-chart-svg" role="img">
        {kind === "bars" && (
          <g>
            {series.map((s, si) => {
              const color = seriesColor(node, s, si);
              const n = s.data.length;
              const groupW = hasSeries ? plotW / Math.max(1, n) : plotW / Math.max(1, n);
              const barW = Math.max(4, groupW / (hasSeries ? series.length + 1 : 1) - 4);
              return s.data.map((d, i) => {
                const y = baseline(d.value);
                const gx = padL + (hasSeries ? (i + 0.5) * groupW - (barW * series.length) / 2 + si * barW + barW / 2 : (i + 0.5) * groupW);
                return (
                  <rect
                    key={`${si}-${i}`}
                    x={gx - barW / 2}
                    y={Math.min(y, baseline(0))}
                    width={barW}
                    height={Math.max(0, Math.abs(baseline(0) - y))}
                    fill={color}
                    rx={2}
                  >
                    <title>{`${d.label}: ${d.value}`}</title>
                  </rect>
                );
              });
            })}
            {series[0]?.data.map((d, i) => (
              <text
                key={`x-${i}`}
                x={padL + (hasSeries ? (i + 0.5) * (plotW / Math.max(1, series[0].data.length)) : x(i, series[0].data.length))}
                y={H - 10}
                textAnchor="middle"
                className="gui-chart-axis"
              >
                {d.label}
              </text>
            ))}
          </g>
        )}
        {kind === "line" &&
          series.map((s, si) => {
            const color = seriesColor(node, s, si);
            const n = s.data.length;
            const pts = s.data.map((d, i) => `${x(i, n).toFixed(1)},${baseline(d.value).toFixed(1)}`);
            return (
              <g key={s.label}>
                <polyline points={pts.join(" ")} fill="none" stroke={color} strokeWidth={2} strokeLinejoin="round" />
                {s.data.map((d, i) => (
                  <circle key={`${si}-${i}`} cx={x(i, n)} cy={baseline(d.value)} r={3} fill={color}>
                    <title>{`${d.label}: ${d.value}`}</title>
                  </circle>
                ))}
              </g>
            );
          })}
        {kind === "line" &&
          series[0]?.data.map((d, i) => (
            <text key={`xl-${i}`} x={x(i, series[0].data.length)} y={H - 10} textAnchor="middle" className="gui-chart-axis">
              {d.label}
            </text>
          ))}
      </svg>
    </div>
  );
}

function Donut({ chart }: { chart: GenuiChart }) {
  const data = chart.data;
  const total = data.reduce((acc, d) => acc + Math.max(0, d.value), 0) || 1;
  const r = 70;
  const C = 2 * Math.PI * r;
  let acc = 0;
  return (
    <div className="gui-chart gui-donut">
      <svg viewBox="0 0 200 200" className="gui-chart-svg" role="img">
        {data.map((d, i) => {
          const frac = Math.max(0, d.value) / total;
          const dash = frac * C;
          const offset = -acc * C;
          acc += frac;
          const color = d.color ?? DEFAULT_SERIES[i % DEFAULT_SERIES.length] ?? "var(--gaea-glow)";
          return (
            <circle
              key={`${d.label}-${i}`}
              cx={100}
              cy={100}
              r={r}
              fill="none"
              stroke={color}
              strokeWidth={26}
              strokeDasharray={`${dash} ${C - dash}`}
              strokeDashoffset={offset}
              transform="rotate(-90 100 100)"
            >
              <title>{`${d.label}: ${d.value}`}</title>
            </circle>
          );
        })}
        <text x={100} y={104} textAnchor="middle" className="gui-donut-total">
          {Math.round(total)}
        </text>
      </svg>
    </div>
  );
}

// ─── 代码展示 ───────────────────────────────────────────

function CodeNode({ node }: { node: GenuiCode }) {
  return (
    <div className="gui-code-block">
      {node.lang !== undefined && <div className="gui-code-lang">{node.lang}</div>}
      <pre className="gui-code-pre">
        <code>{node.code}</code>
      </pre>
    </div>
  );
}

function JsonNode({ node }: { node: GenuiJson }) {
  const pretty = useMemo(() => {
    try {
      return JSON.stringify(node.value, null, 2);
    } catch {
      return String(node.value);
    }
  }, [node.value]);
  return (
    <div className="gui-code-block">
      <pre className="gui-code-pre">
        <code>{pretty}</code>
      </pre>
    </div>
  );
}

function DiffNode({ node }: { node: GenuiDiff }) {
  return (
    <div className="gui-diffs">
      {node.diffs.map((d, i) => (
        <div className="gui-diff" key={`${d.path}-${i}`}>
          <div className="gui-diff-path">{d.path}</div>
          {d.oldText !== undefined && d.oldText !== null && (
            <pre className="gui-diff-old">{d.oldText}</pre>
          )}
          <pre className="gui-diff-new">{d.newText}</pre>
        </div>
      ))}
    </div>
  );
}

// ─── 交互 ───────────────────────────────────────────────

function useActionFeedback(): { triggered: string | null; mark: (name: string) => void } {
  const [triggered, setTriggered] = useState<string | null>(null);
  const mark = (name: string): void => {
    setTriggered(name);
    window.setTimeout(() => {
      setTriggered((prev) => (prev === name ? null : prev));
    }, 900);
  };
  return { triggered, mark };
}

function ButtonNode({ node }: { node: GenuiButton }) {
  const api = useBlockApi();
  const enabled = node.action !== undefined && api?.hasAction === true;
  const { triggered, mark } = useActionFeedback();
  return (
    <button
      type="button"
      disabled={!enabled}
      className={cx(
        "gui-btn",
        node.tone && `gui-btn-${node.tone}`,
        node.full === true && "gui-btn-full",
        node.small === true && "gui-btn-small",
        !enabled && "gui-btn-disabled",
      )}
      onClick={() => {
        if (!enabled || node.action === undefined || !api) return;
        mark(node.action);
        api.emit(node.action, {});
      }}
    >
      {node.icon !== undefined && <span className="gui-btn-icon">{node.icon}</span>}
      <span>{triggered === node.action ? "已触发" : node.label}</span>
    </button>
  );
}

const INPUT_TYPES = new Set(["text", "email", "password"]);

function InputNode({ node }: { node: GenuiInput }) {
  const api = useBlockApi();
  const [value, setValue] = useState(node.value ?? "");
  const changedRef = useRef(false);
  const type = INPUT_TYPES.has(node.inputType ?? "") ? node.inputType : "text";
  const isSecret = type === "password";
  useEffect(() => {
    if (isSecret && node.id !== undefined && api) api.registerSecret(node.id);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
  const submit = (): void => {
    if (node.action === undefined || !api?.hasAction) return;
    if (changedRef.current || value !== (node.value ?? "")) {
      changedRef.current = false;
      const payload: Record<string, unknown> = { value };
      if (node.id !== undefined) payload.id = node.id;
      if (!isSecret && node.id !== undefined) api.setField(node.id, value);
      api.emit(node.action, payload);
    }
  };
  return (
    <label className="gui-field">
      {node.label !== undefined && <span className="gui-field-label">{node.label}</span>}
      <input
        className="gui-input"
        type={type}
        value={value}
        placeholder={node.placeholder}
        onChange={(e) => {
          changedRef.current = true;
          setValue(e.target.value);
          if (node.action === undefined && node.id !== undefined && !isSecret && api) {
            api.setField(node.id, e.target.value);
          }
        }}
        onBlur={submit}
        onKeyDown={(e) => {
          if (e.key === "Enter") submit();
        }}
      />
    </label>
  );
}

function SelectNode({ node }: { node: GenuiSelect }) {
  const api = useBlockApi();
  const [selected, setSelected] = useState<number>(node.selected ?? -1);
  const change = (idx: number): void => {
    setSelected(idx);
    const label = idx >= 0 && idx < node.options.length ? node.options[idx] : "";
    if (node.id !== undefined && api && label !== "") api.setField(node.id, label);
    if (node.action !== undefined && api?.hasAction && idx >= 0) {
      api.emit(node.action, { selected: label, index: idx, id: node.id });
    }
  };
  return (
    <label className="gui-field">
      {node.label !== undefined && <span className="gui-field-label">{node.label}</span>}
      <select
        className="gui-select"
        value={selected}
        onChange={(e) => change(Number(e.target.value))}
      >
        <option value={-1}>{node.label ? "请选择…" : "请选择…"}</option>
        {node.options.map((opt, i) => (
          <option key={opt} value={i}>
            {opt}
          </option>
        ))}
      </select>
    </label>
  );
}

function ToggleNode({ node }: { node: GenuiCheckbox | GenuiSwitch }) {
  const api = useBlockApi();
  const [checked, setChecked] = useState(node.checked ?? false);
  const enabled = node.action !== undefined && api?.hasAction === true;
  return (
    <label className={cx("gui-toggle", node.type === "switch" && "gui-switch")}>
      <input
        type="checkbox"
        className="gui-toggle-input"
        checked={checked}
        disabled={!enabled}
        onChange={() => {
          const next = !checked;
          setChecked(next);
          if (node.action !== undefined && api?.hasAction) {
            api.emit(node.action, { checked: next, label: node.label });
          }
        }}
      />
      <span className="gui-toggle-track" aria-hidden />
      <span className="gui-toggle-label">{node.label}</span>
    </label>
  );
}

function RadioNode({ node }: { node: GenuiRadio }) {
  const api = useBlockApi();
  const groupMode = node.group !== undefined && node.group !== "";
  const inGroup = groupMode && api !== null;
  const groupValue = inGroup && node.group ? api?.answers[node.group] : undefined;
  const [local, setLocal] = useState<number>(node.selected ?? -1);
  useEffect(() => {
    if (node.group !== undefined && node.answer !== undefined && api) {
      api.registerMeta(node.group, {
        label: node.options[0] ?? "",
        answer: node.answer,
        explanation: node.explanation,
      });
    }
    // 仅挂载注册一次；meta 恒定不随渲染变化
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
  const pick = (idx: number): void => {
    if (api?.locked === true) return;
    const label = node.options[idx];
    if (inGroup && node.group && api) {
      api.setAnswer(node.group, label);
      return;
    }
    setLocal(idx);
    if (node.action !== undefined && api?.hasAction) {
      api.emit(node.action, { selected: label, index: idx, group: node.group });
    }
  };
  const groupIdx = inGroup && node.group ? node.options.indexOf(groupValue ?? "") : -1;
  const selectedIdx = groupIdx >= 0 ? groupIdx : (node.group ? -1 : local);
  return (
    <div className="gui-radio" role="radiogroup" aria-label={node.label}>
      {node.label !== undefined && <div className="gui-field-label">{node.label}</div>}
      {node.options.map((opt, i) => (
        <label key={opt} className={cx("gui-radio-row", api?.locked === true && "gui-disabled")}>
          <input
            type="radio"
            className="gui-radio-input"
            checked={selectedIdx === i}
            disabled={api?.locked === true}
            onChange={() => pick(i)}
          />
          <span>{opt}</span>
        </label>
      ))}
    </div>
  );
}

function SliderNode({ node }: { node: GenuiSlider }) {
  const api = useBlockApi();
  const min = node.min ?? 0;
  const max = node.max ?? 100;
  const step = node.step ?? 1;
  const [value, setValue] = useState<number>(node.value ?? min);
  const timer = useRef(0);
  const change = (next: number): void => {
    setValue(next);
    if (node.id !== undefined && api) api.setField(node.id, String(next));
    if (node.action === undefined || !api?.hasAction) return;
    window.clearTimeout(timer.current);
    timer.current = window.setTimeout(() => {
      api.emit(node.action as string, { value: next, id: node.id });
    }, 300);
  };
  useEffect(() => () => window.clearTimeout(timer.current), []);
  return (
    <label className="gui-field">
      <span className="gui-field-label">
        {node.label ?? ""}
        <span className="gui-slider-value">{value}</span>
      </span>
      <input
        type="range"
        className="gui-slider"
        min={min}
        max={max}
        step={step}
        value={value}
        onChange={(e) => change(Number(e.target.value))}
      />
    </label>
  );
}

function TextareaNode({ node }: { node: GenuiTextarea }) {
  const api = useBlockApi();
  const [value, setValue] = useState(node.value ?? "");
  const changedRef = useRef(false);
  const submit = (): void => {
    if (node.action === undefined || !api?.hasAction) return;
    if (changedRef.current || value !== (node.value ?? "")) {
      changedRef.current = false;
      if (node.id !== undefined) api.setField(node.id, value);
      api.emit(node.action, { value, id: node.id });
    }
  };
  return (
    <label className="gui-field">
      {node.label !== undefined && <span className="gui-field-label">{node.label}</span>}
      <textarea
        className="gui-textarea"
        rows={node.rows ?? 3}
        value={value}
        placeholder={node.placeholder}
        onChange={(e) => {
          changedRef.current = true;
          setValue(e.target.value);
        }}
        onBlur={submit}
        onKeyDown={(e) => {
          if ((e.ctrlKey || e.metaKey) && e.key === "Enter") submit();
        }}
      />
    </label>
  );
}

function SubmitNode({ node }: { node: GenuiSubmit }) {
  const api = useBlockApi();
  const groups = node.groups ?? [];
  const graded =
    groups.length > 0 &&
    groups.every((g) => api?.meta[g] !== undefined && api.meta[g].answer !== undefined);
  const answeredAll =
    groups.length > 0 && groups.every((g) => (api?.answers[g] ?? "") !== "");
  const enabled =
    api !== null &&
    !api.locked &&
    (graded ? answeredAll : (node.action !== undefined && api.hasAction) || (node.action === undefined && groups.length === 0 && api.hasAction));
  const gradeNow = (): void => {
    if (!api || api.locked) return;
    if (graded && answeredAll) {
      api.lock();
      return;
    }
    if (node.action !== undefined && api.hasAction) {
      const answers: Record<string, string> = {};
      for (const g of groups) {
        const v = api.answers[g];
        if (v !== undefined) answers[g] = v;
      }
      const fields = { ...api.fields };
      api.emit(node.action, {
        answers,
        fields,
        total: groups.length,
        answered: Object.keys(answers).length,
      });
    }
  };
  if (api?.locked === true) {
    const results = groups.map((g) => {
      const meta = api.meta[g];
      const chosen = api.answers[g] ?? "";
      const ok = meta !== undefined && String(meta.answer) === chosen;
      return { g, chosen, ok, explanation: meta?.explanation };
    });
    const score = results.filter((r) => r.ok).length;
    return (
      <div className="gui-grade-result">
        <div className="gui-grade-score">
          得分 {score}/{results.length}
        </div>
        {results.map((r) => (
          <div key={r.g} className={cx("gui-grade-row", r.ok ? "gui-ok" : "gui-err")}>
            <span>{r.ok ? "✓" : "✗"}</span>
            <span>
              {r.g}：{r.chosen || "未作答"}
              {!r.ok && r.explanation !== undefined && <span className="gui-muted"> · {r.explanation}</span>}
            </span>
          </div>
        ))}
        <button
          type="button"
          className="gui-btn gui-btn-ghost gui-btn-small"
          onClick={() => api?.reset(node.resetAction)}
        >
          重新作答
        </button>
      </div>
    );
  }
  return (
    <button type="button" disabled={!enabled} className={cx("gui-btn gui-btn-primary", !enabled && "gui-btn-disabled")} onClick={gradeNow}>
      {node.label ?? "交卷"}
    </button>
  );
}

function TabsNode({ node, depth }: { node: GenuiTabs; depth: number }) {
  const [active, setActive] = useState(0);
  const safe = Math.min(active, node.tabs.length - 1);
  const onKey = (e: React.KeyboardEvent, idx: number): void => {
    const last = node.tabs.length - 1;
    if (e.key === "ArrowRight") setActive(idx >= last ? 0 : idx + 1);
    else if (e.key === "ArrowLeft") setActive(idx <= 0 ? last : idx - 1);
    else if (e.key === "Home") setActive(0);
    else if (e.key === "End") setActive(last);
  };
  return (
    <div className="gui-tabs">
      <div className="gui-tabs-bar" role="tablist">
        {node.tabs.map((tab, i) => (
          <button
            key={tab.label}
            type="button"
            role="tab"
            aria-selected={safe === i}
            className={cx("gui-tab-btn", safe === i && "gui-tab-active")}
            onClick={() => setActive(i)}
            onKeyDown={(e) => onKey(e, i)}
          >
            {tab.label}
          </button>
        ))}
      </div>
      <div className="gui-tab-panel" role="tabpanel">
        {renderItems(node.tabs[safe]?.items ?? [], depth)}
      </div>
    </div>
  );
}

function AccordionReal({ node, depth }: { node: { type: "accordion"; items: { title: string; items: GenuiNode[] }[] }; depth: number }) {
  const [open, setOpen] = useState<number | null>(0);
  return (
    <div className="gui-accordion">
      {node.items.map((item, i) => (
        <div key={item.title} className="gui-accordion-item">
          <button
            type="button"
            className="gui-accordion-head"
            aria-expanded={open === i}
            onClick={() => setOpen((prev) => (prev === i ? null : i))}
          >
            <span>{item.title}</span>
            <span className="gui-accordion-arrow" aria-hidden>
              {open === i ? "▾" : "▸"}
            </span>
          </button>
          {open === i && <div className="gui-accordion-body">{renderItems(item.items, depth)}</div>}
        </div>
      ))}
    </div>
  );
}

function QuizNode({ node }: { node: GenuiQuiz }) {
  const api = useBlockApi();
  const [picked, setPicked] = useState<number | null>(null);
  const [revealed, setRevealed] = useState(false);
  const enabled = node.action !== undefined && api?.hasAction === true;
  const choose = (idx: number): void => {
    if (revealed) return;
    setPicked(idx);
    setRevealed(true);
    const correct = node.options[idx]?.correct === true;
    if (enabled && node.action !== undefined && api) {
      api.emit(node.action, {
        type: "quiz",
        question: node.question,
        answer: node.options[idx]?.label,
        correct,
      });
    }
  };
  const correct = picked !== null && node.options[picked]?.correct === true;
  return (
    <div className="gui-quiz" key={node.id ?? "quiz"}>
      <div className="gui-quiz-question">{node.question}</div>
      <div className="gui-quiz-options" role="radiogroup">
        {node.options.map((opt, i) => (
          <button
            key={opt.label}
            type="button"
            className={cx(
              "gui-quiz-opt",
              revealed && picked === i && (opt.correct === true ? "gui-ok" : "gui-err"),
              revealed && opt.correct === true && "gui-ok",
            )}
            onClick={() => choose(i)}
          >
            {opt.label}
            {revealed && picked === i && opt.feedback !== undefined && (
              <span className="gui-quiz-feedback"> · {opt.feedback}</span>
            )}
          </button>
        ))}
      </div>
      {revealed && (
        <div className={cx("gui-quiz-result", correct ? "gui-ok" : "gui-err")}>
          {correct ? "回答正确" : "回答错误"}
          {!correct && node.explanation !== undefined && (
            <div className="gui-muted">{node.explanation}</div>
          )}
          <button
            type="button"
            className="gui-btn gui-btn-ghost gui-btn-small"
            onClick={() => {
              setPicked(null);
              setRevealed(false);
            }}
          >
            重试
          </button>
        </div>
      )}
    </div>
  );
}

// ─── 分发 ───────────────────────────────────────────────

export function renderNode(node: GenuiNode, depth = 0): ReactNode {
  switch (node.type) {
    case "text":
      return <TextNode node={node} />;
    case "row":
      return <RowNode node={node} depth={depth} />;
    case "col":
      return <ColNode node={node} depth={depth} />;
    case "grid":
      return <GridNode node={node} depth={depth} />;
    case "card":
      return <CardNode node={node} depth={depth} />;
    case "divider":
      return <hr className="gui-divider" />;
    case "spacer":
      return <div className="gui-spacer" />;
    case "stat":
      return <StatNode node={node} />;
    case "badge":
      return <BadgeNode node={node} />;
    case "progress":
      return <ProgressNode node={node} />;
    case "keyvalue":
      return <KeyValueNode node={node} />;
    case "list":
      return <ListNode node={node} depth={depth} />;
    case "table":
      return <SortableTable node={node} />;
    case "timeline":
      return <TimelineNode node={node} />;
    case "callout":
      return <CalloutNode node={node} />;
    case "steps":
      return <StepsNode node={node} />;
    case "avatar":
      return <AvatarNode node={node} />;
    case "copy":
      return <CopyNode node={node} />;
    case "chart":
      return <ChartNode node={node} />;
    case "code":
      return <CodeNode node={node} />;
    case "json":
      return <JsonNode node={node} />;
    case "diff":
      return <DiffNode node={node} />;
    case "button":
      return <ButtonNode node={node} />;
    case "input":
      return <InputNode node={node} />;
    case "select":
      return <SelectNode node={node} />;
    case "checkbox":
    case "switch":
      return <ToggleNode node={node} />;
    case "radio":
      return <RadioNode node={node} />;
    case "slider":
      return <SliderNode node={node} />;
    case "textarea":
      return <TextareaNode node={node} />;
    case "submit":
      return <SubmitNode node={node} />;
    case "tabs":
      return <TabsNode node={node} depth={depth} />;
    case "accordion":
      return <AccordionReal node={node} depth={depth} />;
    case "quiz":
      return <QuizNode node={node} />;
    default:
      return null;
  }
}
