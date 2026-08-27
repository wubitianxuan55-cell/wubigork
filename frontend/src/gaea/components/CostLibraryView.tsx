import { memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useDebouncedValue } from "../hooks/useDebouncedValue";
import { Input, Modal, message } from "antd";
import {
  BarChart3, ChevronDown, ChevronRight, Clock, CloudUpload, Coins, FolderPlus, List,
  Pencil, Plus, RefreshCw, Table, Trash2,
} from "../icons";
import { app } from "../lib/bridge";
import type { CostCategory, CostSummary, FilePickResult, PriceHistory } from "../lib/types";
import { EmptyState } from "./EmptyState";
import { CostEntryModal } from "./memoryhub/CostEntryModal";
import { CostImportModal } from "./memoryhub/CostImportModal";
import { CostCompareModal } from "./memoryhub/CostCompareModal";

const STATUSES = ["现行", "草稿", "已归档"];

interface CostLibraryViewProps {
  /** 办公右侧窄面板：隐藏分类树侧栏，改用路径下拉 + 紧凑行距。 */
  compact?: boolean;
  /** 办公侧提供时显示「插入」按钮，一键把单价写入输入框。 */
  onInsert?: (e: CostSummary) => void;
}

type SortKey = "title" | "price" | "updatedAt";

/**
 * CostLibraryView 成本库主视图（记忆中枢与办公侧共用）：
 * - 左侧多级分类树（可增删改分类、子树过滤、节点计数）；
 * - 右侧 列表 / 表格 双视图，表格支持排序与多选批量操作；
 * - 条目按「分类路径」多级保存（一级/二级/…/叶子）。
 */
