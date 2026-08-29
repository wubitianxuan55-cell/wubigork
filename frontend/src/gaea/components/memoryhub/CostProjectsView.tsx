import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { Modal } from "antd";
import {
  Calculator, CheckCircle, ChevronRight, Clock, Coins, Layers,
  Plus, RefreshCw, Rollback, Save, Search, Sparkles, Trash2, X,
} from "../../icons";
import { app } from "../../lib/bridge";
import { useToast } from "../Toast";
import { useDebouncedValue } from "../../hooks/useDebouncedValue";
import type { CostComponent, CostComposeEvidence, CostEstimateItem, CostEstimateVersion, CostProject, CostProjectSummary, CostSummary } from "../../lib/types";
import { ComposeModal } from "./ComposeModal";
import { FiveCalcPanel } from "./FiveCalcPanel";

const fmtPrice = (p: number) => "¥" + new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 2 }).format(p);

// 与既有组件一致的内联样式（项目无自定义 .field/.btn-* 类，全部 Tailwind）
const fieldCls =
  "w-full bg-bg border border-border-soft rounded-md text-fg text-[12px] px-2.5 py-1.5 outline-none focus:border-accent transition-colors placeholder:text-fg-faint/50";
const ghostBtn =
  "inline-flex items-center gap-1 px-2.5 h-7 rounded-lg border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors text-[11.5px]";
const solidBtn =
  "inline-flex items-center gap-1 px-2.5 h-7 rounded-lg bg-accent text-white text-[11.5px] hover:opacity-90 transition-opacity disabled:opacity-50";
const iconBtn =
  "inline-flex items-center justify-center w-6 h-6 rounded-md border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors";

/**
 * CostProjectsView — 测算项目与沉淀闭环（zaojia-database 蒸馏：我的项目/
 * 工程量清单/版本留痕 → 沉淀回成本库）。v3.1.1 补前端 UI（后端 costproject
 * 包 + CostB 绑定 v3.1.0 已就绪）。
 *
 * 交互：项目列表（左）→ 详情（右）：明细行可编辑（引用成本库单价或手动估价，
 * 金额=数量×单价自动算）、保存版本（不可变快照）、沉淀选中行回成本库。
 */
