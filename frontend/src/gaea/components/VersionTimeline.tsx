// VersionTimeline — v4.28 B1「文件版本时间线」（对标 Notion 版本史 / Claude
// Artifacts 版本选择器）。
// Why：产物行的 vN 徽标此前只给「改了几次」的次数，用户既看不到每次改了什么，
// 也无法在改坏时退回上一版；证据链（JournalList）已为每次写盘留了基线快照，
// 把它按文件聚合成逐版本列表，「预览 → 恢复」才有落点。
// How：本组件是纯展示层——版本记录（调用方经 groupVersionsByPath 聚合、at
// 倒序）、预览回调（openFilePreview(baselinePath)：基线快照位于工作区内，
// Preview 的 resolvePreviewPath IsAbs 分支可直读）、恢复回调（父级负责
// RollbackRecord(id) + toast + 刷新时间线）全部经 props 注入，本组件不直接
// 触碰 bridge。每行 = 时间/工具/轮次/状态徽标 + 对比/预览/恢复三个操作；恢复语义
// 对齐调研结论（docs/research-2026-09-01/version-timeline-diff.md §4）：
// 不做二次确认弹窗（预览即护栏），顶部常驻说明「恢复会把该文件写回所选版本，
// 当前内容成为新版本」；恢复进行中禁用全部恢复按钮，避免并发写盘竞态。
//
// v4.28 A1「与当前对比」：每行新增对比动作，点击在该行下方展开内联对比区——
// 语义固定为「该基线快照 vs 当前工作区文件」（不做多版本两两对比，后续刀），
// 数据层经 lib/versionCompare.compareVersionWithCurrent(baselinePath, target)
// 取数（本组件唯一触碰 bridge 的口子，仅在展开时发起）。渲染：diffstat 芯片
// （+N −N 绿/红）+ 行级红绿 diff（复用 --add/--del diff 令牌，等宽字体，
// max-h-80 有界滚动）；超过 MAX_DIFF_ROWS 行折叠 + 展开全部开关；unsupported /
// contentMissing / 无差异分别走 vcompare.* 字典降级提示。竞态防护：cmpSeq 单调
// 递增，只有最新一次请求的返回可写回状态（连点不同版本行，旧结果丢弃）；再点
// 同一行（或点面板「收起对比」）折叠并取消挂载对比内容。
//
// A2 结构化对比（v4.28 记账）+ v4.87 统一 diff 查看器收口：text/docx/xlsx
// 三种对比体统一经 ChangesDiff 渲染（改蓝配对 + 字符级高亮 + ctx 折叠 +
// path 语法着色全套生效）——text/docx 构造 { kind:"diff", hunks:[{ rows }] }
// （docx 段落序号走 DiffRow.marker 列）；xlsx 每 sheet 一个 hunk（label =
// sheet 名 + 状态/截断文案，change 单元格 = 相邻 del+add 对，marker = 单元格
// ref，formula 追加 fx 后缀），诚实原则：不伪造 ctx 行、不补未变单元格。
// clampDiffRows(200) + 展开全部开关保留在本组件（ChangesDiff 下方）。
import { useCallback, useRef, useState } from "react";
import { Clock, Diff, Eye, Loader2, Rollback } from "../icons";
import { ChangesDiff } from "./ChangesDiff";
import { useT } from "../lib/i18n";
import type { DiffRow } from "../lib/diff";
import type { JournalChangeRecord } from "../lib/types";
import { versionStatusText, versionTimeText } from "../lib/versionTimeline";
import {
  clampDiffRows,
  compareVersionWithCurrent,
  type VersionCompareResult,
  type VersionXlsxDiff,
} from "../lib/versionCompare";
import type { DocxRow } from "../lib/docxTextDiff";

export interface VersionTimelineProps {
  /** 目标文件（工作区相对路径）——标题与 data-path 定位用，不做数据过滤。 */
  path: string;
  /** 该路径的版本记录（已聚合、at 倒序、只含有 baselinePath 的卡）；null = 加载中。 */
  records: JournalChangeRecord[] | null;
  /** 预览该版本：传入基线快照路径（工作区内相对/绝对路径均可被 Preview 直读）。 */
  onPreview: (baselinePath: string) => void;
  /** 恢复到该版本：父级负责 RollbackRecord(record.id) + 成功 toast + 刷新时间线。 */
  onRestore: (record: JournalChangeRecord) => void | Promise<void>;
}

