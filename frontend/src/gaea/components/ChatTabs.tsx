import { BarChart3, ListTree, MessageSquare } from "../icons";

// 对话窗口上方的视图标签（dsh-context 移植 Phase A）：
// [对话] 现有 Transcript；[轨迹] 工具调用/步骤时间线（暂占位）；
// [上下文] 上下文构成看板。选中态蓝色下划线，对齐效果图。
export type ChatTabId = "chat" | "trajectory" | "context";

const TABS: { id: ChatTabId; label: string; icon: typeof MessageSquare }[] = [
  { id: "chat", label: "对话", icon: MessageSquare },
  { id: "trajectory", label: "轨迹", icon: ListTree },
  { id: "context", label: "上下文", icon: BarChart3 },
];

export function ChatTabs({ active, onChange }: {
  active: ChatTabId;
  onChange: (tab: ChatTabId) => void;
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
            title={t.id === "trajectory" ? "工具调用/步骤时间线" : undefined}
          >
            <Icon size={13} />
            {t.label}
            {selected && <span className="absolute inset-x-2 -bottom-px h-0.5 rounded-full bg-accent" />}
          </button>
        );
      })}
    </div>
  );
}
