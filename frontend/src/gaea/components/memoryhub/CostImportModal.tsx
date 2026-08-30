import { useCallback, useEffect, useState } from "react";
import { Modal } from "antd";
import { CloudUpload, Coins, Loader, Sparkles } from "../../icons";
import { app } from "../../lib/bridge";
import type { CostEntry, CostImportPreview, CostImportRow } from "../../lib/types";
import { useToast } from "../Toast";

// 识别来源 → 中文标注（source 契约：xlsx/csv/pdf_text/pdf_scan/image）。
const SOURCE_LABELS: Record<string, string> = {
  xlsx: "Excel 表格",
  csv: "CSV 表格",
  pdf_text: "PDF 文本",
  pdf_scan: "PDF 扫描件 OCR",
  image: "图片 OCR",
};

// 扩展名路由：xlsx/csv 走表格解析，PDF/图片走视觉识别管线（扫描件自动 OCR）。
const isVisionFile = (p: string) => /\.(pdf|png|jpe?g|webp)$/i.test(p);

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
  const [error, setError] = useState<string | null>(null);
  const toast = useToast();

  useEffect(() => {
    if (!open || !path) return;
    setPreview(null);
    setRows([]);
    setError(null);
    setLoading(true);
    // xlsx/csv 走表格解析；PDF/图片走视觉识别管线（识别报价单/扫描件 OCR）。
    const loader = isVisionFile(path) ? app.CostImportVisionPreview(path) : app.CostImportPreview(path);
    loader
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
      const pv = await app.CostImportAIParse(path);
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
      const entries: CostEntry[] = confirmRows.map((r) => {
        const cat = r.category || "其他";
        const leaf = cat.split("/").filter(Boolean).pop() ?? "其他";
        return {
          name: r.name,
          title: r.title,
          category: leaf,
          categoryPath: cat,
          unit: r.unit,
          price: r.price,
          laborFee: r.laborFee ?? 0,
          materialFee: r.materialFee ?? 0,
          machineFee: r.machineFee ?? 0,
          managementFee: r.managementFee ?? 0,
          profitFee: r.profitFee ?? 0,
          advanceFee: r.advanceFee ?? 0,
          taxRate: r.taxRate ?? 0,
          components: r.components ?? [],
          body: r.body ?? "",
          spec: r.spec,
          source: r.source,
          sourceRow: r.sourceRow ?? 0,
          tags: [],
          status: r.status || "现行",
        };
      });
      // v4.6 询价飞轮反向接线：PDF/图片报价单确认导入时把来源传给后端，
      // 各行自动幂等写入询价库（source=OCR报价）；xlsx/csv 表格导入不写。
      const visionSources = new Set(["pdf_text", "pdf_scan", "image"]);
      const inquiry = preview?.source && visionSources.has(preview.source) ? ["OCR报价"] : [];
      const n = await app.CostImportApply(entries, ...inquiry);
      toast.show(`已导入 ${n} 条成本条目`, "info");
      onImported();
      onClose();
    } catch (e) {
      toast.show(`导入失败：${String(e)}`, "warn");
    } finally {
      setSaving(false);
    }
  }, [confirmRows, preview?.source, onClose, onImported, toast]);

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
      width={1020}
      // WebView2 在特定状态下会冻结 rAF/CSS 动画：退出动画永远不结束，
      // 遮罩残留在窗口上导致整个软件点不了。这里禁用弹层动画，关闭即卸载。
      destroyOnHidden
      transitionName=""
      maskTransitionName=""
      footer={
        <div className="flex items-center gap-2">
          <span className="mr-auto text-[11px] text-fg-faint">
            已选 {confirmRows.length} / {rows.length} 条{preview?.unmapped?.length ? ` · 未映射列：${preview.unmapped.join("、")}` : ""}
            {rows.length > 0 && confirmRows.length === 0 ? " · 行均缺少名称或有效单价，请修正后导入" : ""}
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
            <Coins size={13} />
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
            表头含名称/单价/单位等列的 xlsx/csv 报价单，或内容清晰的 PDF/图片报价单后重新导入。
          </div>
        </div>
      ) : null}
      {aiLoading ? (
        <div className="mb-2 px-3 py-2 rounded-lg bg-bg-elev text-fg-dim text-[11.5px] flex items-center gap-2" role="status">
          <Loader size={13} className="animate-spin text-accent shrink-0" />
          <span>
            AI 智能解析中… 正在读取表格并调用模型（通常 30 秒~2 分钟），请稍候。期间可随时点「取消」关闭。
          </span>
        </div>
      ) : null}
      {preview?.message ? (
        <div className="mb-2 px-3 py-2 rounded-lg bg-bg-elev text-fg-dim text-[11.5px]">{preview.message}</div>
      ) : null}
      {preview?.source ? (
        <div className="mb-2 flex items-center gap-2 px-3 py-1.5 rounded-lg bg-bg-elev text-fg-dim text-[11px]">
          <span className="text-fg-faint">识别来源</span>
          <span className="px-1.5 py-0.5 rounded bg-accent/15 text-accent text-[10.5px]">
            {SOURCE_LABELS[preview.source] ?? preview.source}
          </span>
          {preview.aiUsed && <span className="text-fg-faint">AI 智能解析</span>}
        </div>
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
              <th className="px-2 py-1.5 w-20">人工</th>
              <th className="px-2 py-1.5 w-20">材料</th>
              <th className="px-2 py-1.5 w-20">机械</th>
              <th className="px-2 py-1.5 w-12">组成</th>
              <th className="px-2 py-1.5 min-w-[110px]">规格</th>
              <th className="px-2 py-1.5 min-w-[90px]">来源</th>
              <th className="px-2 py-1.5 w-28">状态</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td colSpan={8} className="px-3 py-8 text-center text-fg-faint">
                  {isVisionFile(path) ? "正在识别报价单…" : "解析中…"}
                </td>
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
                    <input
                      value={r.category}
                      onChange={(e) => patchRow(i, { category: e.target.value })}
                      placeholder="综合单价/道路/土方"
                      className="w-36 bg-transparent outline-none border-b border-transparent focus:border-accent text-fg-dim text-[11px]"
                    />
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
                      type="number"
                      min={0}
                      step="any"
                      value={r.laborFee ?? ""}
                      onChange={(e) => patchRow(i, { laborFee: Number(e.target.value) || 0 })}
                      className="w-full bg-transparent outline-none border-b border-transparent focus:border-accent text-fg text-right"
                    />
                  </td>
                  <td className="px-2 py-1">
                    <input
                      type="number"
                      min={0}
                      step="any"
                      value={r.materialFee ?? ""}
                      onChange={(e) => patchRow(i, { materialFee: Number(e.target.value) || 0 })}
                      className="w-full bg-transparent outline-none border-b border-transparent focus:border-accent text-fg text-right"
                    />
                  </td>
                  <td className="px-2 py-1">
                    <input
                      type="number"
                      min={0}
                      step="any"
                      value={r.machineFee ?? ""}
                      onChange={(e) => patchRow(i, { machineFee: Number(e.target.value) || 0 })}
                      className="w-full bg-transparent outline-none border-b border-transparent focus:border-accent text-fg text-right"
                    />
                  </td>
                  <td className="px-2 py-1 text-center text-fg-faint tabular-nums" title={r.components?.map((c) => `${c.kind} ${c.title} ${c.amount ?? ""}`).join("\n") ?? ""}>
                    {r.components?.length ?? 0}
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