// 状态徽标配色：verified/applied 成功 / warned 警告 / failed 危险 / 其余中性。
function statusColor(status: string): string {
  switch (status) {
    case "verified":
    case "applied":
      return "var(--md-sys-color-success)";
    case "warned":
      return "var(--md-sys-color-warning)";
    case "failed":
      return "var(--md-sys-color-destructive)";
    default:
      return "var(--md-sys-color-text-secondary)";
  }
}

// 小图标操作按钮：与 DeliverablesPanel 同款令牌化样式（可见焦点环走全局 :focus-visible）。
const iconBtn =
  "flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-(color:--md-sys-color-text-secondary) cursor-pointer hover:text-(color:--md-sys-color-text) hover:bg-(color:--md-sys-color-surface-container-high) transition-colors";

// 单次 diff 最大渲染行数：超过折叠为前 N 行 + 「展开全部」开关（复用既有字典）。
const MAX_DIFF_ROWS = 200;

// Codex 式 diffstat 芯片：+N（绿）−N（红），无差异时不渲染。
function DiffStatChip({ add, del }: { add: number; del: number }) {
  if (add === 0 && del === 0) return null;
  return (
    <span
      data-testid="vcompare-stat"
      className="inline-flex shrink-0 items-center gap-1 rounded border px-1 py-px font-mono text-[9.5px] leading-none tabular-nums"
      style={{
        borderColor: "color-mix(in srgb, var(--md-sys-color-outline-variant) 60%, transparent)",
        background: "color-mix(in srgb, var(--md-sys-color-surface-container-high) 40%, transparent)",
      }}
    >
      <span style={{ color: "var(--md-sys-color-success)" }}>+{add}</span>
      <span style={{ color: "var(--md-sys-color-destructive)" }}>−{del}</span>
    </span>
  );
}

// xlsx 单个 sheet 的变更列表 → DiffRow[]（诚实原则：xlsxCellDiff 只产变更
// 单元格，不补未变行/不伪造 ctx）。change = 相邻 del+add 对（ChangesDiff 的
// 改蓝配对自动给出「改蓝 + 行内字符高亮」）；marker 列 = 单元格 ref；formula
// 非空在 new 值后追加「  fx: =公式」后缀；整表级 add/del（无 cells）生成单行
// （text=sheet 名，无 marker）。
function xlsxSheetRows(s: VersionXlsxDiff["sheets"][number]): DiffRow[] {
  if (s.cells.length === 0) {
    if (s.state === "add") return [{ type: "add", text: s.name }];
    if (s.state === "del") return [{ type: "del", text: s.name }];
    return [];
  }
  const rows: DiffRow[] = [];
  for (const c of s.cells) {
    const fx = c.formula ? `  fx: =${c.formula}` : "";
    if (c.kind === "change") {
      rows.push({ type: "del", marker: c.ref, text: c.old });
      rows.push({ type: "add", marker: c.ref, text: `${c.new}${fx}` });
    } else if (c.kind === "add") {
      rows.push({ type: "add", marker: c.ref, text: `${c.new}${fx}` });
    } else {
      rows.push({ type: "del", marker: c.ref, text: c.old });
    }
  }
  return rows;
}

// xlsx 对比体（统一 diff 查看器）：每个 sheet 一个 hunk 经 ChangesDiff 渲染——
// hunk label = sheet 名 + 状态文案（sheetAdd/sheetDel/sheetChanged），截断时拼
// cellTruncated（数据层已按 MAX_XLSX_SHEET_CELLS 截断、计数不失真，label 如实
// 标注，不提供展开全部——数据已截，展开无从恢复，与 text/docx 的 UI 层折叠不同）。
function XlsxCompareBody({ result, path }: { result: VersionXlsxDiff; path?: string }) {
  const t = useT();
  if (result.sheets.length === 0) {
    return (
      <div data-testid="vcompare-empty" className="py-0.5 text-[10px] leading-relaxed" style={{ color: "var(--md-sys-color-text-secondary)" }}>
        {t("vcompare.empty")}
      </div>
    );
  }
  return (
    <div className="flex flex-col gap-1">
      {result.contentMissing && (
        <div data-testid="vcompare-content-missing" className="text-[9.5px] leading-relaxed" style={{ color: "var(--md-sys-color-warning)" }}>
          {t("vcompare.contentMissing")}
        </div>
      )}
      {result.sheets.map((s, i) => {
        const stateText =
          s.state === "add"
            ? t("vcompare.sheetAdd")
            : s.state === "del"
              ? t("vcompare.sheetDel")
              : t("vcompare.sheetChanged", { n: s.total });
        const label = [
          s.name,
          stateText,
          s.truncated ? t("vcompare.cellTruncated", { n: s.cells.length }) : "",
        ]
          .filter(Boolean)
          .join(" · ");
        return (
          <div
            key={`${s.name}-${i}`}
            data-testid={`vcompare-sheet-${i}`}
            className="max-h-60 overflow-auto rounded-md"
          >
            <ChangesDiff diff={{ kind: "diff", hunks: [{ label, rows: xlsxSheetRows(s) }] }} path={path} />
          </div>
        );
      })}
    </div>
  );
}

