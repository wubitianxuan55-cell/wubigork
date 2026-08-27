import { useEffect, useRef, useState } from "react";
import type { Item } from "../lib/store";

type Stage = "idle" | "preparing" | "streaming" | "tool_exec" | "stalled";

const stageConfig: Record<Stage, { label: string; barClass: string; dotClass: string; glowClass: string; textClass: string }> = {
  idle:      { label: "",              barClass: "",                    dotClass: "",                         glowClass: "",                       textClass: "" },
  preparing: { label: "准备中",         barClass: "bg-warning/60",       dotClass: "bg-warning animate-pulse", glowClass: "shadow-[0_0_6px_color-mix(in_srgb,var(--warn)_60%,transparent)]", textClass: "text-warning" },
  streaming: { label: "生成中",         barClass: "bg-info",             dotClass: "bg-info",                  glowClass: "shadow-[0_0_6px_color-mix(in_srgb,var(--info)_60%,transparent)]", textClass: "text-info" },
  tool_exec: { label: "工具执行中…",    barClass: "bg-accent/60",        dotClass: "bg-accent animate-pulse",  glowClass: "shadow-[0_0_6px_color-mix(in_srgb,var(--accent)_60%,transparent)]", textClass: "text-accent" },
  stalled:   { label: "仍在处理…",      barClass: "bg-err/50",           dotClass: "bg-err animate-pulse",     glowClass: "shadow-[0_0_6px_color-mix(in_srgb,var(--err)_55%,transparent)]", textClass: "text-err" },
};

function hasRunningTools(items: Item[]): boolean {
  for (let i = items.length - 1; i >= 0; i--) {
    const it = items[i];
    if (it.kind === "tool" && it.status === "running") return true;
    if (it.kind === "assistant") break; // past the current turn's tools
  }
  return false;
}

function toolCount(items: Item[]): number {
  let n = 0;
  for (let i = items.length - 1; i >= 0; i--) {
    const it = items[i];
    if (it.kind === "tool") n++;
    if (it.kind === "assistant" || it.kind === "user") break;
  }
  return n;
}

/**
 * StreamingIndicator renders a compact "preparing → streaming → tool_exec → stalled"
 * status bar inside the transcript while the model is generating a response.
 * Always occupies layout space (via visibility) to prevent virtual-list jitter.
 */
export function StreamingIndicator({
  running,
  items,
}: {
  running: boolean;
  items: Item[];
}) {
  const last = items[items.length - 1];
  const isStreaming = last?.kind === "assistant" && last.streaming;
  const [stage, setStage] = useState<Stage>("idle");
  const stallTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (!running) {
      setStage("idle");
      if (stallTimer.current) clearTimeout(stallTimer.current);
      return;
    }
    if (isStreaming) {
      setStage("streaming");
      if (stallTimer.current) clearTimeout(stallTimer.current);
      return;
    }
    // Detect tool execution: model emitted tool calls, tools are running.
    if (hasRunningTools(items)) {
      setStage("tool_exec");
      if (stallTimer.current) clearTimeout(stallTimer.current);
      return;
    }
    setStage("preparing");
    if (stallTimer.current) clearTimeout(stallTimer.current);
    stallTimer.current = setTimeout(() => {
      setStage((s) => (s === "preparing" ? "stalled" : s));
    }, 15_000);
    return () => {
      if (stallTimer.current) clearTimeout(stallTimer.current);
    };
  }, [running, isStreaming, items]);

  const hidden = !running || stage === "idle";
  const cfg = stageConfig[hidden ? "idle" : stage];
  const tc = toolCount(items);

  return (
    <div className={`sticky top-0 z-10 flex items-center gap-2.5 px-3 py-2 border-b border-border-soft/60 bg-bg-soft/60 backdrop-blur-md ${hidden ? "invisible" : ""}`}>
      {/* 滚动色条 — 高 3px，顶对齐 */}
      <div className="absolute top-0 left-0 right-0 h-[3px] bg-border-soft/60 overflow-hidden">
        <div className={`h-full rounded-r-sm animate-pulse ${
          stage === "preparing" ? "bg-warning w-1/4" :
          stage === "tool_exec" ? "bg-accent w-[50%]" :
          stage === "stalled" ? "bg-err w-[40%]" :
          "bg-info w-[60%]"
        }`} />
      </div>

      <span className={`w-2 h-2 rounded-full shrink-0 ${cfg.dotClass} ${cfg.glowClass}`} />
      <span className={`text-[12px] font-medium ${cfg.textClass}`}>{cfg.label}</span>

      {stage === "tool_exec" && tc > 0 && (
        <span className="text-fg-faint text-[11px] ml-auto tabular-nums">{tc} 个工具</span>
      )}
      {stage === "preparing" && (
        <span className="text-fg-faint text-[11px] ml-auto tabular-nums">15"</span>
      )}
    </div>
  );
}
