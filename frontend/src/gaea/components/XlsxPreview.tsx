import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { CSSProperties } from "react";
import { AlertCircle, BarChart3, Check, ChevronDown, FileText, LineChart, Loader2, MessageSquare, PieChart, RefreshCw, Table, Wand2, X } from "../icons";
import { app } from "../lib/bridge";
import { useComposerInsertStore, useUpdatedFilesStore } from "../lib/store";
import {
  gbaseMissingColumns,
  gbaseSheetModel,
  gbaseSidecarPath,
  parseGbaseConfig,
  type GbaseSheetModel,
  type GbaseView,
} from "../lib/gbase";
import { GbaseGroupedView } from "./GbaseGroupedView";
import { useToast } from "./Toast";
import type { XlsxCell, XlsxChartResult, XlsxPlanResult, XlsxPreview, XlsxSheet } from "../lib/types";

function parseRef(ref: string): { col: number; row: number } {
  const m = /^([A-Z]+)(\d+)$/.exec(ref);
  if (!m) return { col: 0, row: 0 };
  let col = 0;
  for (const ch of m[1]) col = col * 26 + (ch.charCodeAt(0) - 64);
  return { col, row: parseInt(m[2], 10) };
}

function colToLetter(n: number): string {
  let s = "";
  while (n > 0) {
    const rem = (n - 1) % 26;
    s = String.fromCharCode(65 + rem) + s;
    n = Math.floor((n - 1) / 26);
  }
  return s;
}

// —— 数字格式显示（NumFmt 近似实现）——
// 内置编号 → 格式串（只覆盖常用编号，其余原样显示）
const BUILTIN_NUM_FMTS: Record<string, string> = {
  "1": "0",
  "2": "0.00",
  "3": "#,##0",
  "4": "#,##0.00",
  "5": "$#,##0",
  "6": "$#,##0",
  "7": "$#,##0.00",
  "8": "$#,##0.00",
  "9": "0%",
  "10": "0.00%",
  "11": "0.00E+00",
  "44": "¥#,##0",
  "45": "¥#,##0",
  "46": "¥#,##0.00",
  "47": "¥#,##0.00",
};

const DATE_FMT_IDS = new Set(["14", "15", "16", "17", "18", "19", "20", "21", "22", "27", "28", "29", "30", "31", "32", "33", "34", "35", "36", "50", "51", "52", "53", "54", "55", "56", "57", "58"]);

function applyNumFmt(n: number, fmt: string): string {
  let pattern = fmt.trim();
  if (/^\d+$/.test(pattern) && DATE_FMT_IDS.has(pattern)) return String(n);
  if (/^\d+$/.test(pattern) && BUILTIN_NUM_FMTS[pattern]) pattern = BUILTIN_NUM_FMTS[pattern];
  if (!pattern || /general/i.test(pattern)) return String(n);
  // 只取分号前第一段，去掉 [DBNum*]/[$-xxx]/[Red] 等标记
  pattern = (pattern.split(";")[0] ?? pattern).replace(/\[[^\]]*\]/g, "").trim();
  if (!pattern) return String(n);
  const neg = n < 0;
  const abs = Math.abs(n);
  const isPct = pattern.includes("%");
  const v = isPct ? abs * 100 : abs;
  const m = /\.(0+)/.exec(pattern);
  const decimals = m ? m[1].length : pattern.includes(".") ? 2 : 0;
  const hasThousands = pattern.includes(",");
  let s = v.toFixed(decimals);
  if (hasThousands) {
    const [ip, fp] = s.split(".");
    s = ip.replace(/\B(?=(\d{3})+(?!\d))/g, ",") + (fp !== undefined ? "." + fp : "");
  }
  if (isPct) s += "%";
  if (pattern.includes("$")) s = "$" + s;
  if (pattern.includes("¥") || pattern.includes("￥")) s = "¥" + s;
  if (pattern.includes("€")) s = "€" + s;
  return (neg ? "-" : "") + s;
}

function formatCellValue(cell: XlsxCell | undefined): string {
  const raw = cell?.value ?? "";
  if (raw === "") return "";
  if (cell?.type === "error") return raw;
  const fmt = cell?.style?.numFmt ?? "";
  if (!fmt || /general/i.test(fmt)) return raw;
  const n = Number(raw);
  if (!Number.isFinite(n)) return raw;
  return applyNumFmt(n, fmt);
}

/**
 * XlsxPreview 渲染后端提取的结构化单元格视图：sheet 切换、公式标识、
 * 样式近似还原、合并单元格与列宽。为 P0-③ 单元格编辑提供承载。
 */
const EDIT_PRESETS = [
  { label: "求和", instruction: "对相关数据求和，把公式写入选中单元格" },
  { label: "平均值", instruction: "计算相关数据的平均值，把公式写入选中单元格" },
  { label: "拆分列", instruction: "把选中单元格所在列按分隔符拆分为多列，新列写入相邻列并加表头" },
  { label: "清洗", instruction: "清洗相关数据：去掉首尾空格、统一大小写" },
  { label: "加粗", instruction: "把选中单元格加粗（保留其他样式）" },
  { label: "高亮", instruction: "把选中单元格填充浅黄色 FFF2CC" },
  { label: "合并表头", instruction: "把选中区域合并为一个居中表头单元格" },
];

