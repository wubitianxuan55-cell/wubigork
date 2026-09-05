import { useState } from "react";
import type { ChangeDiff } from "../lib/planDiff";
import { charSegments, foldContext, pairModifications, type DiffPresentRow } from "../lib/diffRender";
import { highlightLine } from "../lib/diffHighlight";
import "./changesdiff-tok.css";

// ChangesDiff —「变更」tab 的行级红绿 diff 渲染（v4.25 变更 tab diff 化）。
//
// Why: 对标 Git 面板的行级红绿 diff。数据源是写类工具的参数片段（planDiff.ts
// 构造），能构造 old→new 的显示红绿行；无 old/new 的写工具诚实降级为
// 「写入内容预览」（中性样式 + 降级原因说明），绝不伪造 diff。
//
// How: 复用 lib/diff.ts 的 DiffRow（ctx|add|del），配色走 styles.css 的
// --add/--del diff 令牌；行数超上限折叠并标注截断，避免大 diff 撑爆面板。
//
// v4.87「2c 统一 diff 渲染升级」：改蓝配对（相邻删块+增块按行配对成
// 「改动」对，蓝底替代红/绿）+ 行内字符高亮（配对行内变化片段加删除/
// 新增强调）+ 上下文折叠（连续 ctx 行中段收起可展开）。三个数据源
// （变更面板 LCS、Git 面板 unified diff、版本时间线 text/docx/xlsx 对比）
// 共用本查看器。语法着色经 path 参数（diffHighlight）。DiffRow.marker
// （docx 段落序号 / xlsx 单元格 ref）渲染为 +/- 列后的定宽右对齐暗色列，
// 配对与折叠占位都携带原行对象，marker 不丢；无 marker 的数据源零变化。

// 单 hunk 最大渲染行数：超出截断（LCS 全量行可能上万）。
const MAX_ROWS = 300;
// content 预览最大字符数（超出截断并标注）。
const MAX_CONTENT_CHARS = 4000;

// 配对行内片段着色：changed 片段加删除/新增强调，未变片段正常。
function Segments({ segs, side }: { segs: { text: string; changed: boolean }[]; side: "old" | "new" }) {
  return (
    <>
      {segs.map((s, i) =>
        s.changed ? (
          <span
            key={i}
            style={{
              background: side === "old" ? "var(--del-bg)" : "var(--add-bg)",
              color: side === "old" ? "var(--del-fg)" : "var(--add-fg)",
              borderRadius: 2,
            }}
          >
            {s.text}
          </span>
        ) : (
          <span key={i}>{s.text}</span>
        ),
      )}
    </>
  );
}

// 单 hunk 展示行（含配对蓝染与折叠占位）。
function PresentRows({ present, path }: { present: DiffPresentRow[]; path?: string }) {
  // 折叠块展开集合（展开后原行就地显示，不再收起；行键=序列下标）。
  const [unfolded, setUnfolded] = useState<Set<number>>(() => new Set());

  const shownAll = present.slice(0, MAX_ROWS);
  return (
    <div className="font-mono text-[10.5px] leading-[1.6] rounded-md overflow-hidden border border-border-soft">
      {shownAll.map((p, i) => {
        if (p.kind === "fold") {
          if (!unfolded.has(i)) {
            return (
              <button
                key={i}
                type="button"
                data-testid="diff-fold-toggle"
                className="w-full cursor-pointer border-0 bg-bg-soft/60 px-2 py-1 text-center text-[10px] text-fg-faint hover:text-fg"
                onClick={() => setUnfolded((cur) => new Set(cur).add(i))}
              >
                ⋯ 已折叠 {p.count} 行未改动上下文（点击展开）
              </button>
            );
          }
          // 展开态：原上下文行就地渲染（marker 列照常携带，docx 段号不丢）。
          return p.rows.map((r, k) => (
            <div key={`${i}-${k}`} className="flex whitespace-pre-wrap break-all" style={{ background: "transparent" }}>
              <span className="w-4 shrink-0 text-center select-none opacity-70"> </span>
              {r.marker !== undefined && (
                <span
                  className="w-9 shrink-0 select-none text-right tabular-nums opacity-60"
                  style={{ color: "var(--md-sys-color-text-secondary)" }}
                >
                  {r.marker}
                </span>
              )}
              <span className="flex-1 min-w-0 pr-2" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                {r.text === "" ? " " : r.text}
              </span>
            </div>
          ));
        }
        const r = p.row;
        const paired = p.pairOld !== undefined && p.pairNew !== undefined;
        // 配对成功（old/new 两行都在）→ 改蓝；未配对的独立增删保持红/绿。
        const bg = paired
          ? "color-mix(in srgb, var(--gaea-glow) 10%, transparent)"
          : r.type === "add"
            ? "var(--add-bg)"
            : r.type === "del"
              ? "var(--del-bg)"
              : "transparent";
        const fg = paired
          ? "var(--md-sys-color-text)"
          : r.type === "add"
            ? "var(--add-fg)"
            : r.type === "del"
              ? "var(--del-fg)"
              : "var(--md-sys-color-text-secondary)";
        return (
          <div
            key={i}
            data-pair={paired ? (r.type === "del" ? "old" : "new") : undefined}
            className="flex whitespace-pre-wrap break-all"
            style={{ background: bg }}
          >
            <span
              className="w-4 shrink-0 text-center select-none opacity-70"
              style={{ color: r.type === "add" ? "var(--add-fg)" : r.type === "del" ? "var(--del-fg)" : "inherit" }}
            >
              {r.type === "add" ? "+" : r.type === "del" ? "-" : " "}
            </span>
            {r.marker !== undefined && (
              <span
                className="w-9 shrink-0 select-none text-right tabular-nums opacity-60"
                style={{ color: "var(--md-sys-color-text-secondary)" }}
              >
                {r.marker}
              </span>
            )}
            <span className="flex-1 min-w-0 pr-2" style={{ color: fg }}>
              {paired && r.type === "del"
                ? <Segments segs={charSegments(p.pairOld!, p.pairNew!).oldSegs} side="old" />
                : paired && r.type === "add"
                  ? <Segments segs={charSegments(p.pairOld!, p.pairNew!).newSegs} side="new" />
                  : path
                    ? highlightLine(r.text, path).map((seg, k) =>
                        seg.cls
                          ? <span key={k} className={seg.cls}>{seg.text || " "}</span>
                          : <span key={k}>{seg.text === "" ? " " : seg.text}</span>,
                      )
                    : r.text === "" ? " " : r.text}
            </span>
          </div>
        );
      })}
      {present.length > MAX_ROWS && (
        <div className="px-2 py-1 text-[10px] text-fg-faint bg-bg-soft/60">
          已截断：共 {present.length} 行，仅展示前 {MAX_ROWS} 行（可在预览中查看完整文件）
        </div>
      )}
    </div>
  );
}

function DiffRows({ rows, path }: { rows: ChangeDiff["hunks"][number]["rows"]; path?: string }) {
  // 2c：配对 → 上下文折叠。
  const present = foldContext(pairModifications(rows));
  return <PresentRows present={present} path={path} />;
}

export function ChangesDiff({ diff, path }: { diff: ChangeDiff; path?: string }) {
  if (diff.kind === "diff") {
    return (
      <div className="flex flex-col gap-1.5">
        {diff.hunks.map((h, i) => (
          <div key={i} data-testid="changes-diff-hunk">
            {h.label && <div className="mb-0.5 text-[10px] text-fg-faint">{h.label}</div>}
            <DiffRows rows={h.rows} path={path} />
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