export function CostProjectsView({ onChanged }: { onChanged?: () => void }) {
  const toast = useToast();
  const [projects, setProjects] = useState<CostProjectSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const selectedRef = useRef<string | null>(null);
  const [project, setProject] = useState<CostProject | null>(null);
  const [items, setItems] = useState<CostEstimateItem[]>([]);
  const [versions, setVersions] = useState<CostEstimateVersion[]>([]);
  const [formOpen, setFormOpen] = useState(false);
  const [formProject, setFormProject] = useState<CostProject>(emptyProject());
  const [versionOpen, setVersionOpen] = useState(false);
  const [versionNote, setVersionNote] = useState("");
  const [savingVersion, setSavingVersion] = useState(false);
  const [snapshot, setSnapshot] = useState<CostEstimateVersion | null>(null);
  const [selectedRows, setSelectedRows] = useState<Set<number>>(new Set());
  const [sedimenting, setSedimenting] = useState(false);
  // ── v4.2 AI 组价（ComposeModal）──
  const [composeOpen, setComposeOpen] = useState(false);
  const [composeDesc, setComposeDesc] = useState("");
  const [composeUnit, setComposeUnit] = useState("");

  const setSel = useCallback((id: string | null) => {
    selectedRef.current = id;
    setSelectedId(id);
  }, []);

  const loadDetail = useCallback(async (id: string) => {
    const [p, its, vers] = await Promise.all([
      app.CostProjectGet(id).catch(() => null),
      app.CostEstimateItems(id).catch(() => [] as CostEstimateItem[]),
      app.CostEstimateVersions(id).catch(() => [] as CostEstimateVersion[]),
    ]);
    setProject(p);
    setItems(its ?? []);
    setVersions(vers ?? []);
    setSelectedRows(new Set());
  }, []);

  const refreshList = useCallback(
    async (preferId?: string | null) => {
      setLoading(true);
      try {
        const list = (await app.CostProjectList()) ?? [];
        setProjects(list);
        const keep = preferId ?? selectedRef.current;
        const next = keep && list.some((p) => p.id === keep) ? keep : (list[0]?.id ?? null);
        setSel(next);
        if (next) {
          await loadDetail(next);
        } else {
          setProject(null);
          setItems([]);
          setVersions([]);
        }
      } catch {
        setProjects([]);
      } finally {
        setLoading(false);
      }
    },
    [loadDetail, setSel],
  );

  useEffect(() => {
    refreshList();
  }, [refreshList]);

  // ── 项目 CRUD ──────────────────────────────────────────────
  const openCreate = useCallback(() => {
    setFormProject(emptyProject());
    setFormOpen(true);
  }, []);

  const openEdit = useCallback(() => {
    if (project) {
      setFormProject({ ...project });
      setFormOpen(true);
    }
  }, [project]);

  const saveProject = useCallback(async () => {
    if (!formProject.name.trim()) {
      toast.show("测算项目需要名称", "warn");
      return;
    }
    try {
      const id = await app.CostProjectSave(formProject);
      toast.show(formProject.id ? "项目已保存" : "项目已创建", "info");
      setFormOpen(false);
      await refreshList(id);
      onChanged?.();
    } catch (e) {
      toast.show(String(e), "error");
    }
  }, [formProject, refreshList, toast, onChanged]);

  const deleteProject = useCallback(() => {
    if (!project) return;
    Modal.confirm({
      title: "删除测算项目",
      content: `将删除「${project.name}」及其全部明细行与版本快照（级联），不可恢复。`,
      okText: "删除",
      okButtonProps: { danger: true },
      cancelText: "取消",
      onOk: async () => {
        try {
          await app.CostProjectDelete(project.id);
          toast.show("项目已删除", "info");
          await refreshList();
          onChanged?.();
        } catch (e) {
          toast.show(String(e), "error");
        }
      },
    });
  }, [project, refreshList, toast, onChanged]);

  // ── 明细行 ─────────────────────────────────────────────────
  const patchItem = useCallback((id: number | undefined, patch: Partial<CostEstimateItem>) => {
    setItems((prev) => prev.map((it) => (it.id === id ? { ...it, ...patch } : it)));
  }, []);

  const addRow = useCallback(() => {
    if (!project) return;
    setItems((prev) => [
      ...prev,
      {
        projectId: project.id,
        name: "",
        title: "",
        categoryPath: "",
        unit: "",
        quantity: 1,
        price: 0,
      },
    ]);
  }, [project]);

  const saveRow = useCallback(
    async (it: CostEstimateItem) => {
      if (!it.title.trim()) return;
      try {
        const id = await app.CostEstimateItemSave({ ...it, quantity: it.quantity || 0, price: it.price || 0 });
        if (!it.id && id) patchItem(undefined, { id });
      } catch (e) {
        toast.show(String(e), "error");
      }
    },
    [patchItem, toast],
  );

  const deleteRow = useCallback((it: CostEstimateItem) => {
    Modal.confirm({
      title: "删除明细行",
      content: `删除「${it.title || "未命名行"}」？`,
      okText: "删除",
      okButtonProps: { danger: true },
      cancelText: "取消",
      onOk: async () => {
        if (it.id) {
          await app.CostEstimateItemDelete(it.id).catch(() => {});
        }
        setItems((prev) => prev.filter((x) => x.id !== it.id));
      },
    });
  }, []);

  // ── v4.2 AI 组价 ─────────────────────────────────────────────
  // openCompose 预填：选中行优先，否则第一行（描述/单位带入弹窗）。
  const openCompose = useCallback(() => {
    const pick = items.find((it) => it.id !== undefined && selectedRows.has(it.id)) ?? items[0];
    setComposeDesc(pick?.title ?? "");
    setComposeUnit(pick?.unit ?? "");
    setComposeOpen(true);
  }, [items, selectedRows]);

  // applyCompose 组价结果作为新明细行应用（描述/单位/推荐价），自动保存。
  const applyCompose = useCallback(
    async (r: { desc: string; unit: string; price: number; components: CostComponent[]; evidence: CostComposeEvidence[] }) => {
      if (!project) return;
      if (!r.desc.trim() || r.price <= 0) {
        toast.show("组价结果无效，请先开始组价", "warn");
        return;
      }
      try {
        const row: CostEstimateItem = {
          projectId: project.id,
          name: "",
          title: r.desc,
          categoryPath: "",
          unit: r.unit,
          quantity: 1,
          price: r.price,
        };
        const id = await app.CostEstimateItemSave(row);
        setItems((prev) => [...prev, { ...row, id: id || undefined, amount: r.price }]);
        toast.show(`已应用组价「${r.desc}」`, "info");
        setComposeOpen(false);
        onChanged?.();
      } catch (e) {
        toast.show(String(e), "error");
      }
    },
    [project, toast, onChanged],
  );

  // ── 版本 ────────────────────────────────────────────────────
  const saveVersion = useCallback(async () => {
    if (!project) return;
    if (items.length === 0) {
      toast.show("项目没有明细行，无法保存版本", "warn");
      return;
    }
    setSavingVersion(true);
    try {
      await app.CostEstimateVersionSave(project.id, versionNote);
      toast.show("版本已保存（不可变快照）", "info");
      setVersionOpen(false);
      setVersionNote("");
      await refreshList(project.id);
    } catch (e) {
      toast.show(String(e), "error");
    } finally {
      setSavingVersion(false);
    }
  }, [project, items.length, versionNote, refreshList, toast]);

  const restoreVersion = useCallback(
    (v: CostEstimateVersion) => {
      if (!project) return;
      Modal.confirm({
        title: `恢复版本 v${v.version}`,
        content: `将用 v${v.version} 的快照（合计 ${fmtPrice(v.total)}）重建当前明细行；现有 ${items.length} 行会先删除。`,
        okText: "恢复",
        cancelText: "取消",
        onOk: async () => {
          try {
            const rows: CostEstimateItem[] = JSON.parse(v.snapshot);
            for (const it of items) {
              if (it.id) await app.CostEstimateItemDelete(it.id).catch(() => {});
            }
            for (const r of rows) {
              const { id: _old, ...rest } = r;
              void _old;
              await app.CostEstimateItemSave({ ...rest, projectId: project.id });
            }
            toast.show(`已恢复版本 v${v.version}`, "info");
            await refreshList(project.id);
          } catch (e) {
            toast.show(String(e), "error");
          }
        },
      });
    },
    [project, items, refreshList, toast],
  );

  // ── 沉淀 ────────────────────────────────────────────────────
  const sediment = useCallback(async () => {
    if (!project || selectedRows.size === 0) return;
    setSedimenting(true);
    try {
      const n = await app.CostEstimateSediment(project.id, [...selectedRows]);
      toast.show(`已沉淀 ${n} 条明细到成本库`, "info");
      await refreshList(project.id);
      onChanged?.();
    } catch (e) {
      toast.show(String(e), "error");
    } finally {
      setSedimenting(false);
    }
  }, [project, selectedRows, refreshList, toast, onChanged]);

  const toggleRow = useCallback((id: number | undefined, checked: boolean) => {
    if (id === undefined) return;
    setSelectedRows((prev) => {
      const next = new Set(prev);
      if (checked) next.add(id);
      else next.delete(id);
      return next;
    });
  }, []);

  const total = items.reduce((s, it) => s + (it.quantity || 0) * (it.price || 0), 0);
  const rowCount = items.length;

  return (
    <div className="h-full flex min-h-0 text-[12.5px]">
      {/* 左列：项目列表 */}
      <aside className="w-60 shrink-0 border-r border-border-soft/60 flex flex-col min-h-0 bg-bg/40">
        <div className="flex items-center justify-between px-3 h-10 border-b border-border-soft/50">
          <span className="text-fg font-semibold text-[12px] flex items-center gap-1.5">
            <Calculator size={13} className="text-accent" /> 测算项目
          </span>
          <div className="flex items-center gap-1">
            <button type="button" className={iconBtn} title="刷新" onClick={() => refreshList()}>
              <RefreshCw size={12} />
            </button>
            <button type="button" className={iconBtn} title="新建项目" onClick={openCreate}>
              <Plus size={13} />
            </button>
          </div>
        </div>
        <div className="flex-1 min-h-0 overflow-y-auto py-1.5 px-1.5 space-y-1">
          {loading && projects.length === 0 ? (
            <div className="text-[11px] text-fg-faint text-center py-6">加载中…</div>
          ) : projects.length === 0 ? (
            <div className="text-center py-8 px-3">
              <Coins size={20} className="mx-auto text-fg-faint" />
              <div className="mt-2 text-[11.5px] text-fg-faint leading-relaxed">
                还没有测算项目。
                <br />
                一次报价/测算工作 = 一个项目。
              </div>
              <button type="button" className={`${solidBtn} mt-3`} onClick={openCreate}>
                <Plus size={12} /> 新建项目
              </button>
            </div>
          ) : (
            projects.map((p) => (
              <button
                key={p.id}
                type="button"
                onClick={() => {
                  setSel(p.id);
                  loadDetail(p.id);
                }}
                className={`w-full text-left rounded-xl px-2.5 py-2 transition-colors ${
                  selectedId === p.id ? "bg-accent/12 ring-1 ring-accent/25" : "hover:bg-bg-elev/60"
                }`}
              >
                <span className="flex items-center justify-between gap-1">
                  <span className="min-w-0 truncate text-[12px] font-medium text-fg">{p.name}</span>
                  <StatusBadge status={p.status} />
                </span>
                <span className="mt-1 flex items-center gap-1.5 text-[10.5px] text-fg-faint">
                  <span>{p.projectType || "未分类"}</span>
                  <span>·</span>
                  <span>{p.itemCount} 行</span>
                  <span>·</span>
                  <span>v{p.versionCount}</span>
                  <span className="ml-auto tabular-nums text-fg-dim">{fmtPrice(p.total)}</span>
                </span>
              </button>
            ))
          )}
        </div>
      </aside>

      {/* 右区：项目详情 */}
      <section className="flex-1 min-w-0 flex flex-col min-h-0">
        {!project ? (
          <div className="flex-1 flex items-center justify-center text-[12px] text-fg-faint">
            选择或新建一个测算项目开始
          </div>
        ) : (
          <>
            <div className="shrink-0 flex items-center gap-2 px-4 h-12 border-b border-border-soft/50">
              <span className="min-w-0 truncate text-fg font-semibold text-[13px]">{project.name}</span>
              <StatusBadge status={project.status} />
              {project.projectType && (
                <span className="px-1.5 py-px rounded bg-bg-elev text-[10px] text-fg-faint">{project.projectType}</span>
              )}
              <span className="ml-auto flex items-center gap-1.5">
                <button type="button" className={ghostBtn} onClick={openEdit} title="编辑项目信息">
                  <Save size={12} /> 编辑信息
                </button>
                <button type="button" className={`${ghostBtn} text-err`} onClick={deleteProject} title="删除项目">
                  <Trash2 size={12} /> 删除
                </button>
              </span>
            </div>

            <div className="flex-1 min-h-0 overflow-y-auto px-4 py-3 space-y-3">
              <div className="flex items-center justify-between">
                <span className="text-[12px] font-semibold text-fg flex items-center gap-1.5">
                  <Layers size={12} className="text-sky-400" /> 工程量清单（{rowCount} 行 · 合计 {fmtPrice(total)}）
                </span>
                <div className="flex items-center gap-1.5">
                  <button type="button" className={ghostBtn} onClick={openCompose} title="AI 组价：清单描述 → 相似清单价格带 + 证据链 + 人材机拆解">
                    <Sparkles size={12} className="text-amber-400" /> AI 组价
                  </button>
                  <button type="button" className={ghostBtn} onClick={addRow} title="添加明细行">
                    <Plus size={12} /> 加行
                  </button>
                  <button
                    type="button"
                    className={`${solidBtn} ${selectedRows.size === 0 ? "opacity-40 pointer-events-none" : ""}`}
                    onClick={sediment}
                    disabled={sedimenting || selectedRows.size === 0}
                    title="把选中的行 UPSERT 回成本库（沉淀即调用）"
                  >
                    <Coins size={12} /> 沉淀选中{selectedRows.size > 0 ? `(${selectedRows.size})` : ""}
                  </button>
                </div>
              </div>

              {items.length === 0 ? (
                <div className="v3-panel rounded-xl py-10 text-center text-[11.5px] text-fg-faint">
                  还没有明细行——「加行」手动录入，或从成本库引用单价
                </div>
              ) : (
                <div className="v3-panel rounded-xl overflow-x-auto">
                  <table className="w-full text-[11.5px]">
                    <thead>
                      <tr className="text-left text-[10px] text-fg-faint border-b border-border-soft/50">
                        <th className="py-1.5 px-2 w-7">
                          <span title="沉淀选择">✓</span>
                        </th>
                        <th className="py-1.5 px-2 min-w-52">名称 / 引用单价</th>
                        <th className="py-1.5 px-2 w-16">单位</th>
                        <th className="py-1.5 px-2 w-20">数量</th>
                        <th className="py-1.5 px-2 w-24">单价</th>
                        <th className="py-1.5 px-2 w-24 text-right">金额</th>
                        <th className="py-1.5 px-2 w-8" />
                      </tr>
                    </thead>
                    <tbody>
                      {items.map((it, idx) => (
                        <ItemRow
                          key={it.id ?? `new-${idx}`}
                          item={it}
                          selected={it.id !== undefined && selectedRows.has(it.id)}
                          onToggle={(c) => toggleRow(it.id, c)}
                          onPatch={(patch) => patchItem(it.id, patch)}
                          onSave={() => saveRow(it)}
                          onDelete={() => deleteRow(it)}
                        />
                      ))}
                    </tbody>
                  </table>
                </div>
              )}

              <div className="flex items-center justify-between">
                <span className="text-[12px] font-semibold text-fg flex items-center gap-1.5">
                  <Clock size={12} className="text-violet-400" /> 版本快照（{versions.length}）
                </span>
                <button type="button" className={ghostBtn} onClick={() => setVersionOpen(true)} title="保存不可变版本快照">
                  <CheckCircle size={12} /> 保存版本
                </button>
              </div>
              {versions.length === 0 ? (
                <div className="text-[11px] text-fg-faint">暂无版本——保存后不可变留痕，可回看/对比/恢复思路。</div>
              ) : (
                <div className="v3-panel rounded-xl divide-y divide-border-soft/40">
                  {versions.map((v) => (
                    <button
                      key={v.id}
                      type="button"
                      className="w-full flex items-center gap-2 px-3 py-2 text-left hover:bg-bg-elev/50 transition-colors"
                      onClick={() => setSnapshot(v)}
                      title="查看快照明细"
                    >
                      <span className="px-1.5 h-[18px] rounded bg-accent/12 text-accent text-[10px] font-semibold leading-[18px]">
                        v{v.version}
                      </span>
                      <span className="min-w-0 flex-1 truncate text-fg-dim text-[11.5px]">{v.note || "（无备注）"}</span>
                      <span className="tabular-nums text-fg font-medium">{fmtPrice(v.total)}</span>
                      <span className="text-fg-faint text-[10.5px]">{fmtTime(v.createdAt)}</span>
                      <ChevronRight size={11} className="text-fg-faint" />
                    </button>
                  ))}
                </div>
              )}
            </div>

            {/* v4.2 五算对比（估/概/预/结/决） */}
            <div className="space-y-2">
              <div className="flex items-center gap-1.5">
                <span className="text-[12px] font-semibold text-fg flex items-center gap-1.5">
                  <Calculator size={12} className="text-sky-400" /> 五算对比
                </span>
              </div>
              <FiveCalcPanel projectId={project.id} onChanged={() => refreshList(project.id)} />
            </div>
          </>
        )}
      </section>

      <Modal
        title={formProject.id ? "编辑测算项目" : "新建测算项目"}
        open={formOpen}
        onOk={saveProject}
        onCancel={() => setFormOpen(false)}
        okText="保存"
        cancelText="取消"
        width={420}
        destroyOnClose
      >
        <ProjectForm value={formProject} onChange={setFormProject} />
      </Modal>

      <Modal
        title="保存版本快照"
        open={versionOpen}
        onOk={saveVersion}
        onCancel={() => setVersionOpen(false)}
        okText="保存"
        cancelText="取消"
        okButtonProps={{ loading: savingVersion }}
        width={420}
        destroyOnClose
      >
        <p className="text-[11.5px] text-fg-faint mb-2">
          对当前 {rowCount} 行明细做 JSON 快照（合计 {fmtPrice(total)}），版本不可变，可回看/对比/恢复思路。
        </p>
        <input className={fieldCls} value={versionNote} onChange={(e) => setVersionNote(e.target.value)} placeholder="版本备注（可选），如：土方工程 V1 初稿" />
      </Modal>

      <Modal
        title={snapshot ? `版本 v${snapshot.version} 快照 · ${fmtPrice(snapshot.total)}` : ""}
        open={snapshot !== null}
        onCancel={() => setSnapshot(null)}
        footer={
          snapshot
            ? [
                <button key="restore" type="button" className={ghostBtn} onClick={() => { restoreVersion(snapshot); setSnapshot(null); }}>
                  <Rollback size={12} /> 恢复此版本
                </button>,
                <button key="close" type="button" className={solidBtn} onClick={() => setSnapshot(null)}>
                  关闭
                </button>,
              ]
            : null
        }
        width={640}
      >
        {snapshot && <SnapshotTable v={snapshot} />}
      </Modal>

      {/* v4.2 AI 组价弹窗：描述 → 价格带/证据链/人材机拆解 → 应用为明细行 */}
      <ComposeModal
        open={composeOpen}
        initialDesc={composeDesc}
        initialUnit={composeUnit}
        onClose={() => setComposeOpen(false)}
        onApply={applyCompose}
      />
    </div>
  );
}