export function XlsxPreview({
  body,
  fileName,
  relPath,
  onQuoteSelection,
}: {
  body: string;
  fileName: string;
  relPath: string;
  /** B3 选区联动：选中单元格引用到对话的自定义出口；缺省走 composer 插入通道。 */
  onQuoteSelection?: (quote: string) => void;
}) {
  const [active, setActive] = useState(0);
  const [selected, setSelected] = useState<string | null>(null);
  const [docBody, setDocBody] = useState(body);
  const markUpdated = useUpdatedFilesStore((s) => s.markUpdated);
  const [instruction, setInstruction] = useState("");
  const [running, setRunning] = useState(false);
  const [editError, setEditError] = useState("");
  const [notice, setNotice] = useState("");
  const [embedding, setEmbedding] = useState(false);
  const [recalcing, setRecalcing] = useState(false);
  const [rowOpsBusy, setRowOpsBusy] = useState(false);
  const [confirmDeleteRow, setConfirmDeleteRow] = useState(false);
  // 列操作 + 预览内排序/筛选（简单模式：排序筛选不改文件）
  const [selectedCol, setSelectedCol] = useState<string | null>(null);
  const [colOpsBusy, setColOpsBusy] = useState(false);
  const [confirmDeleteCol, setConfirmDeleteCol] = useState(false);
  // B1 多维表视图层：.gbase.json sidecar 视图配置 + 激活视图（null=表格视图，既有行为不变）
  const [gbaseState, setGbaseState] = useState<{ views: GbaseView[]; warnings: string } | null>(null);
  const [activeViewId, setActiveViewId] = useState<string | null>(null);
  // P0-2 表格「选中区域 → 一键图表」：选中单元格后把原生图表嵌入工作簿，
  // 前端用 SVG 迷你图即时反馈（原生对象在 Excel/WPS 中可见可编辑）
  const [chartBusy, setChartBusy] = useState(false);
  const [chart, setChart] = useState<XlsxChartResult | null>(null);
  // AI 编辑审阅制：先规划（临时副本试运行 + diff），用户批准后才落盘
  const [plan, setPlan] = useState<XlsxPlanResult | null>(null);
  const [applying, setApplying] = useState(false);
  // v3.0.8 工具栏收敛：图表动作（柱/线/饼/→Word/→PPT）收进一个下拉菜单，
  // 不再 5 个按钮常驻一字排开（与右侧 Tab 收敛同一原则：按上下文、不按功能铺开）
  const [chartMenuOpen, setChartMenuOpen] = useState(false);
  const chartMenuRef = useRef<HTMLDivElement>(null);
  // Excel 式直接编辑：双击单元格/在 fx 栏输入 → 写回文件
  const [editingRef, setEditingRef] = useState<string | null>(null);
  const [draft, setDraft] = useState("");
  const [saving, setSaving] = useState(false);
  const toast = useToast();

  useEffect(() => {
    setDocBody(body);
    setSelected(null);
    setEditingRef(null);
    setEditError("");
    setPlan(null);
    setChart(null);
  }, [body]);

  // B1 sidecar 加载：文件不存在/读取失败/内容非视图配置 → 无视图（静默，= 既有行为）；
  // 坏配置（JSON 坏/根形状坏）→ warnings 横幅；字段级容错（坏视图丢弃、好视图保留）
  // 由 parseGbaseConfig 承担。GaeaReadFile 契约 = {path, markdown, size}（无 kind）。
  useEffect(() => {
    let live = true;
    setActiveViewId(null);
    app.ReadFile(gbaseSidecarPath(relPath))
      .then((r) => {
        if (!live) return;
        const text = r.markdown ?? "";
        // 嗅探：不含 "views" 的文本（mock 兜底正文、纯注释文件）不当配置报错
        if (!text.includes('"views"')) {
          setGbaseState(null);
          return;
        }
        const parsed = parseGbaseConfig(text);
        if (!parsed.config) {
          setGbaseState({ views: [], warnings: parsed.error });
          return;
        }
        setGbaseState({ views: parsed.config.views, warnings: parsed.error });
      })
      .catch(() => {
        if (live) setGbaseState(null);
      });
    return () => {
      live = false;
    };
    // U4 写后预览实时跟随：body 变化（agent 改了 xlsx 本体）也重读 sidecar——
    // 技能口径里 .gbase.json 视图配置与 xlsx 同名同改，本体刷新而视图不刷会
    // 呈现滞后口径。重读会复位激活视图（配置可能已变，诚实重置）。
  }, [relPath, body]);

  const preview = useMemo<XlsxPreview | null>(() => {
    try {
      return JSON.parse(docBody) as XlsxPreview;
    } catch {
      return null;
    }
  }, [docBody]);

  // 规划：AI 操作集在临时副本上试运行，返回变更清单，不落盘
  // B3 选区联动：选中单元格 → 浮动「引用到对话」→ 以引用块插入输入框
  // （走既有 composer 插入通道 requestText，与 SelectionToComposer/WorkspacePanel
  // 的 reference() 同一机制；onQuoteSelection 提供宿主自定义出口）。
  const quoteToComposer = useCallback(() => {
    if (!selected || !preview) return;
    const sheet = preview.sheets[Math.min(active, preview.sheets.length - 1)];
    let cell: XlsxCell | undefined;
    for (const row of sheet.rows) {
      const hit = row.find((x) => x.ref === selected);
      if (hit) {
        cell = hit;
        break;
      }
    }
    const val = cell ? (cell.formula ? `fx =${cell.formula}（值 ${String(cell.value ?? "")}）` : String(cell.value ?? "")) : "";
    const quote = `> 《${fileName}》${sheet.name}!${selected}：${val}\n\n请基于以上选中单元格继续处理。`;
    if (onQuoteSelection) onQuoteSelection(quote);
    else useComposerInsertStore.getState().requestText(quote);
    toast.show("已引用到输入框", "info");
  }, [selected, preview, active, fileName, onQuoteSelection, toast]);

  const runEdit = useCallback(async () => {    if (!selected || !instruction.trim() || !preview) return;
    const sheet = preview.sheets[Math.min(active, preview.sheets.length - 1)];
    setRunning(true);
    setEditError("");
    try {
      const r = await app.XlsxPlanEdit(relPath, sheet.name, instruction.trim(), selected);
      setPlan(r);
    } catch (e) {
      setEditError(e instanceof Error ? e.message : String(e));
    } finally {
      setRunning(false);
    }
  }, [selected, instruction, preview, active, relPath]);

  // 应用：用户批准规划后执行操作集（excelize + LibreOffice 重算）
  const applyPlan = useCallback(async () => {
    if (!plan || applying) return;
    setApplying(true);
    setEditError("");
    try {
      const r = await app.XlsxApplyEdit(relPath, plan.ops);
      setDocBody(r.preview);
      markUpdated(relPath);
      setPlan(null);
      setInstruction("");
      setSelected(null);
      setNotice(`已应用：${r.summary}`);
      window.setTimeout(() => setNotice(""), 6000);
    } catch (e) {
      setEditError(e instanceof Error ? e.message : String(e));
    } finally {
      setApplying(false);
    }
  }, [plan, applying, relPath, markUpdated]);

  const closeEdit = useCallback(() => {
    setSelected(null);
    setInstruction("");
    setEditError("");
    setPlan(null);
  }, []);

  // 点击图表菜单外部关闭下拉
  useEffect(() => {
    if (!chartMenuOpen) return;
    const onDown = (e: PointerEvent) => {
      if (chartMenuRef.current && !chartMenuRef.current.contains(e.target as Node)) {
        setChartMenuOpen(false);
      }
    };
    document.addEventListener("pointerdown", onDown);
    return () => document.removeEventListener("pointerdown", onDown);
  }, [chartMenuOpen]);

  const startEditCell = useCallback((ref: string, initial: string) => {
    setSelected(ref);
    setDraft(initial);
    setEditingRef(ref);
  }, []);

  const cancelEdit = useCallback(() => setEditingRef(null), []);

  // 直接写单元格：值 / =公式 回车后保存并重算
  const commitCell = useCallback(
    async (ref: string, raw: string) => {
      if (!preview || !ref || saving) return;
      const sheet = preview.sheets[Math.min(active, preview.sheets.length - 1)];
      setSaving(true);
      setEditError("");
      setEditingRef(null);
      try {
        const r = await app.XlsxSetCell(relPath, sheet.name, ref, raw);
        setDocBody(r.preview);
        markUpdated(relPath);
        setNotice(`已更新 ${ref}`);
        window.setTimeout(() => setNotice(""), 4000);
      } catch (e) {
        setEditError(e instanceof Error ? e.message : String(e));
        setEditingRef(ref);
      } finally {
        setSaving(false);
      }
    },
    [preview, active, relPath, saving, markUpdated],
  );

  const embedChart = useCallback(
    async (into: "docx" | "pptx") => {
      if (!preview) return;
      const sheet = preview.sheets[Math.min(active, preview.sheets.length - 1)];
      setEmbedding(true);
      setEditError("");
      try {
        const r = await app.CrossEmbed({
          xlsxRel: relPath,
          sheet: sheet.name,
          chartType: "bar",
          title: fileName.replace(/\.xlsx$/i, ""),
          into,
        });
        setNotice(`已生成 ${r.name}（数据更新后重新导出即同步图表）`);
        void app.RevealWorkspacePath(r.path).catch(() => {});
        window.setTimeout(() => setNotice(""), 6000);
      } catch (e) {
        setEditError(e instanceof Error ? e.message : String(e));
      } finally {
        setEmbedding(false);
      }
    },
    [preview, active, relPath, fileName],
  );

  // P0-2 表格「选中区域 → 一键图表」：把选中区域（或自动前两列）数据生成
  // 原生图表嵌入工作簿（Excel/WPS 打开即可见可编辑），前端用 SVG 迷你图即时反馈。
  const genChart = useCallback(
    async (chartType: "bar" | "line" | "pie") => {
      if (!preview || chartBusy) return;
      const sheet = preview.sheets[Math.min(active, preview.sheets.length - 1)];
      setChartBusy(true);
      setEditError("");
      try {
        const r = await app.XlsxChart({
          rel: relPath,
          sheet: sheet.name,
          refs: selected ?? undefined,
          chartType,
          title: fileName.replace(/\.xlsx$/i, ""),
        });
        markUpdated(relPath);
        setChart(r);
        setNotice(`已嵌入图表到 ${r.sheet}!${r.anchor}（在 Excel/WPS 中打开可见、可继续编辑）`);
        window.setTimeout(() => setNotice(""), 6000);
      } catch (e) {
        setEditError(e instanceof Error ? e.message : String(e));
      } finally {
        setChartBusy(false);
      }
    },
    [preview, active, relPath, fileName, selected, chartBusy, markUpdated],
  );

  // 手动重算公式（预览打开时已自动兜底；用户可随时主动刷新）
  const recalc = useCallback(async () => {
    if (!preview || recalcing) return;
    setRecalcing(true);
    setEditError("");
    try {
      const r = await app.XlsxRecalc(relPath);
      setDocBody(r.preview);
      markUpdated(relPath);
      setNotice(r.summary);
      window.setTimeout(() => setNotice(""), 5000);
    } catch (e) {
      setEditError(e instanceof Error ? e.message : String(e));
    } finally {
      setRecalcing(false);
    }
  }, [preview, recalcing, relPath, markUpdated]);

  // 行级操作：插上行/插下行/删除行（基于选中单元格所在行）
  const rowOps = useCallback(
    async (action: "insert_before" | "insert_after" | "delete") => {
      if (!preview || !selected || rowOpsBusy) return;
      const sheet = preview.sheets[Math.min(active, preview.sheets.length - 1)];
      setRowOpsBusy(true);
      setEditError("");
      setConfirmDeleteRow(false);
      try {
        const r = await app.XlsxRowOps(relPath, sheet.name, action, selected);
        setDocBody(r.preview);
        markUpdated(relPath);
        setSelected(null);
        setNotice(r.summary);
        window.setTimeout(() => setNotice(""), 5000);
      } catch (e) {
        setEditError(e instanceof Error ? e.message : String(e));
      } finally {
        setRowOpsBusy(false);
      }
    },
    [preview, active, selected, relPath, rowOpsBusy, markUpdated],
  );

  // 切换选区时取消“删除行”二次确认
  useEffect(() => {
    setConfirmDeleteRow(false);
  }, [selected]);

  const colOps = useCallback(
    async (action: "insert_before" | "insert_after" | "delete") => {
      if (!preview || !selectedCol || colOpsBusy) return;
      const sheet = preview.sheets[Math.min(active, preview.sheets.length - 1)];
      setColOpsBusy(true);
      setEditError("");
      setConfirmDeleteCol(false);
      try {
        const r = await app.XlsxColOps(relPath, sheet.name, action, `${selectedCol}1`);
        setDocBody(r.preview);
        markUpdated(relPath);
        setSelected(null);
        setSelectedCol(null);
        setNotice(r.summary);
        window.setTimeout(() => setNotice(""), 5000);
      } catch (e) {
        setEditError(e instanceof Error ? e.message : String(e));
      } finally {
        setColOpsBusy(false);
      }
    },
    [preview, active, selectedCol, relPath, colOpsBusy, markUpdated],
  );

  const selectCol = useCallback((letter: string) => {
    setSelected(null);
    setEditingRef(null);
    setConfirmDeleteCol(false);
    setSelectedCol((prev) => (prev === letter ? null : letter));
  }, []);

  // B1 视图解析：gModel 仅在激活视图时计算（hooks 必须在早退 return 之前）
  const activeView = gbaseState?.views.find((v) => v.id === activeViewId) ?? null;
  const gModel = useMemo<GbaseSheetModel | null>(() => {
    if (!activeView || !preview || preview.sheets.length === 0) return null;
    const s = preview.sheets[Math.min(active, preview.sheets.length - 1)];
    return gbaseSheetModel(s);
  }, [activeView, preview, active]);

  if (!preview || preview.sheets.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-full gap-3 px-8 text-center">
        <AlertCircle size={30} className="text-err/70" />
        <div className="text-[13px] text-fg-dim">该表格无法预览</div>
      </div>
    );
  }

  const sheet = preview.sheets[Math.min(active, preview.sheets.length - 1)];
  const selectedRow = selected ? Number(/^[A-Z]+(\d+)$/.exec(selected)?.[1]) : 0;
  // sheet 绑定失配或列失配 → 降级表格视图（横幅如实提示，不静默装好）
  const viewSheetMismatch = activeView?.sheet !== undefined && activeView.sheet !== sheet.name;
  const missingCols =
    activeView && !viewSheetMismatch && gModel ? gbaseMissingColumns(activeView, gModel.fields) : [];
  const viewUsable = activeView !== null && !viewSheetMismatch && missingCols.length === 0;

  return (
    <div className="flex flex-col h-full relative">
      {notice && (
        <div className="flex items-center gap-1.5 px-3 py-1.5 border-b border-accent/25 bg-accent/8 text-accent text-[11.5px]">
          <Check size={12} />
          <span className="truncate">{notice}</span>
        </div>
      )}
      {/* B1 视图层降级/告警横幅：坏配置、sheet 失配、列失配（genui 红横幅口径的琥珀版） */}
      {(gbaseState?.warnings || (activeView && (viewSheetMismatch || missingCols.length > 0))) && (
        <div className="flex items-center gap-1.5 px-3 py-1.5 border-b border-amber-500/30 bg-amber-500/10 text-amber-500 text-[11px]">
          <AlertCircle size={12} className="shrink-0" />
          <span className="truncate">
            {gbaseState?.warnings ? `视图配置告警：${gbaseState.warnings}` : ""}
            {activeView && viewSheetMismatch
              ? `视图「${activeView.name}」仅作用于 sheet「${activeView.sheet}」，当前「${sheet.name}」`
              : ""}
            {activeView && !viewSheetMismatch && missingCols.length > 0
              ? `视图「${activeView.name}」引用的列不存在：${missingCols.join("、")}，已回退表格视图`
              : ""}
          </span>
          {activeViewId !== null && (
            <button
              className="ml-auto shrink-0 px-2 py-0.5 rounded border border-amber-500/40 text-amber-500 hover:bg-amber-500/10 cursor-pointer"
              onClick={() => setActiveViewId(null)}
            >
              回表格
            </button>
          )}
        </div>
      )}
      {/* 原生图表迷你预览：图表对象已嵌入 xlsx，这里是即时视觉反馈 */}
      {chart && (
        <div className="px-3 py-2 border-b border-border-soft bg-bg shrink-0">
          <div className="flex items-center gap-1.5 text-[11px] text-fg-dim">
            <BarChart3 size={11} className="text-accent shrink-0" />
            <span className="truncate">
              {chart.title} · {chart.labels} 个数据点 · 已嵌入 {chart.sheet}!{chart.anchor}
            </span>
            <button
              className="ml-auto flex items-center justify-center w-5 h-5 rounded-md border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg shrink-0"
              onClick={() => setChart(null)}
              title="关闭图表预览"
            >
              <X size={11} />
            </button>
          </div>
          <MiniChart chartType={chart.chartType} labels={chart.labelList} values={chart.values} />
        </div>
      )}
      <div className="flex items-center gap-2 px-3 py-1.5 border-b border-border-soft bg-bg-soft/40 text-fg-faint text-[11px] shrink-0">
        <FileText size={11} />
        <span className="truncate">{fileName}</span>
        <span>·</span>
        <span>单元格预览</span>
        <span className="ml-auto inline-flex items-center gap-1.5 shrink-0">
          {/* v3.0.8 收敛：行操作只在选中单元格时出现（按上下文，不常驻铺开） */}
          {selected && editingRef === null && (
            <>
              <span className="w-px h-3.5 bg-border-soft shrink-0" />
              <button
                className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md border border-border-soft bg-transparent text-fg-dim text-[10px] cursor-pointer hover:bg-bg-soft disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
                onClick={() => void rowOps("insert_before")}
                disabled={rowOpsBusy}
                title="在选中行上方插入空行"
              >
                ↑ 插行
              </button>
              <button
                className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md border border-border-soft bg-transparent text-fg-dim text-[10px] cursor-pointer hover:bg-bg-soft disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
                onClick={() => void rowOps("insert_after")}
                disabled={rowOpsBusy}
                title="在选中行下方插入空行"
              >
                ↓ 插行
              </button>
              {confirmDeleteRow ? (
                <button
                  className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md border border-err/40 bg-err/15 text-err text-[10px] cursor-pointer hover:bg-err/25 transition-colors"
                  onClick={() => void rowOps("delete")}
                  disabled={rowOpsBusy}
                  title="再次点击确认删除选中行"
                >
                  {rowOpsBusy ? <Loader2 size={10} className="animate-spin" /> : null}
                  确认删除
                </button>
              ) : (
                <button
                  className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md border border-border-soft bg-transparent text-fg-dim text-[10px] cursor-pointer hover:bg-bg-soft disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
                  onClick={() => setConfirmDeleteRow(true)}
                  disabled={rowOpsBusy}
                  title="删除选中行（需再次确认）"
                >
                  删除行
                </button>
              )}
            </>
          )}
          {selectedRow > 0 && <span className="text-fg-faint text-[10px] shrink-0">第 {selectedRow} 行</span>}
          <button
            className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md border border-border-soft bg-transparent text-fg-dim text-[10px] cursor-pointer hover:bg-bg-soft disabled:opacity-50 transition-colors"
            onClick={() => void recalc()}
            disabled={recalcing}
            title="用 LibreOffice 重算全部公式并刷新结果"
          >
            {recalcing ? <Loader2 size={10} className="animate-spin" /> : <RefreshCw size={10} />}
            重算公式
          </button>
          {/* 图表动作收敛为下拉菜单（柱/线/饼/嵌入 Word/嵌入 PPT） */}
          <div className="relative" ref={chartMenuRef}>
            <button
              className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md border border-accent/30 bg-accent/10 text-accent text-[10px] cursor-pointer hover:bg-accent/20 disabled:opacity-50 transition-colors"
              onClick={() => setChartMenuOpen((o) => !o)}
              disabled={chartBusy || embedding}
              title="把选中区域数据生成原生图表，嵌入工作簿（Excel/WPS 可见可编辑）"
              aria-expanded={chartMenuOpen}
              aria-haspopup="menu"
            >
              {chartBusy || embedding ? <Loader2 size={10} className="animate-spin" /> : <BarChart3 size={10} />}
              图表
              <ChevronDown size={9} />
            </button>
            {chartMenuOpen && (
              <div
                role="menu"
                className="absolute right-0 top-[calc(100%+4px)] z-30 min-w-[150px] bg-bg-elev border border-border rounded-[10px] p-1 anim-menu-in"
                style={{ boxShadow: "var(--ds-shadow-dropdown)" }}
              >
                {[
                  { label: "柱状图", icon: <BarChart3 size={11} />, run: () => void genChart("bar") },
                  { label: "折线图", icon: <LineChart size={11} />, run: () => void genChart("line") },
                  { label: "饼图", icon: <PieChart size={11} />, run: () => void genChart("pie") },
                  { label: "图表→Word", icon: <Wand2 size={11} />, run: () => void embedChart("docx") },
                  { label: "图表→PPT", icon: <Wand2 size={11} />, run: () => void embedChart("pptx") },
                ].map((item) => (
                  <button
                    key={item.label}
                    role="menuitem"
                    className="flex items-center gap-2 w-full px-2 py-1.5 rounded-md bg-transparent border-0 text-[11.5px] text-fg-dim text-left cursor-pointer hover:bg-accent-soft hover:text-accent transition-colors"
                    onClick={() => { setChartMenuOpen(false); item.run(); }}
                  >
                    {item.icon}
                    {item.label}
                  </button>
                ))}
              </div>
            )}
          </div>
        </span>
        <span className="inline-flex items-center gap-1 text-accent/80">
          <Wand2 size={10} />
          单击选中 · 双击直接编辑
        </span>
      </div>

      {/* sheet 切换 */}
      <div className="flex items-center gap-1 px-3 py-1.5 border-b border-border-soft bg-bg shrink-0 overflow-x-auto">
        {preview.sheets.map((s, i) => (
          <button
            key={s.name + i}
            className={`px-2.5 py-1 rounded-md text-[12px] cursor-pointer transition-colors shrink-0 ${
              i === active
                ? "bg-accent/12 text-accent border border-accent/30"
                : "text-fg-dim border border-transparent hover:bg-bg-soft"
            }`}
            onClick={() => {
              setActive(i);
              setSelected(null);
              setSelectedCol(null);
            }}
          >
            {s.name}
          </button>
        ))}
        {gbaseState && gbaseState.views.length > 0 && (
          <>
            <span className="w-px h-4 bg-border-soft mx-1 shrink-0" />
            <button
              data-testid="gbase-view-grid"
              className={`px-2.5 py-1 rounded-md text-[12px] cursor-pointer transition-colors shrink-0 ${
                activeViewId === null
                  ? "bg-accent/12 text-accent border border-accent/30"
                  : "text-fg-dim border border-transparent hover:bg-bg-soft"
              }`}
              onClick={() => setActiveViewId(null)}
            >
              表格
            </button>
            {gbaseState.views.map((v) => (
              <button
                key={v.id}
                data-testid={`gbase-view-${v.id}`}
                title={v.sheet ? `仅作用于 sheet「${v.sheet}」` : undefined}
                className={`px-2.5 py-1 rounded-md text-[12px] cursor-pointer transition-colors shrink-0 ${
                  activeViewId === v.id
                    ? "bg-accent/12 text-accent border border-accent/30"
                    : "text-fg-dim border border-transparent hover:bg-bg-soft"
                }`}
                onClick={() => setActiveViewId(v.id)}
              >
                {v.name}
              </button>
            ))}
          </>
        )}
        {sheet.truncated && (
          <span className="ml-auto text-[10px] text-amber-500/80 shrink-0">
            大表格已截断（仅预览前 2000 行 × 100 列）
          </span>
        )}
      </div>

      {/* 列操作栏：点击列字母选中后出现（插列/删列/排序/筛选） */}
      {selectedCol && (
        <div className="flex items-center gap-1.5 px-3 py-1.5 border-b border-border-soft bg-bg shrink-0 text-[11px] overflow-x-auto">
          <span className="text-fg font-medium shrink-0">列 {selectedCol}</span>
          <button
            className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md border border-border-soft bg-transparent text-fg-dim cursor-pointer hover:bg-bg-soft disabled:opacity-40 transition-colors"
            onClick={() => void colOps("insert_before")}
            disabled={colOpsBusy}
            title="在选中列左侧插入空列"
          >
            ← 插列
          </button>
          <button
            className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md border border-border-soft bg-transparent text-fg-dim cursor-pointer hover:bg-bg-soft disabled:opacity-40 transition-colors"
            onClick={() => void colOps("insert_after")}
            disabled={colOpsBusy}
            title="在选中列右侧插入空列"
          >
            → 插列
          </button>
          {confirmDeleteCol ? (
            <button
              className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md border border-err/40 bg-err/15 text-err cursor-pointer hover:bg-err/25 transition-colors"
              onClick={() => void colOps("delete")}
              disabled={colOpsBusy}
              title="再次点击确认删除选中列"
            >
              {colOpsBusy ? <Loader2 size={10} className="animate-spin" /> : null}
              确认删除
            </button>
          ) : (
            <button
              className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md border border-border-soft bg-transparent text-fg-dim cursor-pointer hover:bg-bg-soft transition-colors"
              onClick={() => setConfirmDeleteCol(true)}
              title="删除选中列（需再次确认）"
            >
              删除列
            </button>
          )}
        </div>
      )}

      {/* 公式栏（可直接输入值或 =公式，回车写回）——选中单元格后的第一层操作 */}
      <FormulaBar
        sheet={sheet}
        selected={selected}
        onCommit={commitCell}
        disabled={saving || editingRef !== null}
      />

      {selected && (
        <div className="px-3 py-1.5 border-b border-border-soft bg-bg shrink-0">
          <div className="flex flex-wrap items-center gap-1.5">
            <span className="text-[11px] text-fg font-medium shrink-0">
              AI 编辑 <span className="font-mono text-accent">{selected}</span>
            </span>
            {EDIT_PRESETS.map((p) => (
              <button
                key={p.label}
                className={`px-2 py-0.5 rounded-md border text-[10.5px] cursor-pointer transition-colors ${
                  instruction === p.instruction
                    ? "border-accent/40 bg-accent/10 text-accent"
                    : "border-border-soft bg-transparent text-fg-dim hover:bg-accent/10 hover:text-accent hover:border-accent/30"
                }`}
                onClick={() => setInstruction(p.instruction)}
                title={p.instruction}
              >
                {p.label}
              </button>
            ))}
            <input
              value={instruction}
              onChange={(e) => setInstruction(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault();
                  runEdit();
                }
              }}
              placeholder="或输入指令，如：对 B 列求和写入 B5 / 拆分 A 列…"
              className="flex-1 min-w-[160px] px-2.5 py-1 rounded-lg border border-border-soft bg-bg text-[12px] text-fg outline-none focus:border-accent/50"
            />
            <button
              className="inline-flex items-center gap-1 px-3 py-1 rounded-lg bg-accent text-bg text-[12px] font-medium cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed hover:opacity-90 transition-opacity"
              disabled={!instruction.trim() || running}
              onClick={runEdit}
            >
              {running ? <Loader2 size={12} className="animate-spin" /> : <Wand2 size={12} />}
              执行
            </button>
            <button
              className="flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg"
              onClick={closeEdit}
              title="取消 (Esc)"
            >
              <X size={12} />
            </button>
          </div>
          {editError && <div className="mt-1 text-[11px] text-err">{editError}</div>}
        </div>
      )}

      {/* 规划审阅卡：AI 编辑先出变更清单，批准后才落盘（Plan/Show Changes 范式） */}
      {plan && (
        <div className="px-3 py-2 border-b border-border-soft bg-bg shrink-0">
          <div className="flex flex-wrap items-center gap-1.5 text-[11px]">
            <Check size={12} className="text-accent shrink-0" />
            <span className="text-fg font-medium shrink-0">
              待应用变更（{plan.total} 处{plan.truncated ? `，仅列出前 ${plan.changes.length} 条` : ""}）
            </span>
            <span className="text-fg-faint truncate flex-1 min-w-[80px]" title={plan.summary}>
              {plan.summary}
            </span>
            <button
              className="inline-flex items-center gap-1 px-3 py-1 rounded-lg bg-accent text-bg text-[11.5px] font-medium cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed hover:opacity-90 transition-opacity shrink-0"
              disabled={applying}
              onClick={() => void applyPlan()}
            >
              {applying ? <Loader2 size={11} className="animate-spin" /> : <Check size={11} />}
              应用
            </button>
            <button
              className="px-2 py-1 rounded-lg border border-border-soft bg-transparent text-fg-dim text-[11.5px] cursor-pointer hover:bg-bg-soft transition-colors shrink-0"
              onClick={() => setPlan(null)}
            >
              放弃
            </button>
          </div>
          {plan.changes.length > 0 && (
            <div className="mt-1.5 max-h-36 overflow-auto rounded-lg border border-border-soft">
              <table className="w-full text-[11px] font-mono border-collapse">
                <tbody>
                  {plan.changes.map((c, i) => (
                    <tr key={`${c.sheet}!${c.cell}-${i}`} className="border-b border-border-soft/60 last:border-b-0">
                      <td className="px-2 py-1 text-fg-faint whitespace-nowrap align-top">{c.sheet}!{c.cell}</td>
                      <td className="px-2 py-1 text-fg-faint line-through max-w-[140px] truncate align-top">
                        {c.before || "（空）"}
                      </td>
                      <td className="px-1 py-1 text-accent align-top">→</td>
                      <td className="px-2 py-1 text-fg max-w-[200px] truncate align-top">
                        {c.formula ? (
                          <span className="text-accent" title={`=${c.formula}`}>fx ={c.formula}</span>
                        ) : (
                          c.after || "（空）"
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
          {editError && <div className="mt-1 text-[11px] text-err">{editError}</div>}
        </div>
      )}

      <div className="flex-1 min-h-0">
        {viewUsable && gModel && activeView ? (
          <GbaseGroupedView model={gModel} view={activeView} />
        ) : (
        <SheetGrid
          sheet={sheet}
          selected={selected}
          onSelect={(ref) => {
            setSelectedCol(null);
            setSelected(ref);
          }}
          selectedCol={selectedCol}
          onSelectCol={selectCol}
          editingRef={editingRef}
          draft={draft}
          onDraftChange={setDraft}
          onEditCell={startEditCell}
          onCommit={commitCell}
          onCancelEdit={cancelEdit}
          disabled={saving}
        />
        )}
        {/* B3 浮动「引用到对话」：选中单元格且不在直接编辑时出现（次级入口，
            不抢占既有 AI 编辑/行列表头/双击编辑等任何交互） */}
        {selected && editingRef === null && (
          <button
            className="absolute bottom-4 left-4 z-20 inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg border border-accent/30 bg-bg-elev-2 text-accent text-[11px] shadow-lg cursor-pointer hover:bg-accent/10 transition-colors"
            onClick={quoteToComposer}
            title="把选中单元格以引用块插入输入框，可编辑后发送"
            data-testid="xlsx-quote-btn"
          >
            <MessageSquare size={11} aria-hidden />
            引用到对话
          </button>
        )}
      </div>
    </div>
  );
}

function FormulaBar({
  sheet,
  selected,
  onCommit,
  disabled,
}: {
  sheet: XlsxSheet;
  selected: string | null;
  onCommit: (ref: string, value: string) => void;
  disabled?: boolean;
}) {
  const cell = useMemo(() => {
    if (!selected) return null;
    for (const row of sheet.rows) {
      const c = row.find((x) => x.ref === selected);
      if (c) return c;
    }
    return null;
  }, [sheet, selected]);

  const [draft, setDraft] = useState("");
  useEffect(() => {
    setDraft(cell ? (cell.formula ? `=${cell.formula}` : String(cell.value ?? "")) : "");
  }, [cell]);

  return (
    <div className="flex items-center gap-2 px-3 py-1 border-b border-border-soft bg-bg-soft/30 text-[11px] shrink-0">
      <span className="font-mono text-accent w-12 shrink-0">{selected ?? "—"}</span>
      <span className="text-fg-faint shrink-0">fx</span>
      <input
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter") {
            e.preventDefault();
            if (selected && !disabled) onCommit(selected, draft);
          } else if (e.key === "Escape") {
            e.preventDefault();
            setDraft(cell ? (cell.formula ? `=${cell.formula}` : String(cell.value ?? "")) : "");
          }
        }}
        disabled={disabled || !selected}
        placeholder="选中单元格后输入值，或 =公式 直接写公式"
        className="flex-1 min-w-0 px-2 py-1 rounded-md border border-border-soft bg-bg text-[12px] font-mono text-fg-dim outline-none focus:border-accent/50 disabled:opacity-40"
      />
    </div>
  );
}

function SheetGrid({
  sheet,
  selected,
  onSelect,
  selectedCol,
  onSelectCol,
  editingRef,
  draft,
  onDraftChange,
  onEditCell,
  onCommit,
  onCancelEdit,
  disabled,
}: {
  sheet: XlsxSheet;
  selected: string | null;
  onSelect: (ref: string) => void;
  selectedCol: string | null;
  onSelectCol: (letter: string) => void;
  editingRef: string | null;
  draft: string;
  onDraftChange: (v: string) => void;
  onEditCell: (ref: string, initial: string) => void;
  onCommit: (ref: string, value: string) => void;
  onCancelEdit: () => void;
  disabled?: boolean;
}) {
  const { maxRow, maxCol, cellMap, mergeTopLeft, mergeContinuations } = useMemo(() => {
    const map = new Map<string, XlsxCell>();
    let maxRow = 0;
    let maxCol = 0;
    for (const row of sheet.rows) {
      for (const cell of row) {
        const { col, row: r } = parseRef(cell.ref);
        map.set(cell.ref, cell);
        if (r > maxRow) maxRow = r;
        if (col > maxCol) maxCol = col;
      }
    }
    const topLeft = new Map<string, { colspan: number; rowspan: number }>();
    const continuations = new Set<string>();
    for (const m of sheet.merged ?? []) {
      const [a, b] = m.split(":");
      if (!a || !b) continue;
      const p1 = parseRef(a);
      const p2 = parseRef(b);
      const colspan = Math.max(1, p2.col - p1.col + 1);
      const rowspan = Math.max(1, p2.row - p1.row + 1);
      topLeft.set(a, { colspan, rowspan });
      for (let r = p1.row; r <= p2.row; r++) {
        for (let c = p1.col; c <= p2.col; c++) {
          const ref = colToLetter(c) + r;
          if (ref !== a) continuations.add(ref);
        }
      }
    }
    return { maxRow, maxCol, cellMap: map, mergeTopLeft: topLeft, mergeContinuations: continuations };
  }, [sheet]);

  // 冻结窗格：只冻结顶部行（最常见场景），固定行高保证 sticky 偏移对齐
  const freezeRow = Math.max(0, sheet.freeze?.row ?? 0);
  const TH_H = 26;
  const ROW_H = 28;

  // —— 大表格行虚拟滚动（观察项收账）——
  // 后端预览上限 2000 行 × 100 列：全量渲染最多 20 万 td，滚动/切换卡顿。
  // 行数超阈值时只渲染可见窗口 ± overscan（冻结行常驻），spacer 行保持滚动
  // 条总高与行号对齐；小表全量渲染，行为逐字节不变。wrap 单元格在虚拟化
  // 模式下按固定行高裁剪（溢出隐藏），与冻结行既有的 ROW_H 对齐假设一致。
  const VIRTUALIZE_MIN_ROWS = 300;
  const OVERSCAN_ROWS = 10;
  const containerRef = useRef<HTMLDivElement>(null);
  const [scrollTop, setScrollTop] = useState(0);
  const [viewportH, setViewportH] = useState(0);

  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    const measure = () => setViewportH(el.clientHeight);
    measure();
    if (typeof ResizeObserver !== "undefined") {
      const ro = new ResizeObserver(measure);
      ro.observe(el);
      return () => ro.disconnect();
    }
    return undefined;
  }, []);

  // 切换 sheet 重置滚动位置（不同表总高不同，防残留偏移）
  useEffect(() => {
    setScrollTop(0);
    const el = containerRef.current;
    if (el) el.scrollTop = 0;
  }, [sheet]);

  const virtualize = maxRow > VIRTUALIZE_MIN_ROWS;
  const frozen = Math.min(freezeRow, maxRow);
  const renderedRows: number[] = [];
  let topPad = 0;
  let bottomPad = 0;
  if (virtualize) {
    for (let r = 1; r <= frozen; r++) renderedRows.push(r);
    const firstRow = Math.max(1, Math.floor((scrollTop - TH_H) / ROW_H) - OVERSCAN_ROWS);
    const lastRow = Math.min(maxRow, Math.ceil((scrollTop - TH_H + viewportH) / ROW_H) + OVERSCAN_ROWS);
    const start = Math.max(frozen + 1, firstRow);
    const end = Math.max(start - 1, lastRow);
    for (let r = start; r <= end; r++) renderedRows.push(r);
    topPad = Math.min(Math.max(0, start - frozen - 1), Math.max(0, maxRow - frozen));
    bottomPad = Math.max(0, maxRow - end);
  } else {
    for (let r = 1; r <= maxRow; r++) renderedRows.push(r);
  }

  if (maxRow === 0 || maxCol === 0) {
    return <div className="p-6 text-center text-[12px] text-fg-faint">（空工作表）</div>;
  }

  const headerCells: string[] = [];
  for (let c = 1; c <= maxCol; c++) headerCells.push(colToLetter(c));

  return (
    <div
      ref={containerRef}
      onScroll={(e) => setScrollTop(e.currentTarget.scrollTop)}
      className="p-3 h-full overflow-auto docx-preview-body"
    >
      <table
        className="border-collapse text-[12px] leading-tight"
        style={{ borderSpacing: 0 }}
      >
        <thead>
          <tr>
            <th
              className="sticky top-0 left-0 z-30 w-8 min-w-[32px] px-1 py-1 text-center text-[10px] text-fg-faint font-normal"
              style={{ background: "var(--bg-elevated, #181b21)", border: "1px solid rgba(128,128,140,0.16)", height: TH_H }}
            />
            {headerCells.map((l) => (
              <th
                key={l}
                onClick={() => onSelectCol(l)}
                className={`sticky top-0 z-20 px-2 py-1 text-center text-[10px] font-normal cursor-pointer transition-colors ${
                  selectedCol === l ? "text-accent" : "text-fg-faint hover:text-fg"
                }`}
                style={{
                  background: selectedCol === l ? "var(--accent-soft, rgba(99,102,241,0.12))" : "var(--bg-elevated, #181b21)",
                  border: "1px solid rgba(128,128,140,0.16)",
                  minWidth: 56,
                  height: TH_H,
                }}
                title="点击选中列（可插列/删列）"
              >
                {l}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {topPad > 0 && (
            <tr aria-hidden="true" style={{ height: topPad * ROW_H }}>
              <td colSpan={maxCol + 1} style={{ border: "none", padding: 0 }} />
            </tr>
          )}
          {renderedRows.map((r) => (
            <tr key={r} style={virtualize ? { height: ROW_H } : undefined}>
              <td
                className="sticky left-0 z-10 px-1 py-0.5 text-center text-[10px] text-fg-faint"
                style={{
                  background: "var(--bg-elevated, #181b21)",
                  border: "1px solid rgba(128,128,140,0.16)",
                  ...(freezeRow > 0 && r <= freezeRow
                    ? { top: TH_H + (r - 1) * ROW_H, zIndex: 25, height: ROW_H, overflow: "hidden" as const }
                    : {}),
                  ...(virtualize && r > freezeRow ? { overflow: "hidden" as const } : {}),
                }}
              >
                {r}
              </td>
              {headerCells.map((letter) => {
                const ref = letter + r;
                if (mergeContinuations.has(ref)) return null;
                const cell = cellMap.get(ref);
                const merge = mergeTopLeft.get(ref);
                const isEditing = editingRef === ref;
                const isFrozenRow = freezeRow > 0 && r <= freezeRow;
                const isColSelected = selectedCol === letter;
                const displayBackground = cell?.style?.fill
                  ? `#${cell.style.fill}`
                  : isFrozenRow
                    ? "var(--bg)"
                    : isColSelected
                      ? "var(--accent-soft, rgba(99,102,241,0.06))"
                      : undefined;
                return (
                  <td
                    key={ref}
                    colSpan={merge?.colspan}
                    rowSpan={merge?.rowspan}
                    onClick={() => { if (!isEditing) onSelect(ref); }}
                    onDoubleClick={(e) => {
                      e.stopPropagation();
                      if (!disabled && !isEditing) {
                        onEditCell(ref, cell?.formula ? `=${cell.formula}` : String(cell?.value ?? ""));
                      }
                    }}
                    className={`relative px-2 py-1 text-fg ${isEditing ? "" : "cursor-cell select-none"} ${
                      selected === ref ? "outline outline-2 outline-accent" : ""
                    } ${isFrozenRow ? "sticky" : ""}`}
                    style={{
                      border: "1px solid rgba(128,128,140,0.16)",
                      minWidth: sheet.colWidths?.[letter] ? sheet.colWidths[letter] * 7 : 56,
                      ...(isFrozenRow
                        ? { top: TH_H + (r - 1) * ROW_H, zIndex: 12, height: ROW_H, overflow: "hidden" as const }
                        : {}),
                      fontWeight: cell?.style?.bold ? 600 : undefined,
                      fontStyle: cell?.style?.italic ? "italic" : undefined,
                      textDecoration: cell?.style?.underline
                        ? "underline"
                        : cell?.style?.strike
                          ? "line-through"
                          : undefined,
                      color: cell?.style?.fontColor ? `#${cell.style.fontColor}` : undefined,
                      background: displayBackground,
                      textAlign: (cell?.style?.align as CSSProperties["textAlign"]) ?? undefined,
                      whiteSpace: isFrozenRow ? "nowrap" : cell?.style?.wrap ? "pre-wrap" : "nowrap",
                      fontVariantNumeric: "tabular-nums",
                      ...(virtualize && !isFrozenRow ? { overflow: "hidden" as const } : {}),
                    }}
                    title={cell?.formula ? `=${cell.formula}` : undefined}
                  >
                    {isEditing ? (
                      <input
                        autoFocus
                        value={draft}
                        onChange={(e) => onDraftChange(e.target.value)}
                        onKeyDown={(e) => {
                          e.stopPropagation();
                          if (e.key === "Enter") {
                            e.preventDefault();
                            onCommit(ref, draft);
                          } else if (e.key === "Escape") {
                            e.preventDefault();
                            onCancelEdit();
                          }
                        }}
                        onBlur={onCancelEdit}
                        onClick={(e) => e.stopPropagation()}
                        onDoubleClick={(e) => e.stopPropagation()}
                        className="w-full min-w-[72px] bg-bg text-fg font-mono text-[12px] outline-none ring-2 ring-accent rounded-sm px-1"
                      />
                    ) : (
                      <>
                        {cell?.formula && (
                          <span className="absolute top-0 right-0.5 text-[8px] font-semibold text-accent/80 pointer-events-none">
                            fx
                          </span>
                        )}
                        <span className={cell?.type === "error" ? "text-err" : ""}>{formatCellValue(cell)}</span>
                      </>
                    )}
                  </td>
                );
              })}
            </tr>
          ))}
          {bottomPad > 0 && (
            <tr aria-hidden="true" style={{ height: bottomPad * ROW_H }}>
              <td colSpan={maxCol + 1} style={{ border: "none", padding: 0 }} />
            </tr>
          )}
        </tbody>
      </table>
      <div className="flex items-center gap-1.5 px-1 py-2 text-[10px] text-fg-faint">
        <Table size={11} />
        双击单元格直接编辑；或在 fx 栏输入值（= 开头写公式），回车保存到文件
      </div>
    </div>
  );
}

const PIE_COLORS = ["#5B8DEF", "#7BC47F", "#F2A65A", "#E57373", "#9B7EDE", "#4FC3C3", "#D4A5E5", "#8D9CAD"]; // hex-exempt 图表调色板（xlsx 原生图表）
const CHART_BLUE = "#5B8DEF"; // hex-exempt 图表调色板
const MAX_MINI_POINTS = 12;

// MiniChart 用 SVG 直接渲染图表数据（原生图表对象已嵌入 xlsx，这里是即时视觉反馈，
// 无需外部图表库）。bar 为柱状、line 折线、pie 扇形，其余类型归一为柱状。
function MiniChart({ chartType, labels, values }: { chartType: string; labels: string[]; values: number[] }) {
  const vals = values.slice(0, MAX_MINI_POINTS);
  const lbls = labels.slice(0, MAX_MINI_POINTS);
  if (vals.length === 0) return null;
  const W = 560;
  const H = 150;
  const trunc = (s: string) => (s.length > 6 ? s.slice(0, 5) + "…" : s);

  if (chartType === "pie") {
    const cx = 75;
    const cy = H / 2;
    const r = H / 2 - 8;
    const total = vals.reduce((s, v) => s + Math.abs(v), 0) || 1;
    let acc = -Math.PI / 2;
    const segs = vals.map((v, i) => {
      const a0 = acc;
      acc += (Math.abs(v) / total) * Math.PI * 2;
      const a1 = acc;
      const large = a1 - a0 > Math.PI ? 1 : 0;
      const x0 = cx + r * Math.cos(a0);
      const y0 = cy + r * Math.sin(a0);
      const x1 = cx + r * Math.cos(a1);
      const y1 = cy + r * Math.sin(a1);
      return {
        d: `M ${cx} ${cy} L ${x0.toFixed(2)} ${y0.toFixed(2)} A ${r} ${r} 0 ${large} 1 ${x1.toFixed(2)} ${y1.toFixed(2)} Z`,
        color: PIE_COLORS[i % PIE_COLORS.length],
        label: trunc(lbls[i] ?? String(i + 1)),
        pct: Math.round((Math.abs(v) / total) * 100),
      };
    });
    return (
      <svg viewBox={`0 0 ${W} ${H}`} className="w-full max-w-[560px] mt-1.5" data-testid="mini-chart" role="img">
        {segs.map((s, i) => (
          <path key={i} d={s.d} fill={s.color} stroke="var(--bg)" strokeWidth="1" />
        ))}
        {segs.map((s, i) => (
          <text
            key={`t${i}`}
            x={170 + (i % 2) * 190}
            y={28 + Math.floor(i / 2) * 22}
            fontSize="11"
            fill="var(--fg-dim, #9aa2ad)"
          >
            <tspan fill={s.color}>■</tspan> {s.label} {s.pct}%
          </text>
        ))}
      </svg>
    );
  }

  const padL = 8;
  const padT = 14;
  const padB = 22;
  const plotW = W - padL - 8;
  const plotH = H - padB - padT;
  const max = Math.max(...vals.map(Math.abs), 1e-9);
  const step = plotW / vals.length;

  if (chartType === "line") {
    const pts = vals.map((v, i) => ({
      x: padL + step * (i + 0.5),
      y: padT + plotH - (Math.abs(v) / max) * plotH,
    }));
    return (
      <svg viewBox={`0 0 ${W} ${H}`} className="w-full max-w-[560px] mt-1.5" data-testid="mini-chart" role="img">
        <polyline
          points={pts.map((p) => `${p.x.toFixed(1)},${p.y.toFixed(1)}`).join(" ")}
          fill="none"
          stroke={CHART_BLUE}
          strokeWidth="2"
        />
        {pts.map((p, i) => (
          <g key={i}>
            <circle cx={p.x.toFixed(1)} cy={p.y.toFixed(1)} r="3" fill={CHART_BLUE} />
            <text x={p.x.toFixed(1)} y={(p.y - 6).toFixed(1)} fontSize="9" textAnchor="middle" fill="var(--fg-dim, #9aa2ad)">
              {vals[i]}
            </text>
            {lbls[i] !== undefined && (
              <text x={p.x.toFixed(1)} y={H - 6} fontSize="9" textAnchor="middle" fill="var(--fg-faint, #6b7280)">
                {trunc(lbls[i])}
              </text>
            )}
          </g>
        ))}
      </svg>
    );
  }

  // bar（默认；scatter 归一为柱状展示）
  return (
    <svg viewBox={`0 0 ${W} ${H}`} className="w-full max-w-[560px] mt-1.5" data-testid="mini-chart" role="img">
      {vals.map((v, i) => {
        const h = Math.max(1, (Math.abs(v) / max) * plotH);
        const x = padL + step * i + step * 0.15;
        const w = step * 0.7;
        const y = padT + plotH - h;
        return (
          <g key={i}>
            <rect x={x.toFixed(1)} y={y.toFixed(1)} width={w.toFixed(1)} height={h.toFixed(1)} rx="2" fill={CHART_BLUE} />
            <text x={(x + w / 2).toFixed(1)} y={(y - 4).toFixed(1)} fontSize="9" textAnchor="middle" fill="var(--fg-dim, #9aa2ad)">
              {v}
            </text>
            {lbls[i] !== undefined && (
              <text x={(x + w / 2).toFixed(1)} y={H - 6} fontSize="9" textAnchor="middle" fill="var(--fg-faint, #6b7280)">
                {trunc(lbls[i])}
              </text>
            )}
          </g>
        );
      })}
    </svg>
  );
}
