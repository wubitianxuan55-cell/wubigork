// WorkcostBulkPriceModal.tsx — 资源价批量调价弹窗（三期 3.4，master plan 原案）
// Excel 上传 → 解析预览（匹配/跳过分区，宁缺勿误）→ 作者确认 → 整批事务应用
// （写调价历史+推现行价）。xlsxBase64 传输（进度计划导入先例）；Preview 与
// Apply 收同一文件，后端重解析+重匹配幂等——同一文件同一匹配结果。
import { useRef, useState } from "react";
import { Alert, Modal, Spin } from "antd";
import { app } from "../../lib/bridge";
import type {
  WorkcostBulkPricePreview,
  WorkcostBulkPriceResult,
} from "../../lib/types";

interface WorkcostBulkPriceModalProps {
  open: boolean;
  onClose: () => void;
  /** 应用成功回调（父层刷新资源表）。 */
  onApplied: () => void;
}

const fmtPrice = (p: number) =>
  "¥" + new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 2 }).format(p);

const toB64 = (buf: ArrayBuffer) =>
  btoa(new Uint8Array(buf).reduce((s, b) => s + String.fromCharCode(b), ""));

export function WorkcostBulkPriceModal({ open, onClose, onApplied }: WorkcostBulkPriceModalProps) {
  const fileRef = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [preview, setPreview] = useState<WorkcostBulkPricePreview | null>(null);
  const [result, setResult] = useState<WorkcostBulkPriceResult | null>(null);
  // 已预览文件的 b64（Apply 复用同一文件：后端重解析+重匹配幂等）。
  const [bulkB64, setBulkB64] = useState("");

  const reset = () => {
    setPreview(null);
    setResult(null);
    setError("");
    setBulkB64("");
  };

  const pickFile = async (file: File) => {
    setBusy(true);
    setError("");
    setPreview(null);
    setResult(null);
    try {
      const b64 = toB64(await file.arrayBuffer());
      const raw = await app.CostBulkPricePreview(b64);
      setBulkB64(b64);
      setPreview(raw as unknown as WorkcostBulkPricePreview);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const apply = async () => {
    setBusy(true);
    setError("");
    try {
      const raw = await app.CostBulkPriceApply(bulkB64);
      setResult(raw as unknown as WorkcostBulkPriceResult);
      onApplied();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal
      title="批量调价（Excel 导入）"
      open={open}
      onCancel={() => { reset(); onClose(); }}
      width={720}
      footer={null}
      destroyOnHidden
    >
      {busy ? (
        <div style={{ textAlign: "center", padding: 32 }}>
          <Spin tip="正在解析与匹配…" />
        </div>
      ) : result ? (
        <div data-testid="bulk-price-result">
          <Alert
            type={result.skipped.length ? "warning" : "success"}
            showIcon
            message={`已更新 ${result.updated} 条（共 ${result.total} 行，跳过 ${result.skipped.length} 条）`}
            description={result.skipped.map((s, i) => (
              <div key={i}>第 {s.row} 行「{s.title}」：{s.reason}</div>
            ))}
          />
          <div style={{ marginTop: 12, textAlign: "right" }}>
            <button type="button" className="px-3 h-7 rounded-lg border border-border text-fg text-[12px]" onClick={() => { reset(); onClose(); }}>
              关闭
            </button>
          </div>
        </div>
      ) : (
        <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
          <Alert
            type="info"
            showIcon
            message="Excel 列：名称 / 单位 / 新价 / 期数（规格、地区、备注可选）"
            description="按「名称+单位(+规格)」精确匹配工料机资源——宁缺勿误：未命中或同身份多行的行会跳过并列出原因，绝不模糊猜。确认后整批写入（调价历史+现行价）。"
          />
          {error && <Alert type="error" showIcon message={error} />}
          {preview && (
            <div data-testid="bulk-price-preview">
              <Alert
                type={preview.skipped.length ? "warning" : "success"}
                showIcon
                message={`匹配 ${preview.matched.length} 条 / 跳过 ${preview.skipped.length} 条（共 ${preview.total} 行）`}
              />
              <div style={{ maxHeight: 260, overflowY: "auto", marginTop: 8, fontSize: 12 }}>
                {preview.matched.map((m) => (
                  <div key={`m${m.row}`} style={{ padding: "3px 0", borderBottom: "1px solid var(--border-subtle, #eee)" }}>
                    <b>{m.title}</b>
                    {m.spec ? <span style={{ color: "var(--color-text-secondary)" }}>（{m.spec}）</span> : null}
                    ：{fmtPrice(m.oldPrice)} → <b>{fmtPrice(m.newPrice)}</b>
                    {m.period ? <span style={{ color: "var(--color-text-secondary)" }}> · {m.period}</span> : null}
                  </div>
                ))}
                {preview.skipped.map((s) => (
                  <div key={`s${s.row}`} style={{ padding: "3px 0", color: "var(--color-danger, #c00)", borderBottom: "1px solid var(--border-subtle, #eee)" }}>
                    第 {s.row} 行「{s.title}」：{s.reason}
                  </div>
                ))}
              </div>
            </div>
          )}
          <div style={{ display: "flex", justifyContent: "flex-end", gap: 8 }}>
            <input
              ref={fileRef}
              type="file"
              accept=".xlsx"
              style={{ display: "none" }}
              onChange={(e) => {
                const f = e.target.files?.[0];
                if (f) void pickFile(f);
                e.target.value = "";
              }}
            />
            <button type="button" className="px-3 h-7 rounded-lg border border-border text-fg text-[12px]" onClick={() => fileRef.current?.click()}>
              选择 Excel
            </button>
            <button
              type="button"
              className="px-3 h-7 rounded-lg bg-accent text-accent-fg text-[12px] disabled:opacity-50"
              data-testid="bulk-price-apply"
              disabled={!preview || preview.matched.length === 0 || busy}
              onClick={apply}
            >
              确认写入（{preview?.matched.length ?? 0} 条）
            </button>
          </div>
        </div>
      )}
    </Modal>
  );
}

export default WorkcostBulkPriceModal;