// ── 明细行（可编辑）────────────────────────────────────────────
function ItemRow({
  item,
  selected,
  onToggle,
  onPatch,
  onSave,
  onDelete,
}: {
  item: CostEstimateItem;
  selected: boolean;
  onToggle: (c: boolean) => void;
  onPatch: (p: Partial<CostEstimateItem>) => void;
  onSave: () => void;
  onDelete: () => void;
}) {
  const [pickOpen, setPickOpen] = useState(false);
  const amount = (item.quantity || 0) * (item.price || 0);
  return (
    <tr className="border-b border-border-soft/30 last:border-0 hover:bg-bg-elev/30 align-top">
      <td className="py-1 px-2">
        <input
          type="checkbox"
          className="accent-accent"
          checked={selected}
          disabled={item.id === undefined}
          title={item.id === undefined ? "先保存该行再沉淀" : "选择以沉淀回成本库"}
          onChange={(e) => onToggle(e.target.checked)}
        />
      </td>
      <td className="py-1 px-2">
        <input
          className={fieldCls}
          value={item.title}
          placeholder="名称（必填）"
          onChange={(e) => onPatch({ title: e.target.value, name: slug(e.target.value) })}
          onBlur={onSave}
        />
        <div className="relative mt-1">
          <div className="flex items-center gap-1">
            <Search size={10} className="text-fg-faint shrink-0" />
            <input
              className={`${fieldCls} !text-[10.5px]`}
              value={item.entryName || ""}
              placeholder="引用成本库单价（搜索）或留空手动估价"
              onChange={(e) => onPatch({ entryName: e.target.value })}
              onFocus={() => setPickOpen(true)}
              onBlur={() => setTimeout(() => setPickOpen(false), 200)}
            />
          </div>
          {pickOpen && (
            <EntryPicker
              query={item.entryName || ""}
              onPick={(e) => {
                onPatch({
                  title: e.title,
                  name: e.name,
                  unit: e.unit,
                  price: e.price,
                  categoryPath: e.categoryPath || e.category || "",
                  entryName: e.name,
                  source: e.source || "成本库引用",
                });
                setPickOpen(false);
                onSave();
              }}
            />
          )}
        </div>
      </td>
      <td className="py-1 px-2">
        <input className={fieldCls} value={item.unit} placeholder="单位" onChange={(e) => onPatch({ unit: e.target.value })} onBlur={onSave} />
      </td>
      <td className="py-1 px-2">
        <input
          className={`${fieldCls} text-right tabular-nums`}
          type="number"
          min={0}
          value={item.quantity || ""}
          placeholder="数量"
          onChange={(e) => onPatch({ quantity: Number(e.target.value) })}
          onBlur={onSave}
        />
      </td>
      <td className="py-1 px-2">
        <input
          className={`${fieldCls} text-right tabular-nums`}
          type="number"
          min={0}
          value={item.price || ""}
          placeholder="单价"
          onChange={(e) => onPatch({ price: Number(e.target.value) })}
          onBlur={onSave}
        />
      </td>
      <td className="py-1 px-2 text-right tabular-nums text-fg font-medium">{fmtPrice(amount)}</td>
      <td className="py-1 px-2">
        <button type="button" className={`${iconBtn} text-fg-faint hover:text-err`} title="删除行" onClick={onDelete}>
          <X size={12} />
        </button>
      </td>
    </tr>
  );
}

