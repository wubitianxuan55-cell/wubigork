import { useCallback, useEffect, useState, type ReactNode } from "react";
import { Modal } from "antd";
import { BookOpen, Pencil, Plus, RefreshCw, Search, Trash2 } from "../../icons";
import { app } from "../../lib/bridge";
import { useToast } from "../Toast";
import type { CostReviewNote } from "../../lib/types";

const fieldCls =
  "w-full bg-bg border border-border-soft rounded-md text-fg text-[12px] px-2.5 py-1.5 outline-none focus:border-accent transition-colors placeholder:text-fg-faint/50";
const ghostBtn =
  "inline-flex items-center gap-1 px-2.5 h-7 rounded-lg border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors text-[11.5px]";
const solidBtn =
  "inline-flex items-center gap-1 px-2.5 h-7 rounded-lg bg-accent text-white text-[11.5px] hover:opacity-90 transition-opacity disabled:opacity-50";
const iconMini =
  "inline-flex items-center justify-center w-6 h-6 rounded-md text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors";

const CONFIDENCES = ["高", "中", "低"];
const STATUSES = ["草稿", "已确认"];

/**
 * CostNotesView — 复盘笔记（zaojia-database 蒸馏：复盘经验 → 沉淀「判断」）。
 * 结论/适用边界/风险提示/证据来源/可信度/有效期/复核状态 + 引用计数。
 */
export function CostNotesView() {
  const toast = useToast();
  const [notes, setNotes] = useState<CostReviewNote[]>([]);
  const [loading, setLoading] = useState(true);
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState("");
  const [editing, setEditing] = useState<CostReviewNote | null>(null);
  const [formOpen, setFormOpen] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const list = (await app.CostNoteList(query, status)) ?? [];
      setNotes(list);
    } catch {
      setNotes([]);
    } finally {
      setLoading(false);
    }
  }, [query, status]);

  useEffect(() => {
    load();
  }, [load]);

  const openCreate = useCallback(() => {
    setEditing({
      title: "",
      conclusion: "",
      boundary: "",
      risk: "",
      evidence: "",
      confidence: "中",
      status: "草稿",
      category: "",
      projectType: "",
      craft: "",
      validUntil: "",
    });
    setFormOpen(true);
  }, []);

  const openEdit = useCallback((n: CostReviewNote) => {
    setEditing({ ...n });
    setFormOpen(true);
  }, []);

  const save = useCallback(async () => {
    if (!editing) return;
    if (!editing.title.trim()) {
      toast.show("复盘笔记需要标题", "warn");
      return;
    }
    try {
      await app.CostNoteSave(editing);
      toast.show(editing.id ? "笔记已保存" : "笔记已创建", "info");
      setFormOpen(false);
      await load();
    } catch (e) {
      toast.show(String(e), "error");
    }
  }, [editing, load, toast]);

  const remove = useCallback(
    (n: CostReviewNote) => {
      Modal.confirm({
        title: "删除复盘笔记",
        content: `删除「${n.title}」？`,
        okText: "删除",
        okButtonProps: { danger: true },
        cancelText: "取消",
        onOk: async () => {
          if (n.id) {
            await app.CostNoteDelete(n.id).catch(() => {});
          }
          await load();
        },
      });
    },
    [load],
  );

  return (
    <div className="h-full flex flex-col min-h-0 text-[12.5px]">
      <div className="shrink-0 flex items-center gap-3 px-5 h-12 border-b border-border-soft/60">
        <span className="text-fg font-semibold text-[13px] flex items-center gap-1.5">
          <BookOpen size={14} className="text-accent" /> 复盘笔记
        </span>
        <span className="text-[11px] text-fg-faint hidden md:inline">沉淀「判断」而非只堆数据——结论/边界/风险/证据</span>
        <div className="ml-auto flex items-center gap-1.5">
          <div className="flex items-center gap-1 px-2 rounded-lg border border-border bg-bg">
            <Search size={11} className="text-fg-faint" />
            <input
              className="w-36 bg-transparent text-[11.5px] text-fg outline-none placeholder:text-fg-faint/50 py-1"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="搜索标题/结论/边界…"
            />
          </div>
          <select
            className="h-7 px-1.5 rounded-lg border border-border bg-bg text-[11px] text-fg outline-none"
            value={status}
            onChange={(e) => setStatus(e.target.value)}
            title="状态过滤"
          >
            <option value="">全部状态</option>
            {STATUSES.map((s) => (
              <option key={s} value={s}>
                {s}
              </option>
            ))}
          </select>
          <button type="button" className={ghostBtn} onClick={load} title="刷新">
            <RefreshCw size={12} />
          </button>
          <button type="button" className={solidBtn} onClick={openCreate} title="新建复盘笔记">
            <Plus size={12} /> 新建笔记
          </button>
        </div>
      </div>

      <div className="flex-1 min-h-0 overflow-y-auto px-5 py-4">
        {loading && notes.length === 0 ? (
          <div className="space-y-2 animate-pulse">
            <div className="v3-panel rounded-xl h-24" />
            <div className="v3-panel rounded-xl h-24" />
            <div className="v3-panel rounded-xl h-24" />
          </div>
        ) : notes.length === 0 ? (
          <div className="v3-panel rounded-2xl py-16 text-center">
            <BookOpen size={26} className="mx-auto text-fg-faint" />
            <div className="mt-3 text-[13px] text-fg font-medium">
              {query || status ? "没有匹配的复盘笔记" : "还没有复盘笔记"}
            </div>
            <p className="mt-1.5 text-[11.5px] text-fg-faint leading-relaxed max-w-md mx-auto">
              一次测算/报价的结论、适用边界、风险与证据，值得记下来——下次直接复用判断。
            </p>
            {!query && !status && (
              <button type="button" className={`${solidBtn} mt-4`} onClick={openCreate}>
                <Plus size={12} /> 写下第一条
              </button>
            )}
          </div>
        ) : (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">
            {notes.map((n) => (
              <article key={n.id} className="v3-card p-3.5 flex flex-col gap-2">
                <header className="flex items-start gap-2">
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-[13px] font-semibold text-fg">{n.title}</span>
                    <span className="mt-0.5 flex items-center gap-1.5 text-[10px] text-fg-faint">
                      <ConfidenceBadge level={n.confidence} />
                      <StatusBadge status={n.status} />
                      {n.category && <span>{n.category}</span>}
                      {n.projectType && <span>· {n.projectType}</span>}
                      {n.craft && <span>· {n.craft}</span>}
                    </span>
                  </span>
                  <span className="shrink-0 flex items-center gap-1">
                    <button type="button" className={iconMini} title="编辑" onClick={() => openEdit(n)}>
                      <Pencil size={12} />
                    </button>
                    <button type="button" className={`${iconMini} hover:text-err`} title="删除" onClick={() => remove(n)}>
                      <Trash2 size={12} />
                    </button>
                  </span>
                </header>
                {n.conclusion && <p className="text-[12px] text-fg-dim leading-relaxed line-clamp-3">{n.conclusion}</p>}
                <footer className="mt-auto pt-1 flex items-center gap-3 text-[10.5px] text-fg-faint">
                  {n.evidence && <span className="min-w-0 truncate" title={`证据：${n.evidence}`}>证据：{n.evidence}</span>}
                  {n.validUntil && <span>有效期至 {n.validUntil}</span>}
                  <span className="ml-auto shrink-0 tabular-nums">引用 {n.refCount ?? 0} 次 · {fmtDate(n.updatedAt)}</span>
                </footer>
              </article>
            ))}
          </div>
        )}
      </div>

      <Modal
        title={editing?.id ? "编辑复盘笔记" : "新建复盘笔记"}
        open={formOpen}
        onOk={save}
        onCancel={() => setFormOpen(false)}
        okText="保存"
        cancelText="取消"
        width={560}
        destroyOnClose
      >
        {editing && (
          <NoteForm value={editing} onChange={setEditing} />
        )}
      </Modal>
    </div>
  );
}

