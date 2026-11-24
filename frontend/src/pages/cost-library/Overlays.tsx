// Overlays — 删除确认与导出菜单。删除是不可恢复动作：确认卡写明对象、规模与
// 后果；后端拒绝（如仍被引用）时把原因留在卡上，不假装成功。
import { useState } from "react";
import { Dropdown, Modal } from "antd";
import { ChevronDown, Download } from "../../gaea/icons";

export interface DeleteRequest {
  /** 确认卡标题。 */
  title: string;
  /** 确认卡正文逐条说明（对象、规模、后果）。 */
  lines: string[];
  /** 确认按钮文案。 */
  confirmLabel: string;
  /** 执行删除；返回失败明细（空=全部成功），由调用方刷新。 */
  run: () => Promise<{ ok: number; failed: string[] }>;
}

export function DeleteDialog({ request, onClose, onDone }: {
  request: DeleteRequest | null;
  onClose: () => void;
  onDone: (summary: string) => void;
}) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const confirm = async () => {
    if (!request) return;
    setBusy(true);
    setError("");
    try {
      const res = await request.run();
      if (res.failed.length > 0) {
        setError(`已删除 ${res.ok} 条，${res.failed.length} 条未删除：${res.failed[0]}`);
        onDone(`已删除 ${res.ok} 条，${res.failed.length} 条失败`);
        return;
      }
      onDone(`已删除 ${res.ok} 条`);
      onClose();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal
      title={request?.title ?? ""}
      open={!!request}
      onCancel={() => { if (!busy) { setError(""); onClose(); } }}
      onOk={() => void confirm()}
      okText={request?.confirmLabel ?? "删除"}
      cancelText="取消"
      confirmLoading={busy}
      okButtonProps={{ danger: true }}
      destroyOnHidden
      transitionName=""
      maskTransitionName=""
      width={460}
    >
      {request && (
        <div className="cl-dialog">
          {request.lines.map((line, i) => <p key={i}>{line}</p>)}
          {error && <p className="cl-dialog-error" role="alert" data-testid="cost-delete-error">{error}</p>}
        </div>
      )}
    </Modal>
  );
}

export interface ExportItem {
  key: string;
  label: string;
  hint?: string;
}

/** 导出菜单：按当前筛选导出 xlsx；结果（路径或失败原因）显示在菜单旁，不弹全局提示。 */
export function ExportMenu({ items, onPick, busy, note, isError }: {
  items: ExportItem[];
  onPick: (key: string) => void;
  busy: boolean;
  note: string;
  isError: boolean;
}) {
  return (
    <span className="cl-export" data-testid="cost-export">
      <Dropdown
        trigger={["click"]}
        disabled={busy}
        menu={{
          items: items.map((it) => ({ key: it.key, label: it.label, title: it.hint })),
          onClick: ({ key }) => onPick(String(key)),
        }}
      >
        <button type="button" className="cl-btn" disabled={busy} aria-label="导出" title="按当前筛选导出 xlsx">
          <Download size={13} aria-hidden />{busy ? "导出中…" : "导出"}<ChevronDown size={11} aria-hidden />
        </button>
      </Dropdown>
      {note && (
        <span className={`cl-export-note${isError ? " is-error" : ""}`} role={isError ? "alert" : "status"} data-testid="cost-export-note">
          {note}
        </span>
      )}
    </span>
  );
}
