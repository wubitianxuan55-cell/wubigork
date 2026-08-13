import { useCallback, useEffect, useState } from "react";
import { RobotOutlined } from "@ant-design/icons";
import { EmptyState } from "../EmptyState";
import { RefreshCw } from "../../icons";
import { app } from "../../lib/bridge";

interface DigitalCharacter {
  id: string;
  name: string;
  gender: string;
  identity: string;
  worldview: string;
  text_model: string;
  intimacy: number;
  trust: number;
  safety: number;
  conflict: number;
  last_interacted_at: string;
  memory_summary: string;
  highlights: string[];
  memory_event_count: number;
  reinforcement: number;
  updated_at: string;
}

interface DigitalEvent {
  type: string;
  title: string;
  summary: string;
  occurred_at: string;
}

interface DigitalLife {
  available: boolean;
  source: string;
  error?: string;
  character_count: number;
  timeline_events: number;
  state_commits: number;
  world_events: number;
  memory_events: number;
  memory_summaries: number;
  relationships: number;
  turn_traces: number;
  characters: DigitalCharacter[];
  recent_timeline: DigitalEvent[];
  recent_world: DigitalEvent[];
}

interface HerdOp {
  id: string;
  kind: string;
  model: string;
  status: string;
  stage: string;
  progress: number;
  artifacts: number;
  created_at: string;
  completed_at: string;
}

function fmtTime(iso: string): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getMonth() + 1}/${d.getDate()} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function kindLabel(kind: string): string {
  const map: Record<string, string> = {
    image_generate: "生图", model_start: "模型启动", model_download: "模型下载",
    speech: "语音合成", tts: "TTS", asr: "语音识别", music: "音乐生成",
  };
  return map[kind] ?? (kind || "任务");
}

function ScoreBar({ label, value, color }: { label: string; value: number; color: string }) {
  return (
    <div className="flex items-center gap-1.5 text-[10.5px]">
      <span className="w-8 shrink-0 text-fg-faint">{label}</span>
      <div className="flex-1 h-1 rounded bg-border-soft overflow-hidden">
        <div className="h-full rounded" style={{ width: `${Math.max(0, Math.min(100, value))}%`, background: color }} />
      </div>
      <span className="w-6 shrink-0 text-right text-fg-dim font-mono">{value}</span>
    </div>
  );
}

function OpStatusChip({ status }: { status: string }) {
  const tone = status === "completed" ? "ok" : status === "running" ? "accent" : status === "failed" || status === "cancelled" ? "danger" : "neutral";
  const cls = `inline-flex items-center rounded-full px-1.5 py-px text-[9.5px] leading-none border ${
    tone === "ok" ? "text-ok border-ok/30 bg-ok/10"
      : tone === "accent" ? "text-accent border-accent/30 bg-accent/10"
        : tone === "danger" ? "text-err border-err/30 bg-err/10"
          : "text-fg-faint border-border-soft bg-bg-soft"
  }`;
  return <span className={cls}>{status || "—"}</span>;
}

