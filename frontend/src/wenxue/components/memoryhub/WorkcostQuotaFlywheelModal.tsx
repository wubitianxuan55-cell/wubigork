// WorkcostQuotaFlywheelModal.tsx — 企业定额自生成弹窗（11.6；roadmap 造价愿景#5）
// 跨项目含量分歧聚类预览（orig_code 词干簇，宁缺勿误不成立簇带原因）→ 作者点名
// → 后端重建候选落 ENT-* 企业定额（含量=各变体中位数）。Preview/Apply 收同一
// 词干列表，后端同一纯函数重建——两边天然一致。
import { useEffect, useState } from "react";
import { Alert, Modal, Spin } from "antd";
import { app } from "../../lib/bridge";
import type {
  WorkcostQuotaFlywheelCandidate,
  WorkcostQuotaFlywheelPreview,
  WorkcostQuotaFlywheelResult,
} from "../../lib/types";

interface WorkcostQuotaFlywheelModalProps {
  open: boolean;
  onClose: () => void;
  /** 应用成功回调（父层刷新定额列表）。 */
  onApplied: () => void;
}

const fmtQty = (v: number) =>
  new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 4 }).format(v);

export function WorkcostQuotaFlywheelModal({ open, onClose, onApplied }: WorkcostQuotaFlywheelModalProps) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [preview, setPreview] = useState<WorkcostQuotaFlywheelPreview | null>(null);
  const [result, setResult] = useState<WorkcostQuotaFlywheelResult | null>(null);
  // 选中的聚类词干；预览载入后默认全选（Apply 按钮才是落库确认）。
  const [picked, setPicked] = useState<Set<string>>(new Set());

  const reset = () => {
    setPreview(null);
    setResult(null);
    setError("");
    setPicked(new Set());
  };

  useEffect(() => {
    if (!open) return;
    setBusy(true);
    setError("");
    setPreview(null);
    setResult(null);
    app.CostQuotaFlywheelPreview()
      .then((raw) => {
        const p = raw as unknown as WorkcostQuotaFlywheelPreview;
        setPreview(p);
        setPicked(new Set(p.candidates.map((c) => c.origCode)));
      })
      .catch((e) => setError(e instanceof Error ? e.message : String(e)))
      .finally(() => setBusy(false));
  }, [open]);

  const toggle = (key: string) => {
    setPicked((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  const apply = async () => {
    setBusy(true);
    setError("");
    try {
      const raw = await app.CostQuotaFlywheelApply([...picked]);
      setResult(raw as unknown as WorkcostQuotaFlywheelResult);
      onApplied();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal
      title="自生成企业定额（跨项目含量聚合）"
      open={open}
      onCancel={() => { reset(); onClose(); }}
      width={760}
      footer={null}
      destroyOnHidden
    >
      {busy ? (
        <div style={{ textAlign: "center", padding: 32 }}>
          <Spin tip="正在扫描跨项目含量分歧…" />
        </div>
      ) : result ? (
        <div data-testid="quota-flywheel-result">
          <Alert
            type={result.skipped.length ? "warning" : "success"}
            showIcon
            message={`已生成 ${result.saved.length} 条企业定额${result.saved.length ? `（${result.saved.join("、")}）` : ""}`}
            description={result.skipped.map((s, i) => (
              <div key={i}>「{s.title || s.key}」：{s.reason}</div>
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
            message="同一工作内容在不同项目里的含量分歧（原码-N 变体）聚合成企业定额"
            description="含量取各变体中位数；覆盖度与极差如实显形，过半变体才标「多数派」。落库为 ENT-<原码> 编码（Source=企业聚合），只覆写 ENT-* 自有命名空间，不碰项目数据。"
          />
          {error && <Alert type="error" showIcon message={error} />}
          {preview && (
            <div data-testid="quota-flywheel-preview">
              <Alert
                type={preview.candidates.length ? "success" : "warning"}
                showIcon
                message={`候选 ${preview.candidates.length} 簇 / 不成立 ${preview.skipped.length} 簇（共扫 ${preview.clusters} 簇）`}
              />
              <div style={{ maxHeight: 320, overflowY: "auto", marginTop: 8, fontSize: 12 }}>
                {preview.candidates.map((c) => (
                  <CandidateCard key={c.origCode} cand={c} picked={picked.has(c.origCode)} onToggle={() => toggle(c.origCode)} />
                ))}
                {preview.skipped.map((s) => (
                  <div key={s.key} style={{ padding: "4px 0", color: "var(--color-danger, #c00)", borderBottom: "1px solid var(--border-subtle, #eee)" }}>
                    「{s.title || s.key}」：{s.reason}
                  </div>
                ))}
              </div>
            </div>
          )}
          <div style={{ display: "flex", justifyContent: "flex-end", gap: 8 }}>
            <button type="button" className="px-3 h-7 rounded-lg border border-border text-fg text-[12px]" onClick={() => { reset(); onClose(); }}>
              取消
            </button>
            <button
              type="button"
              className="px-3 h-7 rounded-lg bg-accent text-white text-[12px] disabled:opacity-50"
              disabled={picked.size === 0}
              onClick={() => void apply()}
            >
              生成所选（{picked.size}）
            </button>
          </div>
        </div>
      )}
    </Modal>
  );
}

function CandidateCard({ cand, picked, onToggle }: {
  cand: WorkcostQuotaFlywheelCandidate;
  picked: boolean;
  onToggle: () => void;
}) {
  return (
    <div style={{ padding: "6px 0", borderBottom: "1px solid var(--border-subtle, #eee)" }}>
      <label style={{ display: "flex", alignItems: "center", gap: 6, cursor: "pointer" }}>
        <input type="checkbox" checked={picked} onChange={onToggle} data-testid={`flywheel-pick-${cand.origCode}`} />
        <b>{cand.title}</b>
        <span style={{ color: "var(--color-text-secondary)" }}>
          {cand.unit ? `（${cand.unit}）` : ""} · {cand.members.length} 变体 / {cand.sources} 项目 → {cand.newCode}
        </span>
      </label>
      <div style={{ margin: "3px 0 0 22px", color: "var(--color-text-secondary)" }}>
        {cand.members.map((m) => (
          <div key={m.code}>{m.code} · {m.itemCount} 行 · {m.source}</div>
        ))}
        {cand.rows.map((r) => (
          <div key={r.resourceCode}>
            [{r.kind}] {r.title || r.resourceCode}：中位 {fmtQty(r.quantity)} {r.unit}
            （{r.coverage}/{cand.members.length} 变体，{fmtQty(r.min)}~{fmtQty(r.max)}
            {r.majority ? "，多数派" : "，少数派"}）
          </div>
        ))}
      </div>
    </div>
  );
}
