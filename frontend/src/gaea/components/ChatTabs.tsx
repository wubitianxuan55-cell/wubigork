import { BarChart3, Gauge, ListTree, MessageSquare } from "../icons";

// 对话窗口上方的视图标签（dsh-context 移植，v4.17-v4.20 已完整接通）：
// [对话] 现有 Transcript；[轨迹] 事件账本（概览/搜索/折叠/增量渲染）；
// [上下文] 上下文构成看板（趋势/事件/文件活动/浏览器/Agent 网络）；
// [概览] 会话统计看板（v4.23 自右栏「分析组·统计」迁入，OverviewPanel 承载）。
export type ChatTabId = "chat" | "trajectory" | "context" | "overview";

/** 动态会话 tab（子代理独立会话，better-sidebar openSubagent 语义）。 */
export interface ChatSessionTab {
  id: string;
  label: string;
  /** 会话状态（子代理运行态；有值时 tab 前显示实时状态点，与主代理运行态同口径） */
  status?: "running" | "completed" | "failed";
  /** 悬停详情（任务/状态/模型等完整信息；缺省回退 id） */
  detail?: string;
}

const TABS: { id: ChatTabId; label: string; icon: typeof MessageSquare }[] = [
  { id: "chat", label: "对话", icon: MessageSquare },
  { id: "trajectory", label: "轨迹", icon: ListTree },
  { id: "context", label: "上下文", icon: BarChart3 },
  { id: "overview", label: "概览", icon: Gauge },
];

export function ChatTabs({ active, onChange, extraTabs, onCloseExtra }: {
  active: string;
  onChange: (id: string) => void;
  /** 独立会话 tab（如子代理会话），渲染在四视图之后。 */
  extraTabs?: ChatSessionTab[];
  /** 关闭动态会话 tab。 */
  onCloseExtra?: (id: string) => void;
}) {
  return (
    <div className="flex items-center gap-1 px-12 pt-2 pb-0 border-b border-border-soft bg-bg/80 select-none">
      {TABS.map((t) => {
        const Icon = t.icon;
        const selected = t.id === active;
        return (
          <button
            key={t.id}
            className={`relative flex items-center gap-1.5 px-3 py-1.5 text-[12px] rounded-t-md border-0 bg-transparent cursor-pointer transition-colors ${
              selected ? "text-accent" : "text-fg-dim hover:text-fg"
            }`}
            onClick={() => onChange(t.id)}
            title={t.id === "trajectory" ? "工具调用/步骤时间线" : t.id === "overview" ? "Token/成本/命中率统计" : undefined}
          >
            <Icon size={13} />
            {t.label}
            {selected && <span className="absolute inset-x-2 -bottom-px h-0.5 rounded-full bg-accent" />}
          </button>
        );
      })}
      {(extraTabs ?? []).map((s) => {
        const selected = s.id === active;
        return (
          <div
            key={s.id}
            role="tab"
            aria-selected={selected}
            tabIndex={0}
            data-chat-session-tab={s.id}
            className={`relative flex items-center gap-1.5 px-3 py-1.5 text-[12px] rounded-t-md border-0 cursor-pointer transition-colors ${
              selected ? "text-accent" : "text-fg-dim hover:text-fg"
            }`}
            onClick={() => onChange(s.id)}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                onChange(s.id);
              }
            }}
            title={s.detail || s.id}
          >
            {s.status && (
              <span
                data-testid={`chat-tab-status-${s.id}`}
                className="inline-block h-1.5 w-1.5 rounded-full shrink-0"
                style={{
                  background:
                    s.status === "running"
                      ? "var(--gaea-glow)"
                      : s.status === "failed"
                        ? "var(--md-sys-color-destructive)"
                        : "var(--md-sys-color-success)",
                }}
                aria-hidden
              />
            )}
            <MessageSquare size={13} />
            <span className="max-w-[140px] truncate">{s.label}</span>
            {onCloseExtra && (
              <button
                type="button"
                aria-label={`关闭 ${s.label}`}
                className="flex items-center justify-center w-4 h-4 rounded text-fg-faint hover:text-fg hover:bg-bg-soft"
                onClick={(e) => {
                  e.stopPropagation();
                  onCloseExtra(s.id);
                }}
              >
                <svg width="8" height="8" viewBox="0 0 10 10" aria-hidden>
                  <path d="M1 1l8 8M9 1L1 9" stroke="currentColor" strokeWidth="1.4" />
                </svg>
              </button>
            )}
            {selected && <span className="absolute inset-x-2 -bottom-px h-0.5 rounded-full bg-accent" />}
          </div>
        );
      })}
    </div>
  );
}
