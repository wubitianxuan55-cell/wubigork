import { useCallback, useEffect, useState } from "react";
import { Modal } from "antd";
import { BookOpen, CloudUpload, Loader, Sparkles } from "../../icons";
import { app } from "../../lib/bridge";
import type { KnowledgeEntry, KnowledgeImportPreview, KnowledgeImportRow } from "../../lib/types";
import { useToast } from "../Toast";

const CATEGORIES = ["规范标准", "工程案例", "经验总结", "材料工艺", "法规政策", "调查报告", "设计方案", "其他"];
const PHASES = ["调查", "设计", "施工", "验收", "运维", "全程"];

// KnowledgeImportModal 知识库「导入文件」：md/txt/docx/pdf/xlsx/csv →
// 候选条目预览（可勾选/修正）→ 确认后批量入库。遵循「无确认不落库」；
// 文档多主题或表格列不规范时可一键切 AI 智能解析（办公功能模型）。
export function KnowledgeImportModal({
  open,
  path,
  fileName,
  onClose,
  onImported,
}: {
  open: boolean;
  path: string;
  fileName: string;
  onClose: () => void;
  onImported: () => void;
}) {
  const [preview, setPreview] = useState<KnowledgeImportPreview | null>(null);
  const [rows, setRows] = useState<KnowledgeImportRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [aiLoading, setAiLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const toast = useToast();

  useEffect(() => {
    if (!open || !path) return;
    setPreview(null);
    setRows([]);
    setError(null);
    setLoading(true);
    app
      .KnowledgeImportPreview(path)
      .then((pv) => {
        setPreview(pv);
        setRows(pv.rows);
      })
      .catch((e) => {
        setError(String(e));
        toast.show(`解析失败：${String(e)}`, "warn");
      })
      .finally(() => setLoading(false));
  }, [open, path, toast]);

  const runAI = useCallback(async () => {
    if (!path) return;
    setAiLoading(true);
    try {
      const pv = await app.KnowledgeImportAIParse(path);
      setPreview(pv);
      setRows(pv.rows);
      setError(null);
      toast.show("AI 智能解析完成，请核对后确认导入", "info");
    } catch (e) {
      setError(`AI 解析失败：${String(e)}`);
      toast.show(`AI 解析失败：${String(e)}`, "warn");
    } finally {
      setAiLoading(false);
    }
  }, [path, toast]);

  const patchRow = useCallback((idx: number, patch: Partial<KnowledgeImportRow>) => {
    setRows((rs) => rs.map((r, i) => (i === idx ? { ...r, ...patch } : r)));
  }, []);

  const toggleSkip = useCallback(
    (idx: number) => setRows((rs) => rs.map((r, i) => (i === idx ? { ...r, skip: !r.skip } : r))),
    [],
  );

  const confirmRows = rows.filter((r) => !r.skip);
  const doApply = useCallback(async () => {
    setSaving(true);
    try {
      const entries: KnowledgeEntry[] = confirmRows.map((r) => ({
        name: r.name,
        title: r.title,
        category: r.category || "其他",
        phase: r.phase,
        discipline: r.discipline,
        tags: r.tags ?? [],
        status: r.status || "现行",
        version: 1,
        author: "",
        reviewer: "",
        source: r.source,
        body: r.body,
      }));
      const n = await app.KnowledgeImportApply(entries);
      toast.show(`已导入 ${n} 条知识条目`, "info");
      onImported();
      onClose();
    } catch (e) {
      toast.show(`导入失败：${String(e)}`, "warn");
    } finally {
      setSaving(false);
    }
  }, [confirmRows, onClose, onImported, toast]);

  return (
    <Modal
      title={
        <span className="flex items-center gap-2">
          <CloudUpload size={14} className="text-accent" />
          导入知识：{fileName}
        </span>
      }
      open={open}
      onCancel={onClose}
      width={900}
      // WebView2 在特定状态下会冻结 rAF/CSS 动画：退出动画永远不结束，
      // 遮罩残留在窗口上导致整个软件点不了。这里禁用弹层动画，关闭即卸载。
      destroyOnHidden
      transitionName=""
      maskTransitionName=""
      footer={
        <div className="flex items-center gap-2">
          <span className="mr-auto text-[11px] text-fg-faint">
            已选 {confirmRows.length} / {rows.length} 条{preview?.unmapped?.length ? ` · 未映射列：${preview.unmapped.join("、")}` : ""}
            {rows.length > 0 && confirmRows.length === 0 ? " · 行均缺少标题或正文，请修正后导入" : ""}
            {rows.length === 0 && error ? " · 解析失败，暂无可导入条目" : ""}
          </span>
          <button
            className="inline-flex items-center gap-1 px-3 h-8 rounded-lg border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors text-[12px] disabled:opacity-50"
            onClick={() => void runAI()}
            disabled={aiLoading || loading}
          >
            <Sparkles size={13} className="text-accent" />
            {aiLoading ? "AI 解析中…" : "AI 智能解析"}
          </button>
          <button
            className="inline-flex items-center gap-1 px-3 h-8 rounded-lg border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors text-[12px] disabled:opacity-50"
            onClick={onClose}
            disabled={saving}
          >
            取消
          </button>
          <button
            className="inline-flex items-center gap-1 px-3 h-8 rounded-lg bg-accent text-white text-[12px] hover:opacity-90 transition-opacity disabled:opacity-50"
            onClick={() => void doApply()}
            disabled={saving || confirmRows.length === 0}
          >
            <BookOpen size={13} />
            {saving ? "导入中…" : `确认导入 ${confirmRows.length} 条`}
          </button>
        </div>
      }
    >
      {error ? (
        <div className="mb-2 px-3 py-2 rounded-lg border border-err/40 bg-err/10 text-fg-dim text-[11.5px]" role="alert">
          <span className="text-err font-medium">解析失败：{error}</span>
          <div className="mt-0.5">
            可点击「AI 智能解析」重试；若仍失败，请检查办公功能模型配置，或更换为
            md/txt/docx/pdf/xlsx/csv 文件后重新导入。
          </div>
        </div>
      ) : null}
      {aiLoading ? (
        <div className="mb-2 px-3 py-2 rounded-lg bg-bg-elev text-fg-dim text-[11.5px] flex items-center gap-2" role="status">
          <Loader size={13} className="animate-spin text-accent shrink-0" />
          <span>
            AI 智能解析中… 正在读取文档并调用模型（通常 30 秒~2 分钟），请稍候。期间可随时点「取消」关闭。
          </span>
        </div>
      ) : null}
      {preview?.message ? (
        <div className="mb-2 px-3 py-2 rounded-lg bg-bg-elev text-fg-dim text-[11.5px]">{preview.message}</div>
      ) : null}
      <div className="max-h-[46vh] overflow-auto rounded-lg border border-border-soft">
        <table className="w-full text-[11.5px]">
          <thead className="sticky top-0 bg-bg-elev text-fg-faint text-left">
            <tr>
              <th className="px-2 py-1.5 w-8">选</th>
              <th className="px-2 py-1.5 min-w-[170px]">标题</th>
              <th className="px-2 py-1.5 w-24">分类</th>
              <th className="px-2 py-1.5 w-20">阶段</th>
              <th className="px-2 py-1.5 w-24">标签</th>
              <th className="px-2 py-1.5 min-w-[130px]">正文预览</th>
              <th className="px-2 py-1.5 w-24">状态</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td colSpan={7} className="px-3 py-8 text-center text-fg-faint">解析中…</td>
              </tr>
            ) : rows.length === 0 ? (
              <tr>
                <td colSpan={7} className="px-3 py-8 text-center text-fg-faint">未识别到有效知识条目</td>
              </tr>
            ) : (
              rows.map((r, i) => (
                <tr key={i} className={`border-t border-border-soft/60 ${r.skip ? "opacity-45" : ""}`}>
                  <td className="px-2 py-1 text-center">
                    <input type="checkbox" checked={!r.skip} onChange={() => toggleSkip(i)} />
                  </td>
                  <td className="px-2 py-1">
                    <input
                      value={r.title}
                      onChange={(e) => patchRow(i, { title: e.target.value })}
                      className="w-full bg-transparent outline-none border-b border-transparent focus:border-accent text-fg"
                    />
                    {r.matchNote && (
                      <span className={`block text-[9.5px] ${r.matchNote === "新增" ? "text-ok" : "text-amber-400"}`}>
                        {r.matchNote}
                      </span>
                    )}
                    {r.similarNote && (
                      <span className="block text-[9.5px] text-red-400" title={r.similarNote}>
                        相似条目
                      </span>
                    )}
                  </td>
                  <td className="px-2 py-1">
                    <select
                      value={r.category}
                      onChange={(e) => patchRow(i, { category: e.target.value })}
                      className="bg-transparent outline-none text-fg-dim text-[11px]"
                    >
                      {CATEGORIES.map((c) => (
                        <option key={c} value={c}>{c}</option>
                      ))}
                    </select>
                  </td>
                  <td className="px-2 py-1">
                    <select
                      value={r.phase}
                      onChange={(e) => patchRow(i, { phase: e.target.value })}
                      className="bg-transparent outline-none text-fg-dim text-[11px]"
                    >
                      <option value="">-</option>
                      {PHASES.map((p) => (
                        <option key={p} value={p}>{p}</option>
                      ))}
                    </select>
                  </td>
                  <td className="px-2 py-1">
                    <input
                      value={(r.tags ?? []).join(",")}
                      onChange={(e) => patchRow(i, { tags: e.target.value.split(",").map((s) => s.trim()).filter(Boolean) })}
                      className="w-24 bg-transparent outline-none border-b border-transparent focus:border-accent text-fg"
                    />
                  </td>
                  <td className="px-2 py-1">
                    <span className="block max-w-[240px] truncate text-fg-faint" title={r.body}>
                      {r.body}
                    </span>
                  </td>
                  <td className="px-2 py-1">
                    <select
                      value={r.status || "现行"}
                      onChange={(e) => patchRow(i, { status: e.target.value })}
                      className="bg-transparent outline-none text-fg-dim text-[11px]"
                    >
                      {["现行", "草稿", "常用", "已归档"].map((s) => (
                        <option key={s} value={s}>{s}</option>
                      ))}
                    </select>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </Modal>
  );
}
