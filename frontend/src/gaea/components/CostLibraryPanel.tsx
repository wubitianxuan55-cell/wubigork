import { useCallback, useEffect, useMemo, useState } from "react";
import { Modal } from "antd";
import { Clock, CloudUpload, Coins, Pencil, Plus, RefreshCw, Trash2 } from "../icons";
import { app } from "../lib/bridge";
import { useComposerInsertStore } from "../lib/store";
import type { CostSummary, FilePickResult, PriceHistory } from "../lib/types";
import { CostEntryModal } from "./memoryhub/CostEntryModal";
import { CostImportModal } from "./memoryhub/CostImportModal";
import { PriceSourcesPanel } from "./memoryhub/PriceSourcesPanel";
import { useToast } from "./Toast";

const CATEGORIES = ["all", "机械", "材料", "人工", "运输", "检测", "其他"];
const STATUSES = ["all", "现行", "草稿", "已归档"];

// CostLibraryPanel — 办公右侧「成本库」Tab：浏览/搜索成本条目（与记忆中枢
// CostLibrary 同库），一键把结构化单价插入输入框供测算引用；支持新建/编辑/
// 删除/批量操作与文件导入（解析预览后确认入库）。
export function CostLibraryPanel() {
  const [entries, setEntries] = useState<CostSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [query, setQuery] = useState("");
  const [category, setCategory] = useState("all");
  const [status, setStatus] = useState("all");
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [editOpen, setEditOpen] = useState(false);
  const [editing, setEditing] = useState<CostSummary | null>(null);
  const [deleteName, setDeleteName] = useState<string | null>(null);
  const [importFile, setImportFile] = useState<FilePickResult | null>(null);
  const [mode, setMode] = useState<"list" | "sources">("list");
  const [historyOpen, setHistoryOpen] = useState(false);
  const [historyName, setHistoryName] = useState("");
  const [historyRows, setHistoryRows] = useState<PriceHistory[]>([]);
  const toast = useToast();

  const load = useCallback(() => {
    setLoading(true);
    app
      .CostSearch(query, category, status)
      .then((list) => {
        setEntries(list ?? []);
        setSelected((prev) => {
          const names = new Set((list ?? []).map((e) => e.name));
          return new Set([...prev].filter((n) => names.has(n)));
        });
      })
      .catch(() => setEntries([]))
      .finally(() => setLoading(false));
  }, [query, category, status]);

  useEffect(() => {
    const t = setTimeout(load, 250);
    return () => clearTimeout(t);
  }, [load]);

  const priceText = useMemo(() => {
    const fmt = new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 2 });
    return (p: number) => "¥" + fmt.format(p);
  }, []);

  // 一键插入：把单条成本条目转成紧凑结构化行进入输入框（不 dump 整库）。
  const insert = useCallback(
    (e: CostSummary) => {
      const lines = [
        `【成本库】${e.title}`,
        `- name: ${e.name}`,
        `- 分类: ${e.category} | 单价: ${priceText(e.price)}${e.unit ? "/" + e.unit : ""} | 规格: ${e.spec || "-"}`,
        `- 来源: ${e.source || "-"} | 状态: ${e.status || "现行"}`,
      ];
      useComposerInsertStore.getState().requestText(lines.join("\n"));
      toast.show(`已把「${e.title}」插入输入框`, "info");
    },
    [priceText, toast],
  );

  const toggleSelect = useCallback((name: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  }, []);

  const pickImport = useCallback(async () => {
    try {
      const files = await app.PickFiles();
      const f = files?.[0];
      if (f) setImportFile(f);
    } catch {
      // 原生对话框不可用时静默
    }
  }, []);

  const batchDelete = useCallback(async () => {
    if (selected.size === 0) return;
    await Promise.all([...selected].map((n) => app.CostDelete(n).catch(() => {})));
    toast.show(`已删除 ${selected.size} 条`, "info");
    setSelected(new Set());
    load();
  }, [selected, toast, load]);

  const batchStatus = useCallback(
    async (next: string) => {
      if (selected.size === 0 || !next) return;
      for (const name of selected) {
        const e = await app.CostGet(name).catch(() => null);
        if (e) await app.CostSave({ ...e, status: next }).catch(() => {});
      }
      toast.show(`已把 ${selected.size} 条改为「${next}」`, "info");
      setSelected(new Set());
      load();
    },
    [selected, toast, load],
  );

  const handleDelete = useCallback(async () => {
    if (!deleteName) return;
    await app.CostDelete(deleteName).catch(() => {});
    setDeleteName(null);
    load();
  }, [deleteName, load]);

  const openHistory = useCallback(async (name: string) => {
    setHistoryName(name);
    setHistoryRows([]);
    setHistoryOpen(true);
    const rows = await app.PriceHistory(name).catch(() => [] as PriceHistory[]);
    setHistoryRows(rows ?? []);
  }, []);

  if (mode === "sources") {
    return (
      <div className="flex flex-col h-full">
        <div className="shrink-0 flex items-center gap-1 px-3 pt-2">
          {(["list", "sources"] as const).map((m) => (
            <button
              key={m}
              onClick={() => setMode(m)}
              className={`px-2.5 h-6 rounded-full text-[10.5px] transition-colors ${
                mode === m ? "bg-accent text-white" : "bg-bg-elev text-fg-faint hover:text-fg border border-border"
              }`}
            >
              {m === "list" ? "成本库" : "价格源"}
            </button>
          ))}
        </div>
        <div className="flex-1 min-h-0">
          <PriceSourcesPanel onChanged={load} />
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full text-fg-dim text-xs">
      <div className="flex items-center justify-between px-3 py-2 border-b border-border-soft">
        <span className="flex items-center gap-1.5 font-semibold text-fg text-sm">
          <Coins size={13} className="text-amber-400" />
          成本库
        </span>
        <div className="flex items-center gap-1.5">
          <button
            className="flex items-center justify-center w-6 h-6 border-0 bg-transparent text-sky-400 cursor-pointer hover:text-sky-300 hover:bg-bg-soft rounded"
            onClick={() => setMode("sources")}
            title="价格源：定时抓取价格更新"
          >
            <CloudUpload size={12} />
          </button>
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="搜索名称/规格/来源…"
            className="w-36 px-2.5 h-7 rounded-lg border border-border bg-bg text-fg text-[11.5px] placeholder:text-fg-faint outline-none focus:border-accent transition-colors"
          />
          <button
            className="flex items-center justify-center w-6 h-6 border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft rounded"
            onClick={load}
            title="刷新成本库"
          >
            <RefreshCw size={12} />
          </button>
          <button
            className="flex items-center justify-center w-6 h-6 border-0 bg-transparent text-amber-400 cursor-pointer hover:text-amber-300 hover:bg-bg-soft rounded"
            onClick={() => void pickImport()}
            title="导入 xlsx/csv 报价单或测算表"
          >
            <CloudUpload size={12} />
          </button>
          <button
            className="flex items-center justify-center w-6 h-6 border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft rounded"
            onClick={() => {
              setEditing(null);
              setEditOpen(true);
            }}
            title="新建成本条目"
          >
            <Plus size={12} />
          </button>
          {entries.length > 0 && (
            <span className="text-[10px] text-fg-faint border border-border-soft/60 rounded-full px-1.5 py-px">
              {entries.length}
            </span>
          )}
        </div>
      </div>

      <div className="shrink-0 flex items-center gap-1 px-3 py-1.5 flex-wrap">
        {CATEGORIES.map((c) => (
          <button
            key={c}
            onClick={() => setCategory(c)}
            className={`px-2 h-6 rounded-full text-[10.5px] transition-colors ${
              category === c ? "bg-accent text-white" : "bg-bg-elev text-fg-faint hover:text-fg border border-border"
            }`}
          >
            {c === "all" ? "全部分类" : c}
          </button>
        ))}
        <span className="w-px h-4 bg-border-soft mx-1" />
        {STATUSES.map((s) => (
          <button
            key={s}
            onClick={() => setStatus(s)}
            className={`px-2 h-6 rounded-full text-[10.5px] transition-colors ${
              status === s ? "bg-accent text-white" : "bg-bg-elev text-fg-faint hover:text-fg border border-border"
            }`}
          >
            {s === "all" ? "全部状态" : s}
          </button>
        ))}
        {selected.size > 0 && (
          <span className="ml-auto flex items-center gap-1.5">
            <span className="text-amber-300 text-[10.5px]">已选 {selected.size}</span>
            <select
              value=""
              onChange={(e) => {
                if (e.target.value) void batchStatus(e.target.value);
              }}
              className="px-1.5 h-6 rounded-md bg-bg-elev text-fg-dim text-[10.5px] border border-border outline-none"
            >
              <option value="" disabled>改状态…</option>
              {["现行", "草稿", "已归档"].map((s) => (
                <option key={s} value={s}>{s}</option>
              ))}
            </select>
            <button
              className="px-2 h-6 rounded-md bg-red-500/15 text-red-400 text-[10.5px] cursor-pointer hover:bg-red-500/25 transition-colors"
              onClick={() => void batchDelete()}
            >
              批量删除
            </button>
          </span>
        )}
      </div>

      <div className="flex-1 min-h-0 overflow-y-auto p-2 flex flex-col gap-1">
        {loading ? (
          <div className="py-8 text-center text-fg-faint text-[11px]">加载中…</div>
        ) : entries.length === 0 ? (
          <div className="flex flex-col items-center justify-center flex-1 gap-2 px-6 text-center text-fg-faint/50">
            <Coins size={24} className="opacity-40" />
            <span className="text-[11px] leading-relaxed">
              暂无成本条目
              <br />
              可导入报价单，或测算完成后在产物上「沉淀到成本库」
            </span>
          </div>
        ) : (
          entries.map((e) => (
            <div key={e.name} className="p-2 rounded-lg border border-border-soft/70 bg-bg-soft/30 hover:border-accent/30 hover:bg-bg-soft/60 transition-colors">
              <div className="flex items-center gap-1.5">
                <input
                  type="checkbox"
                  checked={selected.has(e.name)}
                  onChange={() => toggleSelect(e.name)}
                  title="多选（批量删除/改状态）"
                  className="shrink-0"
                />
                <Coins size={12} className="text-amber-400 shrink-0" />
                <span className="truncate text-fg text-[12px] font-medium">{e.title}</span>
                {e.spec && <span className="text-fg-faint text-[10px] shrink-0">{e.spec}</span>}
                <span className="px-1.5 py-px rounded bg-bg-elev text-fg-faint text-[9.5px] shrink-0">{e.category}</span>
                {e.status !== "现行" && (
                  <span className="px-1.5 py-px rounded bg-bg-elev text-fg-faint text-[9.5px] shrink-0">{e.status}</span>
                )}
                <span className="ml-auto shrink-0 text-fg text-[12px] font-semibold text-amber-300">
                  {priceText(e.price)}
                  {e.unit && <span className="text-fg-faint text-[10px] font-normal">/{e.unit}</span>}
                </span>
                <button
                  className="shrink-0 px-2 h-6 rounded-md bg-accent/15 text-accent text-[11px] cursor-pointer hover:bg-accent/25 transition-colors"
                  onClick={() => insert(e)}
                  title="把该成本条目作为结构化上下文插入输入框"
                >
                  引用
                </button>
                <button
                  className="shrink-0 w-6 h-6 inline-flex items-center justify-center rounded text-fg-faint hover:text-fg hover:bg-bg-elev"
                  onClick={() => {
                    setEditing(e);
                    setEditOpen(true);
                  }}
                  title="编辑"
                >
                  <Pencil size={11} />
                </button>
                <button
                  className="shrink-0 w-6 h-6 inline-flex items-center justify-center rounded text-fg-faint hover:text-red-400 hover:bg-bg-elev"
                  onClick={() => setDeleteName(e.name)}
                  title="删除"
                >
                  <Trash2 size={11} />
                </button>
                <button
                  className="shrink-0 w-6 h-6 inline-flex items-center justify-center rounded text-fg-faint hover:text-fg hover:bg-bg-elev"
                  onClick={() => void openHistory(e.name)}
                  title="价格历史"
                >
                  <Clock size={11} />
                </button>
              </div>
              {e.source && <div className="mt-1 pl-[18px] text-fg-faint text-[10px]">来源：{e.source}</div>}
            </div>
          ))
        )}
      </div>

      <CostEntryModal
        open={editOpen}
        editing={editing}
        onClose={() => setEditOpen(false)}
        onSaved={() => {
          setEditOpen(false);
          load();
        }}
      />
      <CostImportModal
        open={!!importFile}
        path={importFile?.path ?? ""}
        fileName={importFile?.name ?? ""}
        onClose={() => setImportFile(null)}
        onImported={load}
      />
      <Modal
        title="删除成本"
        open={!!deleteName}
        onCancel={() => setDeleteName(null)}
        onOk={() => void handleDelete()}
        okText="删除"
        okButtonProps={{ danger: true }}
        cancelText="取消"
      >
        <p className="text-[13px] text-fg-dim">确定删除成本条目「{deleteName}」吗？</p>
      </Modal>
      <Modal
        title={`价格历史：${historyName}`}
        open={historyOpen}
        onCancel={() => setHistoryOpen(false)}
        footer={null}
        width={520}
      >
        <div className="space-y-1.5 max-h-[46vh] overflow-auto">
          {historyRows.length === 0 ? (
            <div className="py-6 text-center text-fg-faint text-[12px]">暂无价格历史</div>
          ) : (
            historyRows.map((h, i) => (
              <div key={i} className="flex items-center gap-2 px-2 py-1.5 rounded-lg bg-bg-soft/40 text-[12px]">
                <span className="font-semibold text-amber-300">¥{h.price}</span>
                {h.unit && <span className="text-fg-faint">/{h.unit}</span>}
                <span className="ml-auto text-fg-faint">
                  {h.source}{h.period ? `（期 ${h.period}）` : ""}
                </span>
                <span className="text-fg-faint text-[10.5px]">
                  {h.fetchedAt ? new Date(h.fetchedAt).toLocaleString("zh-CN", { hour12: false }) : "手动录入"}
                </span>
              </div>
            ))
          )}
        </div>
      </Modal>
    </div>
  );
}