/** 数字生命库：Herdsman digital-life 虚拟人格记忆只读联动（角色/关系/记忆摘要/时间线/世界事件 + 最近操作）。 */
export function DigitalLifeLibrary() {
  const [life, setLife] = useState<DigitalLife | null>(null);
  const [ops, setOps] = useState<HerdOp[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(() => {
    setLoading(true);
    Promise.allSettled([app.HerdsmanDigitalLife(), app.HerdsmanOperations()])
      .then(([l, o]) => {
        if (l.status === "fulfilled") setLife(l.value as DigitalLife);
        if (o.status === "fulfilled") setOps((o.value as { items: HerdOp[] })?.items ?? []);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, []);

  useEffect(() => { load(); }, [load]);

  const header = (
    <div className="flex items-center justify-between px-3 py-2 border-b border-border-soft">
      <span className="flex items-center gap-1.5 font-semibold text-fg text-sm">
        <RobotOutlined className="text-accent" /> 数字生命 · Herdsman
      </span>
      <button
        type="button"
        className="flex items-center justify-center w-6 h-6 border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft rounded"
        onClick={load}
        title="刷新"
      >
        <RefreshCw size={12} className={loading ? "animate-spin" : ""} />
      </button>
    </div>
  );

  if (!life) {
    return (
      <div className="h-full flex flex-col">
        {header}
        <div className="flex-1 min-h-0 overflow-y-auto">
          <EmptyState message="数字生命不可用：Herdsman 数字生命库未启用或数据目录不可读" />
        </div>
      </div>
    );
  }
  if (!life.available || life.error) {
    return (
      <div className="h-full flex flex-col">
        {header}
        <div className="flex-1 min-h-0 overflow-y-auto">
          <EmptyState message={life.error || "数字生命不可用：请确认 Herdsman 已启用数字生命功能"} />
        </div>
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col">
      {header}
      <div className="flex-1 min-h-0 overflow-y-auto p-3 flex flex-col gap-3">
        {/* 统计条 */}
        <div className="flex flex-wrap gap-1.5">
          {[
            ["角色", life.character_count], ["时间线", life.timeline_events], ["世界事件", life.world_events],
            ["记忆摘要", life.memory_summaries], ["关系", life.relationships], ["状态提交", life.state_commits],
          ].map(([k, v]) => (
            <span key={k as string} className="inline-flex items-center gap-1 rounded-full border border-border-soft bg-bg-soft/50 px-2 py-0.5 text-[11px] text-fg-dim">
              <span className="text-fg-faint">{k}</span>
              <span className="font-semibold text-fg font-mono">{(v as number).toLocaleString()}</span>
            </span>
          ))}
        </div>

        {/* 角色卡片 */}
        {life.characters.length === 0 ? (
          <EmptyState message="暂无角色：Herdsman 数字生命尚未创建角色" />
        ) : (
          life.characters.map((c) => (
            <div key={c.id} className="rounded-lg border border-border-soft bg-bg-soft/30 p-3">
              <div className="flex items-center gap-2 flex-wrap">
                <span className="text-[14px] font-semibold text-fg">{c.name}</span>
                {c.identity && <span className="text-[11px] text-fg-dim">{c.identity}</span>}
                {c.gender && <span className="text-[10px] text-fg-faint">{c.gender}</span>}
                {c.worldview && <span className="text-[10px] text-fg-faint">世界观：{c.worldview}</span>}
                {c.text_model && (
                  <span className="ml-auto font-mono text-[9.5px] text-accent/80 bg-accent/10 rounded px-1.5 py-px">
                    {c.text_model}
                  </span>
                )}
              </div>
              <div className="mt-2 grid grid-cols-3 gap-x-3 gap-y-1 max-w-md">
                <ScoreBar label="亲密度" value={c.intimacy} color="#f472b6" />
                <ScoreBar label="信任" value={c.trust} color="#34d399" />
                <ScoreBar label="安全" value={c.safety} color="#38bdf8" />
              </div>
              <div className="mt-2 text-[11px] text-fg-faint">
                最近互动 {fmtTime(c.last_interacted_at)} · 记忆事件 {c.memory_event_count} · 强化 {c.reinforcement}
              </div>
              {c.memory_summary && (
                <p className="mt-1.5 text-[11.5px] text-fg-dim leading-relaxed line-clamp-3">{c.memory_summary}</p>
              )}
              {c.highlights.length > 0 && (
                <div className="mt-1.5 flex flex-wrap gap-1">
                  {c.highlights.map((h, i) => (
                    <span key={i} className="max-w-[320px] truncate rounded-full border border-border-soft bg-bg-soft px-1.5 py-px text-[9.5px] text-fg-faint" title={h}>
                      {h}
                    </span>
                  ))}
                </div>
              )}
            </div>
          ))
        )}

        {/* 最近时间线 / 世界事件 */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">
          <div className="rounded-lg border border-border-soft bg-bg-soft/20 p-2.5">
            <div className="text-[10px] uppercase tracking-wider text-fg-faint/70 font-medium mb-1.5">最近时间线</div>
            {life.recent_timeline.length === 0 ? (
              <div className="text-[11px] text-fg-faint/60 py-4 text-center">暂无</div>
            ) : (
              life.recent_timeline.map((e, i) => (
                <div key={i} className="py-1 border-b border-border-soft/50 last:border-b-0">
                  <div className="flex items-center gap-1.5 text-[11px]">
                    <span className="text-[9.5px] text-accent/80 bg-accent/10 rounded px-1 py-px">{e.type}</span>
                    <span className="truncate font-medium text-fg">{e.title || "—"}</span>
                    <span className="ml-auto shrink-0 text-[9.5px] text-fg-faint">{fmtTime(e.occurred_at)}</span>
                  </div>
                  {e.summary && <div className="text-[10.5px] text-fg-faint leading-snug line-clamp-1 mt-0.5">{e.summary}</div>}
                </div>
              ))
            )}
          </div>
          <div className="rounded-lg border border-border-soft bg-bg-soft/20 p-2.5">
            <div className="text-[10px] uppercase tracking-wider text-fg-faint/70 font-medium mb-1.5">世界事件</div>
            {life.recent_world.length === 0 ? (
              <div className="text-[11px] text-fg-faint/60 py-4 text-center">暂无</div>
            ) : (
              life.recent_world.map((e, i) => (
                <div key={i} className="py-1 border-b border-border-soft/50 last:border-b-0">
                  <div className="flex items-center gap-1.5 text-[11px]">
                    <span className="text-[9.5px] text-ok/80 bg-ok/10 rounded px-1 py-px">{e.type}</span>
                    <span className="truncate font-medium text-fg">{e.title || "—"}</span>
                    <span className="ml-auto shrink-0 text-[9.5px] text-fg-faint">{fmtTime(e.occurred_at)}</span>
                  </div>
                  {e.summary && <div className="text-[10.5px] text-fg-faint leading-snug line-clamp-1 mt-0.5">{e.summary}</div>}
                </div>
              ))
            )}
          </div>
        </div>

        {/* 最近 Herdsman 操作 */}
        {ops.length > 0 && (
          <div className="rounded-lg border border-border-soft bg-bg-soft/20 p-2.5">
            <div className="text-[10px] uppercase tracking-wider text-fg-faint/70 font-medium mb-1.5">
              最近 Herdsman 操作 · {ops.length}
            </div>
            <div className="flex flex-col">
              {ops.map((op) => (
                <div key={op.id} className="flex items-center gap-2 py-1 border-b border-border-soft/50 last:border-b-0 text-[11px]">
                  <span className="shrink-0 w-14 truncate text-fg-dim">{kindLabel(op.kind)}</span>
                  <span className="min-w-0 flex-1 truncate font-mono text-fg-faint" title={op.model}>{op.model || "—"}</span>
                  <span className="shrink-0 w-10 text-right text-fg-faint font-mono">{op.progress}%</span>
                  {op.artifacts > 0 && <span className="shrink-0 text-fg-faint">产物 {op.artifacts}</span>}
                  <OpStatusChip status={op.status} />
                  <span className="shrink-0 text-[9.5px] text-fg-faint">{fmtTime(op.created_at)}</span>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
