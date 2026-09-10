// KnowledgePanel 拆分——合并相似条目弹窗（P3 结构版2 次批巨文件拆分）。
// 内容自 KnowledgePanel.tsx 原文件整体迁出，逐字节照搬；行为零变化。
import { Modal } from "antd";
import type { SimilarView } from "../../lib/types";

/** 合并相似条目弹窗：勾选要并入的相似条目（标签并集、来源合并、旧条目留档删除）。 */
export function MergeModal({ open, target, candidates, selected, onClose, onToggle, onMerge }: {
  open: boolean;
  target: string;
  candidates: SimilarView[];
  selected: Set<string>;
  onClose: () => void;
  onToggle: (name: string) => void;
  onMerge: () => void;
}) {
  return (
    <Modal
      title={`合并相似条目到「${target}」`}
      open={open}
      onCancel={onClose}
      destroyOnHidden
      transitionName=""
      maskTransitionName=""
      footer={
        <div className="flex items-center justify-end gap-2">
          <button className="px-3 h-8 rounded-lg border border-border text-fg-faint hover:text-fg hover:bg-bg-soft text-[12px]" onClick={onClose} type="button">取消</button>
          <button
            className="px-3 h-8 rounded-lg bg-accent text-accent-fg text-[12px] hover:opacity-90 disabled:opacity-50"
            onClick={onMerge}
            disabled={selected.size === 0}
            type="button"
          >
            合并 {selected.size} 条
          </button>
        </div>
      }
      width={520}
    >
      {candidates.length === 0 ? (
        <div className="py-6 text-center text-fg-faint text-[12px]">暂无相似条目</div>
      ) : (
        <div className="space-y-1.5 max-h-[40vh] overflow-auto">
          {candidates.map((s) => (
            <label key={s.name} className="flex items-center gap-2 px-2 py-1.5 rounded-lg bg-bg-soft/40 text-[12px] cursor-pointer">
              <input
                type="checkbox"
                checked={selected.has(s.name)}
                onChange={() => onToggle(s.name)}
              />
              <span className="flex-1 truncate">{s.title}</span>
              <span className="shrink-0 text-fg-faint text-[10.5px]">{Math.round(s.score * 100)}% 相似</span>
            </label>
          ))}
        </div>
      )}
    </Modal>
  );
}