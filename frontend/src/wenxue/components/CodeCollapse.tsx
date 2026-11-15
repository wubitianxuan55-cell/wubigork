import { useMemo, useState, type ReactNode } from "react";

// 超长代码块折叠（v4.234，对齐 Claude/ChatGPT 长代码处理）：超过阈值行数
// 默认收起（max-height 裁剪+底部渐隐+展开钮），避免几十上百行的代码糊满
// 对话流；短代码零包装直通。办公板块与聊天线共用——阈值与折后高度同一
// 常量，两处观感一致。
//
// 主题适配说明：本组件同时活在 gaea chunk 与聊天 chunk（后者加载不到
// gaea/styles.css 的 --bg-* 令牌），故渐隐色由调用方显式传（fade），
// 按钮用全局 M3 令牌（App.tsx 主题 effect 写 :root）内联样式；文案由
// 调用方注入 label（gaea 走 useT 三语，聊天线与其既有硬编码中文一致）。
export const CODE_COLLAPSE_LINES = 30;
const COLLAPSED_MAX_H = 380;

function isLongCode(text: string): boolean {
  return text.split("\n").length > CODE_COLLAPSE_LINES;
}

export interface CodeCollapseLabels {
  expand: (lines: number) => string;
  collapse: string;
}

const ZH_LABELS: CodeCollapseLabels = { expand: (n) => `展开全部（${n} 行）`, collapse: "收起" };

export function CodeCollapse({
  text,
  fade = "var(--bg, #1B2336)",
  labels = ZH_LABELS,
  children,
}: {
  text: string;
  /** 收起态底部渐隐的目标色=该代码面板的背景色。 */
  fade?: string;
  labels?: CodeCollapseLabels;
  children: ReactNode;
}) {
  const long = useMemo(() => isLongCode(text), [text]);
  const lines = useMemo(() => text.split("\n").length, [text]);
  const [open, setOpen] = useState(false);
  if (!long) return <>{children}</>;
  return (
    <div className="relative">
      <div
        className={`overflow-x-auto ${open ? "" : "overflow-y-hidden"}`}
        style={open ? undefined : { maxHeight: COLLAPSED_MAX_H }}
      >
        {children}
      </div>
      {!open && (
        <div
          aria-hidden
          className="pointer-events-none absolute inset-x-0 bottom-0 h-16"
          style={{ background: `linear-gradient(to bottom, transparent, ${fade})` }}
        />
      )}
      <button
        type="button"
        className="absolute inset-x-0 bottom-0 flex justify-center pb-2 border-0 bg-transparent cursor-pointer"
        onClick={() => setOpen((v) => !v)}
      >
        <span
          style={{
            display: "inline-flex", alignItems: "center", padding: "3px 12px", borderRadius: 999,
            border: "1px solid var(--md-sys-color-outline-variant)",
            background: "var(--md-sys-color-surface-container-high, #1E293B)",
            color: "var(--md-sys-color-on-surface-variant, #CBD5E1)", fontSize: 10.5,
            boxShadow: "0 1px 4px rgba(0,0,0,0.25)",
          }}
        >
          {open ? labels.collapse : labels.expand(lines)}
        </span>
      </button>
    </div>
  );
}