// ── 笔记表单 ──────────────────────────────────────────────────
function NoteForm({ value, onChange }: { value: CostReviewNote; onChange: (n: CostReviewNote) => void }) {
  const set = (patch: Partial<CostReviewNote>) => onChange({ ...value, ...patch });
  return (
    <div className="space-y-2.5">
      <Field label="标题" required>
        <input className={fieldCls} value={value.title} onChange={(e) => set({ title: e.target.value })} placeholder="如：市政道路土方综合单价复盘" autoFocus />
      </Field>
      <Field label="结论">
        <textarea className={fieldCls} rows={3} value={value.conclusion} onChange={(e) => set({ conclusion: e.target.value })} placeholder="这次测算的核心结论…" />
      </Field>
      <div className="grid grid-cols-2 gap-2.5">
        <Field label="适用边界">
          <input className={fieldCls} value={value.boundary} onChange={(e) => set({ boundary: e.target.value })} placeholder="什么场景下适用" />
        </Field>
        <Field label="风险提示">
          <input className={fieldCls} value={value.risk} onChange={(e) => set({ risk: e.target.value })} placeholder="需要注意的风险" />
        </Field>
      </div>
      <Field label="证据来源">
        <input className={fieldCls} value={value.evidence} onChange={(e) => set({ evidence: e.target.value })} placeholder="项目/版本/文件" />
      </Field>
      <div className="grid grid-cols-2 gap-2.5">
        <Field label="可信度">
          <select className={fieldCls} value={value.confidence} onChange={(e) => set({ confidence: e.target.value })}>
            {CONFIDENCES.map((c) => (
              <option key={c} value={c}>
                {c}
              </option>
            ))}
          </select>
        </Field>
        <Field label="复核状态">
          <select className={fieldCls} value={value.status} onChange={(e) => set({ status: e.target.value })}>
            {STATUSES.map((s) => (
              <option key={s} value={s}>
                {s}
              </option>
            ))}
          </select>
        </Field>
      </div>
      <div className="grid grid-cols-3 gap-2.5">
        <Field label="成本分类">
          <input className={fieldCls} value={value.category ?? ""} onChange={(e) => set({ category: e.target.value })} placeholder="机械/材料…" />
        </Field>
        <Field label="项目类型">
          <input className={fieldCls} value={value.projectType ?? ""} onChange={(e) => set({ projectType: e.target.value })} placeholder="房建/市政…" />
        </Field>
        <Field label="有效期至">
          <input className={fieldCls} value={value.validUntil ?? ""} onChange={(e) => set({ validUntil: e.target.value })} placeholder="YYYY-MM-DD" />
        </Field>
      </div>
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

// ── 徽标 ──────────────────────────────────────────────────────
function ConfidenceBadge({ level }: { level?: string }) {
  const tone =
    level === "高" ? "text-ok bg-ok/10" : level === "低" ? "text-amber-400 bg-amber-400/10" : "text-sky-400 bg-sky-400/10";
  return <span className={`px-1.5 py-px rounded text-[9.5px] ${tone}`}>可信度 {level || "中"}</span>;
}

function StatusBadge({ status }: { status?: string }) {
  const tone = status === "已确认" ? "text-accent bg-accent/10" : "text-warning bg-warning/10";
  return <span className={`px-1.5 py-px rounded text-[9.5px] ${tone}`}>{status || "草稿"}</span>;
}

function fmtDate(iso?: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getMonth() + 1}-${p(d.getDate())}`;
}

export default CostNotesView;
