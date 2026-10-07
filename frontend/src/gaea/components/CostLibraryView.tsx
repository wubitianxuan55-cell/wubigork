import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useDebouncedValue } from "../hooks/useDebouncedValue";
import { Popconfirm, message } from "antd";
import {
  ChevronDown, CloudUpload, Coins, FolderPlus, List,
  Plus, RefreshCw, Table,
} from "../icons";
import { app } from "../lib/bridge";
import { costReadErrorText } from "../lib/bridge/cost";
import type { CostCategory, CostSummary, FilePickResult, PriceHistory, SemanticIndexStatus } from "../lib/types";
import { EmptyState } from "./EmptyState";
import { CostEntryModal } from "./memoryhub/CostEntryModal";
import { CostImportModal } from "./memoryhub/CostImportModal";
import { CostCompareModal } from "./memoryhub/CostCompareModal";
import { CategoryNode, CostRow, ListView, TableView, type SortKey } from "./cost_library_parts";
import { CategoryModal, DeleteCostModal, PriceHistoryModal } from "./cost_library_modals";

export { CostRow, ListView, TableView };

const STATUSES = ["现行", "草稿", "已归档"];



// v4.386 分页：首屏 100 条 + 「加载更多」100 条/批——全表拉取时代的桥载荷
// 与 DOM 行数双降（用户库 1500+ 条时每次搜索全量渲染的根治）；排序在
// 服务端排序后切片（name tie-break 全序保证翻页不漂移），客户端不再重排。
const PAGE_SIZE = 100;

/**
 * CostLibraryView 成本库主视图（v4.459 起为「价格数据 → 资料条目」段——
 * 旧价格手册形态降级归档，检索/导入/AI 解析全保留，仍是资源化与询价的沉淀池）：
 * - 左侧多级分类树（可增删改分类、子树过滤、节点计数）；
 * - 右侧 列表 / 表格 双视图，表格支持排序与多选批量操作（服务端排序）；
 * - 条目按「分类路径」多级保存（一级/二级/…/叶子）。
 * 询价库不在本视图（v4.50 起归「价格数据」模块，与价格源/价格仓库同域）。
 */
