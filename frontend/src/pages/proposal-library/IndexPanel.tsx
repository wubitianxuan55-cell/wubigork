// IndexPanel.tsx — 方案库板块的「索引」状态钮与体检浮层（2026-11-28 索引优化批）。
// 全文索引行与语义块靠各写入口同步维护，绕开产品改库或早期版本没清干净会留下孤儿行、
// 缺行。平时一眼能看出索引好不好，点开看明细，一键「修复并重建」。
import { useState } from "react";
import { Popover } from "antd";
import { Loader2, Wrench } from "../../gaea/icons";
import type { ProposalIndexStatus } from "../../gaea/lib/types/proposal";
import { indexSummary } from "./logic";

function message(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}

export function IndexPanel({ status, modelDown, onReindex }: {
  status: ProposalIndexStatus | null;
  /** 检索时发现本地语义模型没响应：向量可能明明是齐的，语义检索却是坏的。 */
  modelDown: boolean;
  /** 修复并重建；页面负责调绑定、重读状态。 */
  onReindex: () => Promise<void>;
}) {
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  if (!status) return null;
  const s = indexSummary(status);
  const pending = status.vecTotal - status.vecDone;
  const broken = s.broken || modelDown;
  const tone = broken ? "warn" : pending > 0 ? "partial" : "ok";
  const text = busy
    ? "索引重建中…"
    : broken ? (modelDown ? "语义模型 不可用" : "索引 待修复")
      : pending > 0 ? `语义索引 ${status.vecDone}/${status.vecTotal}` : "语义索引 就绪";

  return (
    <Popover
      trigger="click"
      placement="bottomRight"
      open={open}
      onOpenChange={(next) => { if (!busy) { setOpen(next); if (!next) setError(""); } }}
      destroyOnHidden
      content={(
        <div className="pl-index-pop" data-testid="proposal-index-pop">
          <p className="pl-index-line">语义向量 <strong>{status.vecDone}/{status.vecTotal}</strong>{status.model ? ` · ${status.model}` : ""}</p>
          {pending > 0 && <p className="pl-index-line">还有 {pending} 块没向量化：检索时自动补，或点下面立即补齐。</p>}
          {s.broken && (
            <ul className="pl-index-issues">
              {s.lines.map((line) => <li key={line}>{line}</li>)}
            </ul>
          )}
          {modelDown && (
            <p className="pl-index-line is-warn">
              检索时本地语义模型没有响应，暂时只按关键词匹配。到模型中心确认检索模型在运行，再点「修复并重建」重试。
            </p>
          )}
          {!s.broken && !modelDown && pending === 0 && <p className="pl-index-line">索引完好，没有要修的。</p>}
          {error && <p className="pl-index-line is-warn" role="alert">{error}</p>}
          <button
            type="button"
            className="pl-btn pl-btn-primary"
            data-testid="proposal-reindex"
            disabled={busy}
            onClick={() => {
              setBusy(true);
              setError("");
              void onReindex()
                .catch((e: unknown) => setError(message(e)))
                .finally(() => setBusy(false));
            }}
          >
            {busy ? <Loader2 size={13} className="pl-spin" aria-hidden /> : <Wrench size={12} aria-hidden />}
            修复并重建
          </button>
        </div>
      )}
    >
      <button
        type="button"
        className="pl-index"
        data-tone={tone}
        data-testid="proposal-index-status"
        title="全文索引与语义向量的体检：孤儿行、缺行、没补齐的向量"
      >
        {text}
      </button>
    </Popover>
  );
}
