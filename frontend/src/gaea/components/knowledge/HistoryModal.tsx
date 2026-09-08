// KnowledgePanel 拆分——版本历史弹窗（P3 结构版2 次批巨文件拆分）。
// 内容自 KnowledgePanel.tsx 原文件整体迁出，逐字节照搬；行为零变化。
import { Modal } from "antd";
import type { KnowledgeHistoryView } from "../../lib/types";

/** 版本历史弹窗：保存（内容变化）自动留档的逐版本快照。 */
export function HistoryModal({ open, name, rows, onClose }: {
  open: boolean;
  name: string;
  rows: KnowledgeHistoryView[];
  onClose: () => void;
}) {
  return (
    <Modal
      title={`版本历史：${name}`}
      open={open}
      onCancel={onClose}
      footer={null}
      width={560}
      destroyOnHidden
      transitionName=""
      maskTransitionName=""
    >
      <div className="space-y-2 max-h-[46vh] overflow-auto">
        {rows.length === 0 ? (
          <div className="py-6 text-center text-fg-faint text-[12px]">暂无版本历史（保存时内容变化会自动留档）</div>
        ) : (
          rows.map((h, i) => (
            <div key={i} className="p-2 rounded-lg bg-bg-soft/40 text-[12px]">
              <div className="flex items-center gap-2">
                <span className="font-semibold text-fg">v{h.version}</span>
                <span className="text-fg-faint">{h.note}</span>
                <span className="ml-auto text-fg-faint text-[10.5px]">{h.changedAt ? new Date(h.changedAt).toLocaleString("zh-CN", { hour12: false }) : ""}</span>
              </div>
              <div className="mt-1 text-fg-dim whitespace-pre-wrap break-words max-h-[120px] overflow-y-auto">{h.body}</div>
            </div>
          ))
        )}
      </div>
    </Modal>
  );
}