import type { ChangeDiff } from "../lib/planDiff";

// ChangesDiff —「变更」tab 的行级红绿 diff 渲染（v4.25 变更 tab diff 化）。
//
// Why: 对标 Git 面板的行级红绿 diff。数据源是写类工具的参数片段（planDiff.ts
// 构造），能构造 old→new 的显示红绿行；无 old/new 的写工具诚实降级为
// 「写入内容预览」（中性样式 + 降级原因说明），绝不伪造 diff。
//
// How: 复用 lib/diff.ts 的 DiffRow（ctx|add|del），配色走 styles.css 的
// --add/--del diff 令牌；行数超上限折叠并标注截断，避免大 diff 撑爆面板。

// 单 hunk 最大渲染行数：超出截断（LCS 全量行可能上万）。
const MAX_ROWS = 300;
// content 预览最大字符数（超出截断并标注）。
const MAX_CONTENT_CHARS = 4000;

function DiffRows({ rows }: { rows: ChangeDiff["hunks"][number]["rows"] }) {
  const shown = rows.slice(0, MAX_ROWS);
  return (
    <div className="font-mono text-[10.5px] leading-[1.6] rounded-md overflow-hidden border border-border-soft">
      {shown.map((r, i) => (
        <div
          key={i}
          className="flex whitespace-pre-wrap break-all"
          style={{
            background: r.type === "add" ? "var(--add-bg)" : r.type === "del" ? "var(--del-bg)" : "transparent",
          }}
        >
          <span
            className="w-4 shrink-0 text-center select-none opacity-70"
            style={{ color: r.type === "add" ? "var(--add-fg)" : r.type === "del" ? "var(--del-fg)" : "inherit" }}
          >
            {r.type === "add" ? "+" : r.type === "del" ? "-" : " "}
          </span>
          <span
            className="flex-1 min-w-0 pr-2"
            style={{
              color: r.type === "add" ? "var(--add-fg)" : r.type === "del" ? "var(--del-fg)" : "var(--md-sys-color-text-secondary)",
            }}
          >
            {r.text === "" ? " " : r.text}
          </span>
        </div>
      ))}
      {rows.length > MAX_ROWS && (
        <div className="px-2 py-1 text-[10px] text-fg-faint bg-bg-soft/60">
          已截断：共 {rows.length} 行，仅展示前 {MAX_ROWS} 行（可在预览中查看完整文件）
        </div>
      )}
    </div>
  );
}

export function ChangesDiff({ diff }: { diff: ChangeDiff }) {
  if (diff.kind === "diff") {
    return (
      <div className="flex flex-col gap-1.5">
        {diff.hunks.map((h, i) => (
          <div key={i} data-testid="changes-diff-hunk">
            {h.label && <div className="mb-0.5 text-[10px] text-fg-faint">{h.label}</div>}
            <DiffRows rows={h.rows} />
          </div>
        ))}
      </div>
    );
  }
  if (diff.kind === "content") {
    // 诚实降级：无 old/new，只展示新写入内容（中性样式，不用红绿行冒充 diff）。
    const truncated = (diff.content?.length ?? 0) > MAX_CONTENT_CHARS;
    return (
      <div className="flex flex-col gap-1" data-testid="changes-content-preview">
        <div className="text-[10px] leading-snug text-fg-faint">{diff.note}</div>
        <pre className="m-0 max-h-40 overflow-auto font-mono text-[10.5px] leading-[1.6] whitespace-pre-wrap break-all rounded-md border border-border-soft bg-bg-soft/40 px-2 py-1.5 text-fg-dim">
          {truncated ? `${(diff.content ?? "").slice(0, MAX_CONTENT_CHARS)}…` : diff.content}
        </pre>
        {truncated && (
          <div className="text-[10px] text-fg-faint">内容过长已截断（共 {diff.content?.length} 字符）</div>
        )}
      </div>
    );
  }
  return (
    <div className="text-[10.5px] leading-snug text-fg-faint" data-testid="changes-diff-none">
      {diff.note}
    </div>
  );
}