// ── 成本库单价搜索下拉 ────────────────────────────────────────
function EntryPicker({ query, onPick }: { query: string; onPick: (e: CostSummary) => void }) {
  const [q] = useState(query);
  const [results, setResults] = useState<CostSummary[]>([]);
  const debounced = useDebouncedValue(q, 250);
  useEffect(() => {
    if (!debounced.trim()) {
      setResults([]);
      return;
    }
    let alive = true;
    app
      .CostSearch(debounced, "", "")
      .then((r) => {
        if (alive) setResults((r ?? []).slice(0, 8));
      })
      .catch(() => {});
    return () => {
      alive = false;
    };
  }, [debounced]);
  return (
    <div className="absolute z-20 left-0 right-0 top-full mt-0.5 rounded-lg border border-border bg-bg-elev shadow-xl overflow-hidden">
      {results.length === 0 ? (
        <div className="px-2.5 py-2 text-[10.5px] text-fg-faint">输入关键词搜索成本库单价…</div>
      ) : (
        results.map((e) => (
          <button
            key={e.name}
            type="button"
            className="w-full text-left px-2.5 py-1.5 hover:bg-accent/10 transition-colors"
            onMouseDown={(ev) => ev.preventDefault()}
            onClick={() => onPick(e)}
          >
            <span className="block truncate text-[11px] text-fg">{e.title}</span>
            <span className="block truncate text-[9.5px] text-fg-faint">
              {e.categoryPath || e.category || "未分类"} · {e.unit || "-"} · <span className="tabular-nums">{fmtPrice(e.price)}</span>
            </span>
          </button>
        ))
      )}
    </div>
  );
}