export function CostLibraryView({ compact = false, onInsert }: CostLibraryViewProps) {
  const [entries, setEntries] = useState<CostSummary[]>([]);
  const [categories, setCategories] = useState<CostCategory[]>([]);
  const [loading, setLoading] = useState(true);
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState("all");
  const [selectedPath, setSelectedPath] = useState("");
  const [view, setView] = useState<"list" | "table">("list");
  const [expanded, setExpanded] = useState<Set<number>>(new Set());
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<CostSummary | null>(null);
  const [deleteName, setDeleteName] = useState<string | null>(null);
  const [importFile, setImportFile] = useState<FilePickResult | null>(null);
  const [catModal, setCatModal] = useState<{ mode: "create" | "rename"; parentId: number; node: CostCategory | null } | null>(null);
  const [catName, setCatName] = useState("");
  const [historyOpen, setHistoryOpen] = useState(false);
  const [historyName, setHistoryName] = useState("");
  const [historyRows, setHistoryRows] = useState<PriceHistory[]>([]);
  const [compare, setCompare] = useState<{ name: string; title: string; price: number } | null>(null);
  const [sortKey, setSortKey] = useState<SortKey | null>(null);
  const [sortDir, setSortDir] = useState<1 | -1>(1);
  const treeInited = useRef(false);

  // 分类树 → 节点/路径索引（多级路径：一级/二级/…/叶子）。
  const { nodeById, pathById, allPaths } = useMemo(() => {
    const nodeById = new Map<number, CostCategory>();
    const pathById = new Map<number, string>();
    const allPaths: string[] = [];
    const walk = (nodes: CostCategory[] | undefined, prefix: string) => {
      for (const n of nodes ?? []) {
        nodeById.set(n.id, n);
        const p = prefix ? `${prefix}/${n.name}` : n.name;
        pathById.set(n.id, p);
        allPaths.push(p);
        walk(n.children, p);
      }
    };
    walk(categories, "");
    return { nodeById, pathById, allPaths };
  }, [categories]);

  // 高频搜索防抖：输入框值即时更新（query），CostSearch 消费防抖后的值（250ms；清空即时生效）
  const debouncedQuery = useDebouncedValue(query, 250);

  const load = useCallback(() => {
    setLoading(true);
    app
      .CostSearch(debouncedQuery, selectedPath, status)
      .then((list) => {
        const items = list ?? [];
        setEntries(items);
        setSelected((prev) => new Set([...prev].filter((n) => items.some((e) => e.name === n))));
      })
      .catch(() => setEntries([]))
      .finally(() => setLoading(false));
  }, [debouncedQuery, selectedPath, status]);

  const loadCategories = useCallback(() => {
    app
      .CostCategories()
      .then((tree) => {
        setCategories(tree ?? []);
        // 首次加载默认展开含子节点的分类，让多级结构立即可见。
        if (!treeInited.current) {
          treeInited.current = true;
          const ids = new Set<number>();
          const walk = (nodes: CostCategory[]) => {
            for (const n of nodes ?? []) {
              if (n.children?.length) ids.add(n.id);
              walk(n.children ?? []);
            }
          };
          walk(tree ?? []);
          setExpanded(ids);
        }
      })
      .catch(() => {});
  }, []);

  // 防抖已由 useDebouncedValue 承担（query 变化延迟 250ms 后触发 load；清空立即触发）
  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    loadCategories();
  }, [loadCategories]);

  // 分类被改名/删除后，失效的选中路径回退到「全部」。
  useEffect(() => {
    if (selectedPath && !allPaths.includes(selectedPath)) setSelectedPath("");
  }, [allPaths, selectedPath]);

  const priceText = useMemo(() => {
    const fmt = new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 2 });
    return (p: number) => "¥" + fmt.format(p);
  }, []);

  const sorted = useMemo(() => {
    if (!sortKey) return entries;
    const arr = [...entries];
    arr.sort((a, b) => {
      let r = 0;
      if (sortKey === "price") r = a.price - b.price;
      else if (sortKey === "title") r = a.title.localeCompare(b.title, "zh-CN");
      else r = (a.updatedAt ?? "").localeCompare(b.updatedAt ?? "");
      return r * sortDir;
    });
    return arr;
  }, [entries, sortKey, sortDir]);

  const toggleSort = (k: SortKey) => {
    if (sortKey === k) setSortDir((d) => (d === 1 ? -1 : 1));
    else {
      setSortKey(k);
      setSortDir(1);
    }
  };

  const openCreate = () => {
    setEditing(null);
    setModalOpen(true);
  };
  const openEdit = useCallback((s: CostSummary) => {
    setEditing(s);
    setModalOpen(true);
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
  const handleDelete = async () => {
    if (!deleteName) return;
    try {
      await app.CostDelete(deleteName);
      setDeleteName(null);
      load();
    } catch (e: unknown) {
      message.error((e as Error)?.message ?? "删除失败");
    }
  };
  const toggleSelect = useCallback((name: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  }, []);
  const batchDelete = async () => {
    if (selected.size === 0) return;
    await Promise.all([...selected].map((n) => app.CostDelete(n).catch(() => {})));
    message.info(`已删除 ${selected.size} 条`);
    setSelected(new Set());
    load();
  };
  const batchStatus = async (next: string) => {
    if (selected.size === 0 || !next) return;
    for (const name of selected) {
      const e = await app.CostGet(name).catch(() => null);
      if (e) await app.CostSave({ ...e, status: next }).catch(() => {});
    }
    message.info(`已把 ${selected.size} 条改为「${next}」`);
    setSelected(new Set());
    load();
  };
  const openHistory = useCallback(async (name: string) => {
    setHistoryName(name);
    setHistoryRows([]);
    setHistoryOpen(true);
    const rows = await app.PriceHistory(name).catch(() => [] as PriceHistory[]);
    setHistoryRows(rows ?? []);
  }, []);
  const openCompare = useCallback((e: CostSummary) => setCompare({ name: e.name, title: e.title, price: e.price }), []);

  // ── 分类管理 ──
  const openCatModal = (mode: "create" | "rename", parentId: number, node: CostCategory | null) => {
    setCatModal({ mode, parentId, node });
    setCatName(node?.name ?? "");
  };
  const saveCategory = async () => {
    if (!catModal) return;
    const name = catName.trim();
    if (!name) {
      message.warning("请输入分类名称");
      return;
    }
    try {
      await app.CostCategorySave(catModal.parentId, name, 0, catModal.node?.id ?? 0);
      setCatModal(null);
      loadCategories();
      load();
    } catch (e: unknown) {
      message.error((e as Error)?.message ?? "保存分类失败");
    }
  };
  const deleteCategory = async (node: CostCategory) => {
    try {
      await app.CostCategoryDelete(node.id);
      loadCategories();
      load();
    } catch (e: unknown) {
      message.error((e as Error)?.message ?? "删除分类失败");
    }
  };

  const breadcrumb = selectedPath.split("/").filter(Boolean);

  return (
    <div className={`flex h-full min-h-0 ${compact ? "text-[11.5px]" : "text-[12.5px]"}`}>
      {/* 左：多级分类树（办公窄面板隐藏，改用下拉） */}
      {!compact && (
        <aside className="w-52 shrink-0 flex flex-col min-h-0 border-r border-border-soft/70">
          <div className="shrink-0 flex items-center gap-1.5 px-3 h-9 border-b border-border-soft/60">
            <Coins size={13} className="text-amber-400" />
            <span className="text-fg text-[12.5px] font-semibold">分类</span>
            <button
              className="ml-auto inline-flex items-center justify-center w-6 h-6 rounded text-fg-faint hover:text-accent hover:bg-bg-elev transition-colors"
              title="新建一级分类"
              onClick={() => openCatModal("create", 0, null)}
            >
              <FolderPlus size={13} />
            </button>
          </div>
          <div className="flex-1 min-h-0 overflow-y-auto py-1.5 px-1.5 space-y-px">
            <div
              className={`flex items-center gap-1.5 h-7 px-2 rounded-md cursor-pointer transition-colors ${
                selectedPath === "" ? "bg-accent/15 text-accent" : "text-fg-dim hover:bg-bg-soft"
              }`}
              onClick={() => setSelectedPath("")}
            >
              <ChevronDown size={12} className="opacity-0" />
              <span className="text-[12px] font-medium">全部条目</span>
              <span className="ml-auto text-[10px] text-fg-faint tabular-nums">{entries.length || ""}</span>
            </div>
            {categories.map((n) => (
              <CategoryNode
                key={n.id}
                node={n}
                depth={0}
                selectedPath={selectedPath}
                pathById={pathById}
                expanded={expanded}
                onToggle={(id) =>
                  setExpanded((prev) => {
                    const next = new Set(prev);
                    if (next.has(id)) next.delete(id);
                    else next.add(id);
                    return next;
                  })
                }
                onSelect={setSelectedPath}
                onAddChild={(node) => openCatModal("create", node.id, null)}
                onRename={(node) => openCatModal("rename", node.parentId, node)}
                onDelete={(node) => deleteCategory(node)}
              />
            ))}
          </div>
        </aside>
      )}

      {/* 右：工具条 + 列表/表格 */}
      <div className="flex-1 min-w-0 flex flex-col">
        {/* 工具条 */}
        <div className={`shrink-0 flex items-center gap-2 ${compact ? "px-3 py-1.5" : "px-4 pt-2.5 pb-1.5"}`}>
          {compact ? (
            <Coins size={13} className="text-amber-400 shrink-0" />
          ) : (
            <>
              <div className="text-fg text-[13px] font-medium">成本库</div>
              <span className="text-fg-faint text-[11px]">综合单价一级 · 人材机二级 · 按专业/分部分类</span>
            </>
          )}
          <div className="ml-auto flex items-center gap-1.5">
            {!compact && (
              <div className="flex items-center rounded-lg border border-border overflow-hidden">
                <button
                  className={`inline-flex items-center gap-1 px-2 h-7 text-[11px] transition-colors ${
                    view === "list" ? "bg-accent text-white" : "text-fg-faint hover:text-fg"
                  }`}
                  onClick={() => setView("list")}
                  title="列表视图"
                >
                  <List size={12} />
                </button>
                <button
                  className={`inline-flex items-center gap-1 px-2 h-7 text-[11px] transition-colors ${
                    view === "table" ? "bg-accent text-white" : "text-fg-faint hover:text-fg"
                  }`}
                  onClick={() => setView("table")}
                  title="表格视图"
                >
                  <Table size={12} />
                </button>
              </div>
            )}
            <input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="搜索名称/规格/来源…"
              className={`px-2.5 h-7 rounded-lg border border-border bg-bg text-fg placeholder:text-fg-faint outline-none focus:border-accent transition-colors ${
                compact ? "w-32 text-[11px]" : "w-44 text-[12px]"
              }`}
            />
            <button
              className="inline-flex items-center justify-center w-7 h-7 rounded-lg border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors"
              onClick={load}
              title="刷新"
            >
              <RefreshCw size={12} />
            </button>
            <button
              className="inline-flex items-center justify-center w-7 h-7 rounded-lg border border-border text-amber-400 hover:text-amber-300 hover:bg-bg-soft transition-colors"
              onClick={() => void pickImport()}
              title="导入 xlsx/csv 报价单或测算表"
            >
              <CloudUpload size={12} />
            </button>
            <button
              className="inline-flex items-center gap-1 px-2.5 h-7 rounded-lg bg-accent text-white text-[11.5px] hover:opacity-90 transition-opacity"
              onClick={openCreate}
            >
              <Plus size={12} /> 新建
            </button>
          </div>
        </div>

        {/* 筛选行：分类面包屑/下拉 + 状态 + 计数 */}
        <div className={`shrink-0 flex items-center gap-2 flex-wrap ${compact ? "px-3 pb-1" : "px-4 pb-2"}`}>
          {compact ? (
            <select
              value={selectedPath}
              onChange={(e) => setSelectedPath(e.target.value)}
              className="h-6 px-1.5 rounded-md bg-bg-elev text-fg-dim text-[10.5px] border border-border outline-none max-w-[150px]"
              title="分类路径（含子分类）"
            >
              <option value="">全部分类</option>
              {allPaths.map((p) => (
                <option key={p} value={p}>{p}</option>
              ))}
            </select>
          ) : (
            <div className="flex items-center gap-1 min-w-0">
              <span
                className={`px-2 h-6 rounded-full text-[11px] transition-colors cursor-pointer ${
                  selectedPath === "" ? "bg-accent text-white" : "bg-bg-elev text-fg-faint hover:text-fg border border-border"
                }`}
                onClick={() => setSelectedPath("")}
              >
                全部
              </span>
              {breadcrumb.map((seg, i) => {
                const path = breadcrumb.slice(0, i + 1).join("/");
                return (
                  <span
                    key={path}
                    className={`px-2 h-6 rounded-full text-[11px] transition-colors cursor-pointer ${
                      selectedPath === path ? "bg-accent text-white" : "bg-bg-elev text-fg-faint hover:text-fg border border-border"
                    }`}
                    onClick={() => setSelectedPath(path)}
                  >
                    {seg}
                  </span>
                );
              })}
            </div>
          )}
          <div className="flex items-center gap-1">
            {STATUSES.map((s) => (
              <button
                key={s}
                onClick={() => setStatus(status === s ? "all" : s)}
                className={`px-2 h-6 rounded-full text-[11px] transition-colors ${
                  status === s ? "bg-accent text-white" : "bg-bg-elev text-fg-faint hover:text-fg border border-border"
                }`}
              >
                {s}
              </button>
            ))}
          </div>
          <span className="ml-auto text-fg-faint text-[11px] tabular-nums">{entries.length} 条</span>
          {selected.size > 0 && (
            <span className="flex items-center gap-1.5">
              <span className="text-amber-300 text-[11px]">已选 {selected.size}</span>
              <select
                value=""
                onChange={(e) => {
                  if (e.target.value) void batchStatus(e.target.value);
                }}
                className="px-1.5 h-6 rounded-md bg-bg-elev text-fg-dim text-[11px] border border-border outline-none"
              >
                <option value="" disabled>改状态…</option>
                {STATUSES.map((s) => (
                  <option key={s} value={s}>{s}</option>
                ))}
              </select>
              <button
                className="px-2 h-6 rounded-md bg-red-500/15 text-red-400 text-[11px] cursor-pointer hover:bg-red-500/25 transition-colors"
                onClick={() => void batchDelete()}
              >
                批量删除
              </button>
            </span>
          )}
        </div>

        {/* 内容区 */}
        <div className="flex-1 min-h-0 overflow-y-auto">
          {loading ? (
            <div className="p-4 space-y-2 animate-pulse">
              {Array.from({ length: compact ? 4 : 6 }).map((_, i) => (
                <div key={i} className="h-11 rounded-lg bg-bg-elev/60" />
              ))}
            </div>
          ) : sorted.length === 0 ? (
            <div className="h-full flex items-center justify-center">
              <EmptyState message="暂无成本条目 — 新建、导入报价单，或测算完成后沉淀到成本库" />
            </div>
          ) : view === "table" && !compact ? (
            <TableView
              rows={sorted}
              selected={selected}
              toggleSelect={toggleSelect}
              sortKey={sortKey}
              sortDir={sortDir}
              toggleSort={toggleSort}
              priceText={priceText}
              onSelectAll={setSelected}
              onEdit={openEdit}
              onDelete={setDeleteName}
              onHistory={openHistory}
              onCompare={openCompare}
              onInsert={onInsert}
            />
          ) : (
            <ListView
              rows={sorted}
              selected={selected}
              toggleSelect={toggleSelect}
              priceText={priceText}
              onEdit={openEdit}
              onDelete={setDeleteName}
              onHistory={openHistory}
              onCompare={openCompare}
              onInsert={onInsert}
              compact={compact}
            />
          )}
        </div>
      </div>

      {/* 新建/编辑条目 */}
      <CostEntryModal
        open={modalOpen}
        editing={editing}
        onClose={() => setModalOpen(false)}
        onSaved={() => {
          setModalOpen(false);
          load();
          loadCategories();
        }}
      />

      {/* 导入文件 → 解析预览 → 确认入库 */}
      <CostImportModal
        open={!!importFile}
        path={importFile?.path ?? ""}
        fileName={importFile?.name ?? ""}
        onClose={() => setImportFile(null)}
        onImported={() => {
          load();
          loadCategories();
        }}
      />

      {/* 供应商比价：跨来源对比现价跳幅 */}
      <CostCompareModal
        open={!!compare}
        name={compare?.name ?? ""}
        title={compare?.title ?? ""}
        currentPrice={compare?.price}
        onClose={() => setCompare(null)}
      />

      {/* 删除条目确认 */}
      <Modal
        title="删除成本"
        open={!!deleteName}
        onCancel={() => setDeleteName(null)}
        onOk={handleDelete}
        okText="删除"
        destroyOnHidden
        transitionName=""
        maskTransitionName=""
        okButtonProps={{ danger: true }}
        cancelText="取消"
        width={440}
      >
        <p className="text-[13px] text-fg-dim">确定删除成本条目「{deleteName}」吗？</p>
      </Modal>

      {/* 分类 新建/重命名 */}
      <Modal
        title={catModal?.mode === "rename" ? "重命名分类" : "新建分类"}
        open={!!catModal}
        onCancel={() => setCatModal(null)}
        onOk={saveCategory}
        okText={catModal?.mode === "rename" ? "保存" : "创建"}
        destroyOnHidden
        transitionName=""
        maskTransitionName=""
        cancelText="取消"
        width={420}
      >
        <div className="space-y-2 py-1">
          {catModal && catModal.mode === "create" && catModal.parentId > 0 && (
            <div className="text-[11.5px] text-fg-faint">
              父分类：<span className="text-fg">{pathById.get(catModal.parentId) ?? "—"}</span>
            </div>
          )}
          <Input
            value={catName}
            onChange={(e) => setCatName(e.target.value)}
            placeholder="分类名称，如 钢材 / 桩基机械"
            autoFocus
            onPressEnter={saveCategory}
            className="!text-[13px]"
          />
        </div>
      </Modal>

      {/* 价格历史 */}
      <Modal
        title={`价格历史：${historyName}`}
        open={historyOpen}
        onCancel={() => setHistoryOpen(false)}
        footer={null}
        destroyOnHidden
        transitionName=""
        maskTransitionName=""
        width={520}
      >
        {historyRows.length === 0 ? (
          <div className="py-6 text-center text-fg-faint text-[12px]">暂无价格历史（保存时内容变化会自动留档）</div>
        ) : (
          <div className="space-y-1.5 max-h-[46vh] overflow-auto">
            {historyRows.map((h, i) => (
              <div key={i} className="flex items-center gap-2 px-2 py-1.5 rounded-lg bg-bg-soft/40 text-[12px]">
                <Clock size={12} className="text-fg-faint shrink-0" />
                <span className="text-fg font-medium tabular-nums">{priceText(h.price)}{h.unit ? `/${h.unit}` : ""}</span>
                {h.period && <span className="text-fg-faint">{h.period}</span>}
                {h.source && <span className="text-fg-faint truncate">来源：{h.source}</span>}
                <span className="ml-auto text-fg-faint text-[10.5px] shrink-0">
                  {h.fetchedAt ? new Date(h.fetchedAt).toLocaleString("zh-CN", { hour12: false }) : ""}
                </span>
              </div>
            ))}
          </div>
        )}
      </Modal>
    </div>
  );
}

// ── 分类树节点（递归）──
function CategoryNode({
  node,
  depth,
  selectedPath,
  pathById,
  expanded,
  onToggle,
  onSelect,
  onAddChild,
  onRename,
  onDelete,
}: {
  node: CostCategory;
  depth: number;
  selectedPath: string;
  pathById: Map<number, string>;
  expanded: Set<number>;
  onToggle: (id: number) => void;
  onSelect: (path: string) => void;
  onAddChild: (node: CostCategory) => void;
  onRename: (node: CostCategory) => void;
  onDelete: (node: CostCategory) => void;
}) {
  const path = pathById.get(node.id) ?? "";
  const hasChildren = !!node.children?.length;
  const isOpen = expanded.has(node.id);
  const active = selectedPath === path;
  return (
    <div>
      <div
        className={`group flex items-center gap-1 h-7 pr-1 rounded-md cursor-pointer transition-colors ${
          active ? "bg-accent/15 text-accent" : "text-fg-dim hover:bg-bg-soft"
        }`}
        style={{ paddingLeft: 6 + depth * 14 }}
        onClick={() => onSelect(path)}
        title={path}
      >
        <button
          className={`w-3.5 h-3.5 inline-flex items-center justify-center shrink-0 text-fg-faint ${hasChildren ? "" : "invisible"}`}
          onClick={(e) => {
            e.stopPropagation();
            onToggle(node.id);
          }}
        >
          {isOpen ? <ChevronDown size={11} /> : <ChevronRight size={11} />}
        </button>
        <span className="truncate text-[12px]">{node.name}</span>
        {node.count > 0 && (
          <span className="ml-auto shrink-0 text-[10px] text-fg-faint tabular-nums bg-bg-elev rounded-full px-1.5">{node.count}</span>
        )}
        <span
          className="hidden group-hover:flex items-center gap-0.5 shrink-0"
          onClick={(e) => e.stopPropagation()}
        >
          <button className="w-5 h-5 inline-flex items-center justify-center rounded text-fg-faint hover:text-accent hover:bg-bg-elev" title="添加子分类" onClick={() => onAddChild(node)}>
            <Plus size={11} />
          </button>
          <button className="w-5 h-5 inline-flex items-center justify-center rounded text-fg-faint hover:text-fg hover:bg-bg-elev" title="重命名" onClick={() => onRename(node)}>
            <Pencil size={11} />
          </button>
          <button className="w-5 h-5 inline-flex items-center justify-center rounded text-fg-faint hover:text-red-400 hover:bg-bg-elev" title="删除" onClick={() => onDelete(node)}>
            <Trash2 size={11} />
          </button>
        </span>
      </div>
      {hasChildren && isOpen && (
        <div>
          {node.children!.map((child) => (
            <CategoryNode
              key={child.id}
              node={child}
              depth={depth + 1}
              selectedPath={selectedPath}
              pathById={pathById}
              expanded={expanded}
              onToggle={onToggle}
              onSelect={onSelect}
              onAddChild={onAddChild}
              onRename={onRename}
              onDelete={onDelete}
            />
          ))}
        </div>
      )}
    </div>
  );
}

interface CostRowProps {
  row: CostSummary;
  selected: boolean;
  compact: boolean;
  priceText: (p: number) => string;
  onToggleSelect: (name: string) => void;
  onEdit: (e: CostSummary) => void;
  onDelete: (name: string) => void;
  onHistory: (name: string) => void;
  onCompare: (e: CostSummary) => void;
  onInsert?: (e: CostSummary) => void;
}

interface ListViewProps {
  rows: CostSummary[];
  selected: Set<string>;
  toggleSelect: (name: string) => void;
  priceText: (p: number) => string;
  onEdit: (e: CostSummary) => void;
  onDelete: (name: string) => void;
  onHistory: (name: string) => void;
  onCompare: (e: CostSummary) => void;
  onInsert?: (e: CostSummary) => void;
  compact: boolean;
}

interface TableRowProps {
  row: CostSummary;
  selected: boolean;
  priceText: (p: number) => string;
  onToggleSelect: (name: string) => void;
  onEdit: (e: CostSummary) => void;
  onDelete: (name: string) => void;
  onHistory: (name: string) => void;
  onCompare: (e: CostSummary) => void;
  onInsert?: (e: CostSummary) => void;
}

interface TableViewProps {
  rows: CostSummary[];
  selected: Set<string>;
  toggleSelect: (name: string) => void;
  sortKey: SortKey | null;
  sortDir: 1 | -1;
  toggleSort: (k: SortKey) => void;
  priceText: (p: number) => string;
  onSelectAll: (next: Set<string>) => void;
  onEdit: (e: CostSummary) => void;
  onDelete: (name: string) => void;
  onHistory: (name: string) => void;
  onCompare: (e: CostSummary) => void;
  onInsert?: (e: CostSummary) => void;
}

// ── 列表项（React.memo：props 稳定（useCallback 回调）时跳过重渲染）──
export const CostRow = memo(function CostRow({
  row: e,
  selected,
  compact,
  priceText,
  onToggleSelect,
  onEdit,
  onDelete,
  onHistory,
  onCompare,
  onInsert,
}: CostRowProps) {
  return (
    <div
      key={e.name}
      className={`group rounded-lg border transition-colors ${
        selected
          ? "border-accent/50 bg-accent/10"
          : "border-border/70 bg-bg-soft/40 hover:border-accent/30 hover:bg-bg-soft/70"
      } ${compact ? "p-2" : "p-2.5"}`}
    >
      <div className="flex items-center gap-2">
        <input
          type="checkbox"
          checked={selected}
          onChange={() => onToggleSelect(e.name)}
          className="accent-[var(--accent)] shrink-0"
          title="多选"
        />
        <Coins size={14} className="text-amber-400 shrink-0" />
        <span className={`text-fg font-medium truncate ${compact ? "text-[12px]" : "text-[13px]"}`}>{e.title}</span>
        <button
          className="w-5 h-5 shrink-0 inline-flex items-center justify-center rounded text-fg-faint opacity-0 group-hover:opacity-100 hover:text-sky-400 hover:bg-bg-elev transition-opacity"
          onClick={() => onCompare(e)}
          title="比价"
        >
          <BarChart3 size={11} />
        </button>
        {e.spec && <span className="text-fg-faint text-[11px] shrink-0 truncate max-w-[140px]">{e.spec}</span>}
        {e.categoryPath && (
          <span className="px-1.5 py-0.5 rounded bg-bg-elev text-fg-faint text-[10px] shrink-0 truncate max-w-[160px]" title={e.categoryPath}>
            {e.categoryPath}
          </span>
        )}
        {e.status !== "现行" && (
          <span className="px-1.5 py-0.5 rounded bg-bg-elev text-fg-faint text-[10px] shrink-0">{e.status}</span>
        )}
        <span className="ml-auto shrink-0 text-fg font-semibold text-amber-300 tabular-nums">
          {priceText(e.price)}
          {e.unit && <span className="text-fg-faint font-normal"> /{e.unit}</span>}
        </span>
        <span className="flex items-center gap-0.5 shrink-0">
          {onInsert && (
            <button className="w-6 h-6 inline-flex items-center justify-center rounded text-sky-400 hover:text-sky-300 hover:bg-bg-elev" onClick={() => onInsert(e)} title="插入输入框">
              <Plus size={12} />
            </button>
          )}
          <button className="w-6 h-6 inline-flex items-center justify-center rounded text-fg-faint hover:text-sky-400 hover:bg-bg-elev" onClick={() => onCompare(e)} title="比价">
            <BarChart3 size={12} />
          </button>
          <button className="w-6 h-6 inline-flex items-center justify-center rounded text-fg-faint hover:text-fg hover:bg-bg-elev" onClick={() => onHistory(e.name)} title="价格历史">
            <Clock size={12} />
          </button>
          <button className="w-6 h-6 inline-flex items-center justify-center rounded text-fg-faint hover:text-fg hover:bg-bg-elev" onClick={() => onEdit(e)} title="编辑">
            <Pencil size={12} />
          </button>
          <button className="w-6 h-6 inline-flex items-center justify-center rounded text-fg-faint hover:text-red-400 hover:bg-bg-elev" onClick={() => onDelete(e.name)} title="删除">
            <Trash2 size={12} />
          </button>
        </span>
      </div>
      {(e.source || e.region || e.priceType || e.priceDate || e.validUntil) && (
        <div className="mt-0.5 pl-6 flex items-center gap-1.5 text-fg-faint text-[10.5px] min-w-0">
          {e.source && <span className="truncate">来源：{e.source}</span>}
          {e.region && <span className="shrink-0 truncate max-w-[90px]">· {e.region}</span>}
          {e.priceType && <span className="shrink-0 px-1 py-px rounded bg-bg-elev">{e.priceType}</span>}
          {e.priceDate && <span className="shrink-0 truncate max-w-[110px]">· {e.priceDate}</span>}
          {e.validUntil && <span className="shrink-0">· 至 {e.validUntil}</span>}
        </div>
      )}
      {(e.laborFee > 0 || e.materialFee > 0 || e.machineFee > 0) && (
        <div className="mt-1 pl-6 flex items-center gap-2 text-[10.5px] text-fg-faint min-w-0">
          <span className="shrink-0 font-medium">人材机</span>
          <div className="shrink-0 w-24">
            <MiniCompositionBar labor={e.laborFee ?? 0} material={e.materialFee ?? 0} machine={e.machineFee ?? 0} />
          </div>
          <span className="shrink-0 tabular-nums text-sky-400/90">人工 {priceText(e.laborFee)}</span>
          <span className="shrink-0 tabular-nums text-emerald-400/90">材料 {priceText(e.materialFee)}</span>
          <span className="shrink-0 tabular-nums text-amber-400/90">机械 {priceText(e.machineFee)}</span>
        </div>
      )}
    </div>
  );
});

// ── 人材机组成 mini 条（列表行内：人工/材料/机械 占比）─────────────
function MiniCompositionBar({ labor, material, machine }: { labor: number; material: number; machine: number }) {
  const total = labor + material + machine;
  if (total <= 0) return null;
  const seg = (v: number, cls: string, label: string) =>
    v > 0 ? (
      <div
        className={`h-full ${cls}`}
        style={{ width: `${Math.max(2, (v / total) * 100)}%` }}
        title={`${label} ${v.toFixed(2)}`}
      />
    ) : null;
  return (
    <div
      className="flex h-1 rounded-full overflow-hidden bg-bg-elev"
      role="img"
      aria-label={`人材机：人工 ${labor.toFixed(2)}，材料 ${material.toFixed(2)}，机械 ${machine.toFixed(2)}`}
    >
      {seg(labor, "bg-sky-400/80", "人工")}
      {seg(material, "bg-emerald-400/80", "材料")}
      {seg(machine, "bg-amber-400/80", "机械")}
    </div>
  );
}

// ── 列表视图 ──
export const ListView = memo(function ListView({
  rows,
  selected,
  toggleSelect,
  priceText,
  onEdit,
  onDelete,
  onHistory,
  onCompare,
  onInsert,
  compact,
}: ListViewProps) {
  return (
    <div className={compact ? "p-2 space-y-1" : "px-4 pb-4 space-y-1.5"}>
      {rows.map((e) => (
        <CostRow
          key={e.name}
          row={e}
          selected={selected.has(e.name)}
          compact={compact}
          priceText={priceText}
          onToggleSelect={toggleSelect}
          onEdit={onEdit}
          onDelete={onDelete}
          onHistory={onHistory}
          onCompare={onCompare}
          onInsert={onInsert}
        />
      ))}
    </div>
  );
});

// ── 表格行（React.memo：props 稳定时跳过重渲染）──
export const TableRow = memo(function TableRow({
  row: e,
  selected,
  priceText,
  onToggleSelect,
  onEdit,
  onDelete,
  onHistory,
  onCompare,
  onInsert,
}: TableRowProps) {
  return (
    <tr key={e.name} className={`border-b border-border-soft/50 hover:bg-bg-soft/50 transition-colors ${selected ? "bg-accent/10" : ""}`}>
      <td className="px-3 py-1.5">
        <input type="checkbox" checked={selected} onChange={() => onToggleSelect(e.name)} className="accent-[var(--accent)]" />
      </td>
      <td className="px-3 py-1.5">
        <div className="text-fg font-medium truncate max-w-[220px]">{e.title}</div>
        <div className="text-fg-faint text-[10px] font-mono truncate max-w-[220px]">{e.name}</div>
      </td>
      <td className="px-3 py-1.5 text-fg-dim whitespace-nowrap max-w-[180px] truncate" title={e.categoryPath}>
        {e.categoryPath || e.category || "—"}
      </td>
      <td className="px-3 py-1.5 text-fg-dim">{e.spec || "—"}</td>
      <td className="px-3 py-1.5 text-fg-faint text-[11px] whitespace-nowrap max-w-[150px] truncate" title={[e.region, e.priceDate].filter(Boolean).join(" · ")}>
        {[e.region, e.priceDate].filter(Boolean).join(" · ") || "—"}
      </td>
      <td className="px-3 py-1.5 text-fg-dim">{e.unit || "—"}</td>
      <td className="px-3 py-1.5 text-fg-faint text-[11px] whitespace-nowrap">
        {e.laborFee > 0 || e.materialFee > 0 || e.machineFee > 0 ? (
          <span className="tabular-nums">
            人 {priceText(e.laborFee)} · 材 {priceText(e.materialFee)} · 机 {priceText(e.machineFee)}
          </span>
        ) : (
          "—"
        )}
      </td>
      <td className="px-3 py-1.5 text-fg-faint text-[11px] text-center tabular-nums">
        {(e.componentCount ?? 0) > 0 ? `${e.componentCount} 行` : "—"}
      </td>
      <td className="px-3 py-1.5 text-fg-faint text-[11px] whitespace-nowrap">{e.priceType || "—"}</td>
      <td className="px-3 py-1.5 text-right text-amber-300 font-semibold tabular-nums whitespace-nowrap">
        {priceText(e.price)}
      </td>
      <td className="px-3 py-1.5 text-fg-faint text-[11px] max-w-[140px] truncate" title={e.source}>{e.source || "—"}</td>
      <td className="px-3 py-1.5">
        <span className={`px-1.5 py-0.5 rounded text-[10px] ${e.status === "现行" ? "bg-emerald-500/15 text-emerald-400" : e.status === "草稿" ? "bg-amber-500/15 text-amber-400" : "bg-bg-elev text-fg-faint"}`}>
          {e.status || "现行"}
        </span>
      </td>
      <td className="px-3 py-1.5 text-fg-faint text-[10.5px] tabular-nums whitespace-nowrap">
        {e.updatedAt ? new Date(e.updatedAt).toLocaleDateString("zh-CN") : "—"}
      </td>
      <td className="px-3 py-1.5">
        <div className="flex items-center justify-end gap-0.5">
          {onInsert && (
            <button className="w-6 h-6 inline-flex items-center justify-center rounded text-sky-400 hover:text-sky-300 hover:bg-bg-elev" onClick={() => onInsert(e)} title="插入输入框">
              <Plus size={12} />
            </button>
          )}
          <button className="w-6 h-6 inline-flex items-center justify-center rounded text-fg-faint hover:text-sky-400 hover:bg-bg-elev" onClick={() => onCompare(e)} title="比价">
            <BarChart3 size={12} />
          </button>
          <button className="w-6 h-6 inline-flex items-center justify-center rounded text-fg-faint hover:text-fg hover:bg-bg-elev" onClick={() => onHistory(e.name)} title="价格历史">
            <Clock size={12} />
          </button>
          <button className="w-6 h-6 inline-flex items-center justify-center rounded text-fg-faint hover:text-fg hover:bg-bg-elev" onClick={() => onEdit(e)} title="编辑">
            <Pencil size={12} />
          </button>
          <button className="w-6 h-6 inline-flex items-center justify-center rounded text-fg-faint hover:text-red-400 hover:bg-bg-elev" onClick={() => onDelete(e.name)} title="删除">
            <Trash2 size={12} />
          </button>
        </div>
      </td>
    </tr>
  );
});

// ── 表格视图（记忆中枢完整面板使用）──
const TableView = memo(function TableView({
  rows,
  selected,
  toggleSelect,
  sortKey,
  sortDir,
  toggleSort,
  priceText,
  onSelectAll,
  onEdit,
  onDelete,
  onHistory,
  onCompare,
  onInsert,
}: TableViewProps) {
  const sortArrow = (k: SortKey) => (sortKey === k ? (sortDir === 1 ? " ↑" : " ↓") : "");
  const th = (label: string, k?: SortKey, align: "left" | "right" = "left") => (
    <th
      className={`px-3 py-2 font-medium text-fg-faint text-[11px] whitespace-nowrap select-none ${align === "right" ? "text-right" : "text-left"} ${k ? "cursor-pointer hover:text-fg" : ""}`}
      onClick={k ? () => toggleSort(k) : undefined}
    >
      {label}
      {k ? sortArrow(k) : ""}
    </th>
  );
  return (
    <div className="px-3 pb-4">
      <table className="w-full text-[12px] border-collapse">
        <thead>
          <tr className="border-b border-border-soft/80">
            <th className="px-3 py-2 w-8">
              <input
                type="checkbox"
                checked={rows.length > 0 && selected.size === rows.length}
                onChange={(e) => {
                  if (e.target.checked) setAllSelected();
                  else clearSelection();
                }}
                className="accent-[var(--accent)]"
              />
            </th>
            {th("标题", "title")}
            {th("分类")}
            {th("规格")}
            {th("地区 · 期数")}
            {th("单位")}
            {th("人材机")}
            {th("组成")}
            {th("口径")}
            {th("单价（元）", "price", "right")}
            {th("来源")}
            {th("状态")}
            {th("更新", "updatedAt")}
            <th className="px-3 py-2 text-right font-medium text-fg-faint text-[11px] whitespace-nowrap">操作</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((e) => (
            <TableRow
              key={e.name}
              row={e}
              selected={selected.has(e.name)}
              priceText={priceText}
              onToggleSelect={toggleSelect}
              onEdit={onEdit}
              onDelete={onDelete}
              onHistory={onHistory}
              onCompare={onCompare}
              onInsert={onInsert}
            />
          ))}
        </tbody>
      </table>
    </div>
  );

  function setAllSelected() {
    onSelectAll(new Set(rows.map((e) => e.name)));
  }
  function clearSelection() {
    onSelectAll(new Set());
  }
});
