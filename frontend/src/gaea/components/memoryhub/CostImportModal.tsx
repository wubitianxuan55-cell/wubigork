import { useCallback, useEffect, useState } from "react";
import { Modal } from "antd";
import { CloudUpload, Coins, Sparkles } from "../../icons";
import { app } from "../../lib/bridge";
import type { CostEntry, CostImportPreview, CostImportRow } from "../../lib/types";
import { useToast } from "../Toast";

const CATEGORIES = ["机械", "材料", "人工", "运输", "检测", "其他"];

// CostImportModal 成本库「导入文件」：解析 xlsx/csv 报价单/测算表 →
// 候选条目预览（可勾选/修正）→ 确认后批量入库。遵循「无确认不落库」；
// 表头识别不理想时可一键切到 AI 智能解析（办公功能模型归一化）。
export function CostImportModal({
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
  const [preview, setPreview] = useState<CostImportPreview | null>(null);
  const [rows, setRows] = useState<CostImportRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [aiLoading, setAiLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const toast = useToast();

  useEffect(() => {
    if (!open || !path) return;
    setPreview(null);
    setRows([]);
    setLoading(true);
    app
      .CostImportPreview(path)
      .then((pv) => {
        setPreview(pv);
        setRows(pv.rows);
      })
      .catch((e) => toast.show(`解析失败：${String(e)}`, "warn"))
      .finally(() => setLoading(false));
  }, [open, path, toast]);

  const runAI = useCallback(async () => {
    if (!path) return;
    setAiLoading(true);
    try {
      const pv = await app.CostImportAIParse(path);
      setPreview(pv);
      setRows(pv.rows);
      toast.show("AI 智能解析完成，请核对后确认导入", "info");
    } catch (e) {
      toast.show(`AI 解析失败：${String(e)}`, "warn");
    } finally {
      setAiLoading(false);
    }
  }, [path, toast]);

  const patchRow = useCallback((idx: number, patch: Partial<CostImportRow>) => {
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
      const entries: CostEntry[] = confirmRows.map((r) => ({
        name: r.name,
        title: r.title,
        category: r.category || "其他",
        unit: r.unit,
        price: r.price,
        spec: r.spec,
        source: r.source,
        tags: [],
        status: r.status || "现行",
        body: "",
        updatedAt: "",
        createdAt: "",
      }));
      const n = await app.CostImportApply(entries);
      toast.show(`已导入 ${n} 条成本条目`, "info");
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
          <CloudUpload size={14} className="text-amber-400" />
          导入成本：{fileName}
        </span>
      }
      open={open}
      onCancel={onClose}
      width={860}
      footer={
        <div className="flex items-center gap-2">
          <span className="mr-auto text-[11px] text-fg-faint">
            已选 {confirmRows.length} / {rows.length} 条{preview?.unmapped?.length ? ` · 未映射列：${preview.unmapped.join("、")}` : ""}
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
            <Coins size={13} />
            {saving ? "导入中…" : `确认导入 ${confirmRows.length} 条`}
          </button>
        </div>
      }
    >
      {preview?.message ? (
        <div className="mb-2 px-3 py-2 rounded-lg bg-bg-elev text-fg-dim text-[11.5px]">{preview.message}</div>
      ) : null}
      <div className="max-h-[46vh] overflow-auto rounded-lg border border-border-soft">
        <table className="w-full text-[11.5px]">
          <thead className="sticky top-0 bg-bg-elev text-fg-faint text-left">
            <tr>
              <th className="px-2 py-1.5 w-8">选</th>
              <th className="px-2 py-1.5 min-w-[160px]">名称</th>
              <th className="px-2 py-1.5 w-20">分类</th>
              <th className="px-2 py-1.5 w-20">单位</th>
              <th className="px-2 py-1.5 w-24">单价(元)</th>
              <th className="px-2 py-1.5 min-w-[110px]">规格</th>
              <th className="px-2 py-1.5 min-w-[90px]">来源</th>
              <th className="px-2 py-1.5 w-28">状态</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td colSpan={8} className="px-3 py-8 text-center text-fg-faint">解析中…</td>
              </tr>
            ) : rows.length === 0 ? (
              <tr>
                <td colSpan={8} className="px-3 py-8 text-center text-fg-faint">未识别到有效成本行</td>
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
                    <input
                      value={r.unit}
                      onChange={(e) => patchRow(i, { unit: e.target.value })}
                      className="w-16 bg-transparent outline-none border-b border-transparent focus:border-accent text-fg"
                    />
                  </td>
                  <td className="px-2 py-1">
                    <input
                      type="number"
                      min={0}
                      step="any"
                      value={Number.isFinite(r.price) ? r.price : ""}
                      onChange={(e) => patchRow(i, { price: Number(e.target.value) || 0 })}
                      className="w-full bg-transparent outline-none border-b border-transparent focus:border-accent text-fg text-right"
                    />
                  </td>
                  <td className="px-2 py-1">
                    <input
                      value={r.spec}
                      onChange={(e) => patchRow(i, { spec: e.target.value })}
                      className="w-full bg-transparent outline-none border-b border-transparent focus:border-accent text-fg"
                    />
                  </td>
                  <td className="px-2 py-1">
                    <input
                      value={r.source}
                      onChange={(e) => patchRow(i, { source: e.target.value })}
                      className="w-full bg-transparent outline-none border-b border-transparent focus:border-accent text-fg"
                    />
                  </td>
                  <td className="px-2 py-1">
                    <select
                      value={r.status || "现行"}
                      onChange={(e) => patchRow(i, { status: e.target.value })}
                      className="bg-transparent outline-none text-fg-dim text-[11px]"
                    >
                      {["现行", "草稿", "已归档"].map((s) => (
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