// 内联对比区：该基线快照 vs 当前工作区文件。result === null 表示取数进行中
// （spinner）；text/docx 构造 { kind:"diff", hunks:[{ rows }] } 交 ChangesDiff
// 统一渲染（docx 段落序号进 DiffRow.marker 列），xlsx 走结构化对比体
// XlsxCompareBody；行配色/配对/折叠与「变更」「Git」面板同源（ChangesDiff）。
function VersionComparePanel({
  label,
  result,
  showAll,
  onToggleAll,
  onHide,
  path,
}: {
  label: string;
  result: VersionCompareResult | null;
  showAll: boolean;
  onToggleAll: () => void;
  onHide: () => void;
  /** 目标文件路径：透传给 ChangesDiff 走 diffHighlight 语法着色（文本类生效）。 */
  path?: string;
}) {
  const t = useT();
  const rowResult =
    result?.kind === "text" ? result.rows : result?.kind === "docx" ? result.rows : null;
  const isDocx = result?.kind === "docx";
  const clamped = rowResult
    ? clampDiffRows(rowResult, showAll ? Number.MAX_SAFE_INTEGER : MAX_DIFF_ROWS)
    : null;
  // 行模型归一为 DiffRow（docx 段落序号 → marker 列），交 ChangesDiff 统一渲染；
  // clampDiffRows(200) + 展开全部开关保留在本组件（UI 层折叠逻辑不变）。
  const diffRows: DiffRow[] | null = clamped
    ? clamped.shown.map((r) =>
        isDocx
          ? { type: r.type, text: r.text, marker: String((r as DocxRow).index) }
          : r,
      )
    : null;
  // 仅当全量行数超上限时出现折叠开关（展开后切换为「收起」）。
  const overLimit = rowResult !== null && rowResult.length > MAX_DIFF_ROWS;
  return (
    <div
      data-testid="vcompare-panel"
      className="mb-0.5 mt-1 flex flex-col gap-1 rounded-md border px-1.5 py-1"
      style={{
        borderColor: "color-mix(in srgb, var(--md-sys-color-outline-variant) 60%, transparent)",
        background: "color-mix(in srgb, var(--md-sys-color-surface-container-low) 50%, transparent)",
      }}
    >
      {/* 标题行：对比标题 + diffstat 芯片 + 收起对比 */}
      <div className="flex items-center gap-1.5">
        <Diff size={11} aria-hidden style={{ color: "var(--gaea-glow)" }} />
        <span className="shrink-0 text-[9.5px] font-medium" style={{ color: "var(--md-sys-color-text)" }}>
          {t("vcompare.title", { label })}
        </span>
        {(result?.kind === "text" || result?.kind === "docx" || result?.kind === "xlsx") && (
          <DiffStatChip add={result.add} del={result.del} />
        )}
        <span className="min-w-0 flex-1" />
        <button
          type="button"
          data-testid="vcompare-hide"
          className="shrink-0 cursor-pointer rounded border-0 bg-transparent px-1 py-px text-[9.5px] text-(color:--md-sys-color-text-secondary) hover:text-(color:--md-sys-color-text)"
          onClick={onHide}
        >
          {t("vcompare.hide")}
        </button>
      </div>
      {result === null ? (
        <div data-testid="vcompare-loading" className="flex items-center gap-1.5 py-1 text-[10px]">
          <Loader2 size={11} className="animate-spin" />
        </div>
      ) : result.kind === "unsupported" ? (
        <div
          data-testid="vcompare-unsupported"
          className="text-[10px] leading-relaxed"
          style={{ color: "var(--md-sys-color-text-secondary)" }}
        >
          {t("vcompare.unsupported")}
        </div>
      ) : result.kind === "xlsx" ? (
        <XlsxCompareBody result={result} />
      ) : (
        <div className="flex flex-col gap-1">
          {/* 基线/当前任一侧内容不可用：顶部提示（结果仍展示，宁漏勿误口径） */}
          {result.contentMissing && (
            <div
              data-testid="vcompare-content-missing"
              className="text-[9.5px] leading-relaxed"
              style={{ color: "var(--md-sys-color-warning)" }}
            >
              {t("vcompare.contentMissing")}
            </div>
          )}
          {result.add === 0 && result.del === 0 ? (
            <div
              data-testid="vcompare-empty"
              className="py-0.5 text-[10px] leading-relaxed"
              style={{ color: "var(--md-sys-color-text-secondary)" }}
            >
              {t("vcompare.empty")}
            </div>
          ) : (
            <div data-testid="vcompare-diff" className="max-h-80 overflow-auto rounded-md">
              {/* 统一 diff 查看器：改蓝配对 / 字符级高亮 / ctx 折叠 / 语法着色 */}
              <ChangesDiff diff={{ kind: "diff", hunks: [{ rows: diffRows ?? [] }] }} path={path} />
              {overLimit && (
                <button
                  type="button"
                  data-testid="vcompare-expand"
                  className="w-full cursor-pointer border-0 bg-transparent px-2 py-1 text-left font-sans text-[10px] text-(color:--md-sys-color-text-secondary) hover:text-(color:--md-sys-color-text) hover:bg-(color:--md-sys-color-surface-container-high)"
                  onClick={onToggleAll}
                >
                  {showAll ? t("common.collapse") : t("tool.expandAllLines", { n: rowResult!.length })}
                </button>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export function VersionTimeline({ path, records, onPreview, onRestore }: VersionTimelineProps) {
  const t = useT();
  // 恢复进行中的卡 id：期间禁用所有「恢复」按钮（避免并发写盘），本行转圈。
  const [restoringId, setRestoringId] = useState<string | null>(null);
  // 对比区状态：展开行的 id + 该行取数结果（result=null 表示请求进行中）。
  // 同一时刻至多展开一行：切换行即替换，收起即置 null（对比内容随之取消挂载）。
  const [cmp, setCmp] = useState<{
    id: string;
    baseline: string;
    target: string;
    result: VersionCompareResult | null;
  } | null>(null);
  // 长 diff 折叠开关（展开全部）；切行/收起时复位。
  const [showAllRows, setShowAllRows] = useState(false);
  // 竞态防护：请求序号单调递增，只有最新一次请求的返回可写回状态——
  // 连续点击不同版本行时，旧请求的慢返回会被序号淘汰，不覆盖新状态。
  const cmpSeq = useRef(0);

  const handleRestore = useCallback((r: JournalChangeRecord) => {
    if (restoringId) return;
    setRestoringId(r.id);
    void Promise.resolve(onRestore(r)).finally(() => {
      setRestoringId((cur) => (cur === r.id ? null : cur));
    });
  }, [onRestore, restoringId]);

  // 对比开关：再点同一行 = 收起（不重新取数）；展开新行 = 置 loading 并发起
  // compareVersionWithCurrent(baselinePath, target)——语义固定「基线快照 vs 当前」。
  const handleCompare = useCallback(
    (r: JournalChangeRecord) => {
      const baseline = r.baselinePath ?? "";
      const opening = cmp?.id !== r.id;
      setShowAllRows(false);
      setCmp(opening ? { id: r.id, baseline, target: r.target, result: null } : null);
      cmpSeq.current++; // 收起时也推进序号，令在途旧结果失效
      if (!opening) return;
      const seq = cmpSeq.current;
      void compareVersionWithCurrent(baseline, r.target).then((res) => {
        if (cmpSeq.current !== seq) return; // 已有更新的一次请求/收起，丢弃
        setCmp((c) => (c && c.id === r.id ? { ...c, result: res } : c));
      });
    },
    [cmp],
  );

  const collapseCompare = useCallback(() => {
    cmpSeq.current++;
    setCmp(null);
    setShowAllRows(false);
  }, []);

  return (
    <div
      data-testid="version-timeline"
      data-path={path}
      className="mx-1 mb-1.5 flex flex-col gap-1 rounded-lg border px-2 py-1.5"
      style={{
        borderColor: "color-mix(in srgb, var(--md-sys-color-outline-variant) 60%, transparent)",
        background: "color-mix(in srgb, var(--gaea-glow) 4%, transparent)",
        color: "var(--md-sys-color-text-secondary)",
      }}
    >
      {/* 标题行：文件名 + 快照计数 */}
      <div className="flex items-center gap-1 text-[10px]">
        <Clock size={11} aria-hidden style={{ color: "var(--gaea-glow)" }} />
        <span className="shrink-0 font-medium" style={{ color: "var(--md-sys-color-text)" }}>版本时间线</span>
        <span className="min-w-0 flex-1 truncate font-mono text-[9px]" title={path}>{path}</span>
        {records && records.length > 0 && (
          <span
            className="shrink-0 rounded px-1 py-px font-mono text-[9px]"
            style={{
              color: "var(--gaea-glow)",
              background: "color-mix(in srgb, var(--gaea-glow) 10%, transparent)",
            }}
          >
            {records.length}
          </span>
        )}
      </div>
      {/* 恢复语义常驻说明：预览即护栏，不做二次确认弹窗 */}
      <div className="text-[9px] leading-relaxed" style={{ color: "var(--md-sys-color-warning)" }}>
        恢复会把该文件写回所选版本，当前内容成为新版本
      </div>
      {records === null ? (
        <div data-testid="version-timeline-loading" className="flex items-center gap-1.5 py-1.5 text-[10px]">
          <Loader2 size={11} className="animate-spin" />
          正在加载版本记录…
        </div>
      ) : records.length === 0 ? (
        <div data-testid="version-timeline-empty" className="py-1.5 text-center text-[10px] leading-relaxed">
          暂无可回滚的版本快照
          <br />
          该文件的历史变更未保留基线文件
        </div>
      ) : (
        <ol className="flex flex-col gap-0.5">
          {records.map((r) => {
            const restoring = restoringId === r.id;
            const cmpOpen = cmp?.id === r.id;
            return (
              <li
                key={r.id}
                data-testid="version-timeline-row"
                className="flex flex-col rounded-md px-1 py-0.5"
                title={r.afterSummary || r.beforeSummary}
              >
                <div className="flex items-center gap-1.5 hover:bg-(color:--md-sys-color-surface-container-high)">
                  <span className="shrink-0 font-mono text-[9px] tabular-nums">{versionTimeText(r)}</span>
                  <span
                    className="shrink-0 rounded px-1 py-px font-mono text-[9px]"
                    style={{
                      color: "var(--md-sys-color-primary)",
                      background: "color-mix(in srgb, var(--md-sys-color-primary) 12%, transparent)",
                    }}
                    title={`写入工具 ${r.tool}`}
                  >
                    {r.tool}
                  </span>
                  <span className="shrink-0 text-[9px]">{r.turn > 0 ? `第 ${r.turn} 轮` : "轮外"}</span>
                  <span
                    className="shrink-0 rounded px-1 py-px text-[9px]"
                    style={{
                      color: statusColor(r.status),
                      background: "color-mix(in srgb, currentColor 10%, transparent)",
                    }}
                  >
                    {versionStatusText(r.status)}
                  </span>
                  <span className="min-w-0 flex-1" />
                  <button
                    type="button"
                    className={iconBtn}
                    onClick={() => handleCompare(r)}
                    disabled={!r.baselinePath}
                    title={cmpOpen ? t("vcompare.hide") : t("vcompare.action")}
                    aria-label={`${t("vcompare.action")} ${versionTimeText(r)} ${r.tool}`}
                  >
                    <Diff size={11} />
                  </button>
                  <button
                    type="button"
                    className={iconBtn}
                    onClick={() => onPreview(r.baselinePath ?? "")}
                    disabled={!r.baselinePath}
                    title="预览该版本快照"
                    aria-label={`预览 ${versionTimeText(r)} ${r.tool} 版本快照`}
                  >
                    <Eye size={11} />
                  </button>
                  <button
                    type="button"
                    className={iconBtn}
                    onClick={() => handleRestore(r)}
                    disabled={restoringId !== null}
                    title={`恢复到 ${versionTimeText(r)} 版本：将回滚到该时间版本`}
                    aria-label={`恢复到 ${versionTimeText(r)} ${r.tool} 版本`}
                  >
                    {restoring ? <Loader2 size={11} className="animate-spin" /> : <Rollback size={11} />}
                  </button>
                </div>
                {/* 内联对比区：仅展开行挂载（收起即取消挂载，不留 DOM） */}
                {cmpOpen && cmp && (
                  <VersionComparePanel
                    label={versionTimeText(r)}
                    result={cmp.result}
                    showAll={showAllRows}
                    onToggleAll={() => setShowAllRows((v) => !v)}
                    onHide={collapseCompare}
                    path={path}
                  />
                )}
              </li>
            );
          })}
        </ol>
      )}
    </div>
  );
}