// ── 项目表单 ──────────────────────────────────────────────────
function ProjectForm({ value, onChange }: { value: CostProject; onChange: (p: CostProject) => void }) {
  const set = (patch: Partial<CostProject>) => onChange({ ...value, ...patch });
  return (
    <div className="space-y-2.5">
      <Field label="项目名称" required>
        <input className={fieldCls} value={value.name} onChange={(e) => set({ name: e.target.value })} placeholder="如：XX 市政道路土方测算" autoFocus />
      </Field>
      <div className="grid grid-cols-2 gap-2.5">
        <Field label="项目类型">
          <input className={fieldCls} value={value.projectType} onChange={(e) => set({ projectType: e.target.value })} placeholder="房建 / 市政 / 安装…" />
        </Field>
        <Field label="规模">
          <input className={fieldCls} value={value.scale} onChange={(e) => set({ scale: e.target.value })} placeholder="如：5 万 m²" />
        </Field>
      </div>
      <Field label="工艺 / 说明">
        <input className={fieldCls} value={value.craft} onChange={(e) => set({ craft: e.target.value })} placeholder="施工工艺或备注" />
      </Field>
      <Field label="备注">
        <textarea className={fieldCls} rows={2} value={value.note} onChange={(e) => set({ note: e.target.value })} placeholder="可选" />
      </Field>
    </div>
  );
}

