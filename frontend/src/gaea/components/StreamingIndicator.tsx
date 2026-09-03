import { useEffect, useRef, useState } from "react";
import { t } from "../lib/i18n";
import type { DictKey } from "../locales/en";

// StreamingIndicator — 对话窗最底兜底的连接状态条（v4.26 重定）。
//
// Why：v4.26 工作态头部行（WorkHeader）已接管主反馈（阶段文本 + 已用时 +
// 步数，items 为空也渲染），本组件降级为「首个事件都没到」的最底兜底——
// 原「准备中→生成中→工具执行中→仍在处理…」多阶段文案与头部信息重复，
// 收敛为两档：连接中（≤5s）/ 仍在等待事件（>5s，附「可切轨迹面板查看」
// 提示，指引用户去 TrajectoryView 深度层排查，而不是干等）。
//
// How：running 时启动 5s 定时器，超时未收到任何反馈即升级 waiting 档；
// 保留 sticky 顶条布局（占位防虚拟列表抖动）与语义色点。

type Stage = "idle" | "connecting" | "waiting";

const WAIT_THRESHOLD_MS = 5_000;

const stageConfig: Record<Stage, { labelKey?: DictKey; dotClass: string; glowClass: string; textClass: string; barClass: string }> = {
  idle:       { labelKey: undefined,           dotClass: "",                        glowClass: "",                       textClass: "",        barClass: "" },
  connecting: { labelKey: "stream.connecting", dotClass: "bg-info",                 glowClass: "shadow-[0_0_6px_color-mix(in_srgb,var(--info)_60%,transparent)]", textClass: "text-info", barClass: "bg-info w-[45%]" },
  waiting:    { labelKey: "stream.waiting",    dotClass: "bg-warning animate-pulse", glowClass: "shadow-[0_0_6px_color-mix(in_srgb,var(--warn)_60%,transparent)]", textClass: "text-warning", barClass: "bg-warning/70 w-[30%]" },
};

export function StreamingIndicator({
  running,
}: {
  running: boolean;
}) {
  const [stage, setStage] = useState<Stage>("idle");
  const waitTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (!running) {
      setStage("idle");
      if (waitTimer.current) clearTimeout(waitTimer.current);
      return;
    }
    setStage("connecting");
    if (waitTimer.current) clearTimeout(waitTimer.current);
    waitTimer.current = setTimeout(() => {
      setStage((s) => (s === "connecting" ? "waiting" : s));
    }, WAIT_THRESHOLD_MS);
    return () => {
      if (waitTimer.current) clearTimeout(waitTimer.current);
    };
  }, [running]);

  const hidden = !running || stage === "idle";
  const cfg = stageConfig[hidden ? "idle" : stage];

  return (
    <div className={`sticky top-0 z-10 flex items-center gap-2.5 px-3 py-2 border-b border-border-soft/60 bg-bg-soft/60 backdrop-blur-md ${hidden ? "invisible" : ""}`}>
      {/* 滚动色条 — 高 3px，顶对齐 */}
      <div className="absolute top-0 left-0 right-0 h-[3px] bg-border-soft/60 overflow-hidden">
        <div className={`h-full rounded-r-sm animate-pulse ${cfg.barClass}`} />
      </div>

      <span className={`w-2 h-2 rounded-full shrink-0 ${cfg.dotClass} ${cfg.glowClass}`} />
      <span className={`text-[12px] font-medium ${cfg.textClass}`}>{cfg.labelKey ? t(cfg.labelKey) : ""}</span>

      {stage === "waiting" && (
        <span className="text-fg-faint text-[11px] ml-auto truncate">{t("stream.trajectoryHint")}</span>
      )}
    </div>
  );
}