export function CostLibraryView() {
  const [entries, setEntries] = useState<CostSummary[]>([]);
  const [total, setTotal] = useState(0);
  // GA6-09：分页检索的读取失败文案（空=无错误）；非空时列表区上方显形提示条。
  const [readError, setReadError] = useState("");
  const [categories, setCategories] = useState<CostCategory[]>([]);
  const [loading, setLoading] = useState(true);
  const [moreBusy, setMoreBusy] = useState(false);
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
  const [statusBusy, setStatusBusy] = useState(false);
  const treeInited = useRef(false);
  // 分页请求序号：新 load 递增后，在途的过期响应（含 loadMore）按序号丢弃。
  const reqSeq = useRef(0);

  // 分类树 → 节点/路径索引（多级路径：一级/二级/…/叶子）。
  const { pathById, allPaths } = useMemo(() => {
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
    const seq = ++reqSeq.current;
    setLoading(true);
    app
      .CostSearchPage(debouncedQuery, selectedPath, status, sortKey ?? "", sortDir, PAGE_SIZE, 0)
      .then((r) => {
        if (seq !== reqSeq.current) return; // 过期响应（筛选已变）丢弃
        const items = r?.items ?? [];
        setEntries(items);
        setTotal(r?.total ?? items.length);
        // GA6-09：Go 侧读取成本库失败时 Items/Total 仍是已读到的部分，
        // 错误在 error 字段——非空即显形提示条，不得静默当「无匹配条目」。
        setReadError(costReadErrorText(r?.error));
        setSelected((prev) => new Set([...prev].filter((n) => items.some((e) => e.name === n))));
      })
      .catch((e: unknown) => {
        if (seq !== reqSeq.current) return;
        setEntries([]);
        setTotal(0);
        // 整调用失败（桥/后端异常）同样不能伪装成空库：复用同一条提示条。
        setReadError(costReadErrorText(e instanceof Error ? e.message : String(e)));
      })
      .finally(() => {
        if (seq === reqSeq.current) setLoading(false);
      });
  }, [debouncedQuery, selectedPath, status, sortKey, sortDir]);

  // 追加下一页：offset=已载条数；跨页去重（翻页期间数据增删的偏移漂移兜底）。
  const loadMore = useCallback(() => {
    if (moreBusy || loading || entries.length >= total) return;
    const seq = reqSeq.current;
    setMoreBusy(true);
    app
      .CostSearchPage(debouncedQuery, selectedPath, status, sortKey ?? "", sortDir, PAGE_SIZE, entries.length)
      .then((r) => {
        if (seq !== reqSeq.current) return;
        const items = r?.items ?? [];
        setEntries((prev) => {
          const seen = new Set(prev.map((e) => e.name));
          return [...prev, ...items.filter((e) => !seen.has(e.name))];
        });
        setTotal(r?.total ?? entries.length);
        // 追加页同样上报读取失败（部分页读坏时不能只因首页成功就吞掉）。
        setReadError(costReadErrorText(r?.error));
      })
      .catch(() => {})
      .finally(() => setMoreBusy(false));
  }, [moreBusy, loading, entries.length, total, debouncedQuery, selectedPath, status, sortKey, sortDir]);

  // v4.196 语义索引显形+显式补齐（成本条目向量覆盖；不做后台守护协程）。
  const [index, setIndex] = useState<SemanticIndexStatus | null>(null);
  const [indexBusy, setIndexBusy] = useState(false);
  const loadIndex = useCallback(() => {
    app.SemanticIndexStatus()
      .then((r) => setIndex(r ?? null))
      .catch(() => setIndex(null));
  }, []);
  useEffect(() => {
    loadIndex();
  }, [loadIndex]);
  const backfillIndex = useCallback(async () => {
    setIndexBusy(true);
    try {
      const r = await app.SemanticIndexBackfill();
      message.success(`语义索引已补齐：新增/更新 ${r.updated} 条，覆盖 ${r.indexed}/${r.total}`);
      loadIndex();
    } catch (e) {
      message.warning(String(e));
    } finally {
      setIndexBusy(false);
    }
  }, [loadIndex]);

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

  const toggleSort = (k: SortKey) => {
    if (sortKey === k) setSortDir((d) => (d === 1 ? -1 : 1));
    else {
      setSortKey(k);
      setSortDir(1);
    }
    // 排序键/方向变化经 load 的依赖自动回到第 1 页重拉（服务端排序）。
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
    // v4.350：失败可见化——此前逐条吞错后无条件提示「已删除 N 条」，后端全挂时
    // 用户看到假成功，实际一条没删。
    const results = await Promise.all(
      [...selected].map((n) => app.CostDelete(n).then(() => true).catch(() => false)),
    );
    const ok = results.filter(Boolean).length;
    const failed = results.length - ok;
    if (failed === 0) message.info(`已删除 ${ok} 条`);
    else message.warning(`已删除 ${ok} 条，${failed} 条失败，请重试`);
    setSelected(new Set());
    load();
  };
  const batchStatus = async (next: string) => {
    if (selected.size === 0 || !next || statusBusy) return;
    setStatusBusy(true);
    try {
      // v4.361：失败可见化——照 batchDelete 计数口径（v4.350），逐条 Get+Save
      // 失败不再吞成假成功「已把 N 条改为…」，后端全挂时用户能看见。
      const results = await Promise.all(
        [...selected].map(async (n) => {
          const e = await app.CostGet(n).catch(() => null);
          if (!e) return false;
          return app.CostSave({ ...e, status: next }).then(() => true).catch(() => false);
        }),
      );
      const ok = results.filter(Boolean).length;
      const failed = results.length - ok;
      if (failed === 0) message.info(`已把 ${ok} 条改为「${next}」`);
      else message.warning(`已把 ${ok} 条改为「${next}」，${failed} 条失败，请重试`);
      setSelected(new Set());
      load();
    } finally {
      setStatusBusy(false);
    }
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
    <div className="flex h-full min-h-0 text-[12.5px]">
      {/* 左：多级分类树 */}
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
              <span className="ml-auto text-[10px] text-fg-faint tabular-nums">{total || ""}</span>
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

      {/* 右：工具条 + 列表/表格 */}
      <div className="flex-1 min-w-0 flex flex-col">
        {/* 工具条 */}
        <div className="shrink-0 flex items-center gap-2 px-4 pt-2.5 pb-1.5">
          <div className="text-fg text-[13px] font-medium">成本库</div>
          <span className="text-fg-faint text-[11px]">综合单价一级 · 人材机二级 · 按专业/分部分类</span>
          {index && (index.modelOk ? (
            <button
              type="button"
              data-testid="semantic-index-chip"
              disabled={indexBusy || index.indexed >= index.total}
              className={`shrink-0 px-1.5 h-[18px] rounded text-[9.5px] font-semibold leading-[18px] tabular-nums transition-opacity ${
                index.indexed >= index.total
                  ? "bg-emerald-500/15 text-emerald-400"
                  : "bg-amber-400/15 text-amber-300 hover:opacity-80"
              } disabled:opacity-70`}
              title={
                index.indexed >= index.total
                  ? "语义索引已全覆盖（正文变更会自动重嵌）"
                  : "部分条目未向量化——点击补齐（首次检索更快、语义召回更全）"
              }
              onClick={() => void backfillIndex()}
            >
              语义索引 {index.indexed}/{index.total}
            </button>
          ) : (
            <span
              className="shrink-0 px-1.5 h-[18px] rounded text-[9.5px] leading-[18px] bg-bg-elev text-fg-faint"
              title={index.modelNote || "本地语义模型未配置——语义召回停用，检索退化为关键词"}
              data-testid="semantic-index-chip"
            >
              语义索引 未启用
            </span>
          ))}
          <div className="ml-auto flex items-center gap-1.5">
            <div className="flex items-center rounded-lg border border-border overflow-hidden">
              <button
                className={`inline-flex items-center gap-1 px-2 h-7 text-[11px] transition-colors ${
                  view === "list" ? "bg-accent text-accent-fg" : "text-fg-faint hover:text-fg"
                }`}
                onClick={() => setView("list")}
                title="列表视图"
              >
                <List size={12} />
              </button>
              <button
                className={`inline-flex items-center gap-1 px-2 h-7 text-[11px] transition-colors ${
                  view === "table" ? "bg-accent text-accent-fg" : "text-fg-faint hover:text-fg"
                }`}
                onClick={() => setView("table")}
                title="表格视图"
              >
                <Table size={12} />
              </button>
            </div>
            <input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="搜索名称/规格/来源…"
              className="px-2.5 h-7 w-44 rounded-lg border border-border bg-bg text-fg text-[12px] placeholder:text-fg-faint outline-none focus:border-accent transition-colors"
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
              className="inline-flex items-center gap-1 px-2.5 h-7 rounded-lg bg-accent text-accent-fg text-[11.5px] hover:opacity-90 transition-opacity"
              onClick={openCreate}
            >
              <Plus size={12} /> 新建
            </button>
          </div>
        </div>

        {/* 筛选行：分类面包屑 + 状态 + 计数 */}
        <div className="shrink-0 flex items-center gap-2 flex-wrap px-4 pb-2">
          <div className="flex items-center gap-1 min-w-0">
            <span
              className={`px-2 h-6 rounded-full text-[11px] transition-colors cursor-pointer ${
                selectedPath === "" ? "bg-accent text-accent-fg" : "bg-bg-elev text-fg-faint hover:text-fg border border-border"
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
                    selectedPath === path ? "bg-accent text-accent-fg" : "bg-bg-elev text-fg-faint hover:text-fg border border-border"
                  }`}
                  onClick={() => setSelectedPath(path)}
                >
                  {seg}
                </span>
              );
            })}
          </div>
          <div className="flex items-center gap-1">
            {STATUSES.map((s) => (
              <button
                key={s}
                onClick={() => setStatus(status === s ? "all" : s)}
                className={`px-2 h-6 rounded-full text-[11px] transition-colors ${
                  status === s ? "bg-accent text-accent-fg" : "bg-bg-elev text-fg-faint hover:text-fg border border-border"
                }`}
              >
                {s}
              </button>
            ))}
          </div>
          <span className="ml-auto text-fg-faint text-[11px] tabular-nums" data-testid="cost-count">
            {entries.length < total ? `已载 ${entries.length} / 共 ${total} 条` : `${total} 条`}
          </span>
          {selected.size > 0 && (
            <span className="flex items-center gap-1.5">
              <span className="text-amber-300 text-[11px]">已选 {selected.size}</span>
              <select
                value=""
                disabled={statusBusy}
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
              <Popconfirm
                title={`删除已选 ${selected.size} 条成本记录？`}
                description="删除后不可恢复"
                okText="删除"
                cancelText="取消"
                okButtonProps={{ danger: true }}
                onConfirm={() => void batchDelete()}
              >
                <button
                  className="px-2 h-6 rounded-md bg-red-500/15 text-red-400 text-[11px] cursor-pointer hover:bg-red-500/25 transition-colors"
                  type="button"
                >
                  批量删除
                </button>
              </Popconfirm>
            </span>
          )}
        </div>

        {/* GA6-09 读取失败提示条：error 非空时显形（含可重试路径），不静默当空库 */}
        {readError && (
          <div
            role="alert"
            data-testid="cost-read-error"
            className="flex items-center gap-2 px-3 py-2 border-b border-amber-400/30 bg-amber-400/10 text-amber-200 text-[11.5px]"
          >
            <span className="flex-1 min-w-0">{readError}——下方为已读到的部分，结果可能不完整。</span>
            <button
              type="button"
              data-testid="cost-read-error-retry"
              onClick={load}
              className="shrink-0 px-2 h-6 rounded-md border border-amber-400/40 hover:bg-amber-400/20 transition-colors"
            >
              重试
            </button>
          </div>
        )}

        {/* 内容区 */}
        <div className="flex-1 min-h-0 overflow-y-auto">
          {loading ? (
            <div className="p-4 space-y-2 animate-pulse">
              {Array.from({ length: 6 }).map((_, i) => (
                <div key={i} className="h-11 rounded-lg bg-bg-elev/60" />
              ))}
            </div>
          ) : entries.length === 0 ? (
            <div className="h-full flex items-center justify-center">
              <EmptyState
                message={
                  readError
                    ? "成本库读取失败，暂无可显示条目——点上方提示条「重试」重新读取"
                    : "暂无成本条目 — 新建、导入报价单，或测算完成后沉淀到成本库"
                }
              />
            </div>
          ) : view === "table" ? (
            <TableView
              rows={entries}
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
            />
          ) : (
            <ListView
              rows={entries}
              selected={selected}
              toggleSelect={toggleSelect}
              priceText={priceText}
              onEdit={openEdit}
              onDelete={setDeleteName}
              onHistory={openHistory}
              onCompare={openCompare}
            />
          )}
          {/* v4.386 分页尾：未载完时「加载更多」 */}
          {!loading && entries.length < total && (
            <div className="px-4 pb-4 pt-1 flex justify-center">
              <button
                type="button"
                data-testid="cost-load-more"
                disabled={moreBusy}
                onClick={loadMore}
                className="px-3 h-7 rounded-lg border border-border text-fg-dim text-[11.5px] hover:bg-bg-soft disabled:opacity-60 transition-colors"
              >
                {moreBusy ? "加载中…" : `加载更多（余 ${total - entries.length} 条）`}
              </button>
            </div>
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
      <DeleteCostModal name={deleteName} onCancel={() => setDeleteName(null)} onOk={handleDelete} />

      {/* 分类 新建/重命名 */}
      <CategoryModal
        catModal={catModal}
        name={catName}
        onNameChange={setCatName}
        onSave={saveCategory}
        onCancel={() => setCatModal(null)}
        parentPath={catModal && catModal.mode === "create" && catModal.parentId > 0 ? (pathById.get(catModal.parentId) ?? "—") : ""}
      />

      {/* 价格历史 */}
      <PriceHistoryModal open={historyOpen} name={historyName} rows={historyRows} onClose={() => setHistoryOpen(false)} priceText={priceText} />
    </div>
  );
}