function Field({ label, required, children }: { label: string; required?: boolean; children: ReactNode }) {
  return (
    <label className="block">
      <span className="block text-[10.5px] text-fg-faint mb-1">
        {label}
        {required && <span className="text-err"> *</span>}
      </span>
      {children}
    </label>
  );
}

// ── 快照表格 ──────────────────────────────────────────────────
function SnapshotTable({ v }: { v: CostEstimateVersion }) {
  let rows: CostEstimateItem[] = [];
  try {
    rows = JSON.parse(v.snapshot);
  } catch {
    rows = [];
  }
  return (
    <div>
      <p className="text-[11px] text-fg-faint mb-2">
        {v.note || "（无备注）"} · 保存于 {fmtTime(v.createdAt)} · 合计 {fmtPrice(v.total)}
      </p>
      {rows.length === 0 ? (
        <div className="text-[11.5px] text-fg-faint py-6 text-center">快照为空或已损坏</div>
      ) : (
        <table className="w-full text-[11.5px]">
          <thead>
            <tr className="text-left text-[10px] text-fg-faint border-b border-border-soft/50">
              <th className="py-1.5 px-2">名称</th>
              <th className="py-1.5 px-2 w-14">单位</th>
              <th className="py-1.5 px-2 w-16 text-right">数量</th>
              <th className="py-1.5 px-2 w-24 text-right">单价</th>
              <th className="py-1.5 px-2 w-24 text-right">金额</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((r, i) => (
              <tr key={i} className="border-b border-border-soft/20 last:border-0">
                <td className="py-1 px-2 text-fg-dim">{r.title}</td>
                <td className="py-1 px-2 text-fg-faint">{r.unit}</td>
                <td className="py-1 px-2 text-right tabular-nums">{r.quantity}</td>
                <td className="py-1 px-2 text-right tabular-nums">{fmtPrice(r.price)}</td>
                <td className="py-1 px-2 text-right tabular-nums">{fmtPrice((r.quantity || 0) * (r.price || 0))}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}

// ── 小工具 ────────────────────────────────────────────────────
function StatusBadge({ status }: { status: string }) {
  const tone =
    status === "已沉淀"
      ? "text-ok bg-ok/10"
      : status === "已保存版本"
        ? "text-accent bg-accent/10"
        : "text-amber-400 bg-amber-400/10";
  return <span className={`shrink-0 px-1.5 py-px rounded text-[9.5px] ${tone}`}>{status || "编制中"}</span>;
}

function fmtTime(iso?: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getMonth() + 1}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`;
}

function slug(s: string): string {
  return s
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9\u4e00-\u9fa5]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 60);
}

function emptyProject(): CostProject {
  return { id: "", name: "", projectType: "", scale: "", craft: "", status: "编制中", note: "" };
}
