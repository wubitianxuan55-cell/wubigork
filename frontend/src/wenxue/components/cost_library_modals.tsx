// cost_library_modals.tsx — 成本库弹窗展示件（批 40 FE2-02 续刀，自
// CostLibraryView.tsx 原位搬移零逻辑改动）：删除确认/分类新建重命名/价格历史
// 三个内联 antd Modal 上提为受控纯展示件——状态与处理器（deleteName/catModal/
// catName/history* 状态及 handleDelete/saveCategory/openHistory）留主件，
// props 穿线。三个 memoryhub 弹窗（CostEntry/Import/Compare）本就独立文件，
// 不在本批范围。主组件见 CostLibraryView.tsx。
import { Clock } from "../icons";
import { Input, Modal } from "antd";
import type { CostCategory, PriceHistory } from "../lib/types";

// 删除条目确认
export function DeleteCostModal({ name, onCancel, onOk }: {
  name: string | null;
  onCancel: () => void;
  onOk: () => void;
}) {
  return (
    <Modal
      title="删除成本"
      open={!!name}
      onCancel={onCancel}
      onOk={onOk}
      okText="删除"
      destroyOnHidden
      transitionName=""
      maskTransitionName=""
      okButtonProps={{ danger: true }}
      cancelText="取消"
      width={440}
    >
      <p className="text-[13px] text-fg-dim">确定删除成本条目「{name}」吗？</p>
    </Modal>
  );
}

// 分类 新建/重命名
export function CategoryModal({ catModal, name, onNameChange, onSave, onCancel, parentPath }: {
  catModal: { mode: "create" | "rename"; parentId: number; node: CostCategory | null } | null;
  name: string;
  onNameChange: (v: string) => void;
  onSave: () => void;
  onCancel: () => void;
  parentPath: string;
}) {
  return (
    <Modal
      title={catModal?.mode === "rename" ? "重命名分类" : "新建分类"}
      open={!!catModal}
      onCancel={onCancel}
      onOk={onSave}
      okText={catModal?.mode === "rename" ? "保存" : "创建"}
      destroyOnHidden
      transitionName=""
      maskTransitionName=""
      cancelText="取消"
      width={420}
    >
      <div className="space-y-2 py-1">
        {catModal && catModal.mode === "create" && catModal.parentId > 0 && (
          <div className="text-[11.5px] text-fg-faint">
            父分类：<span className="text-fg">{parentPath}</span>
          </div>
        )}
        <Input
          value={name}
          onChange={(e) => onNameChange(e.target.value)}
          placeholder="分类名称，如 钢材 / 桩基机械"
          autoFocus
          onPressEnter={onSave}
          className="!text-[13px]"
        />
      </div>
    </Modal>
  );
}

// 价格历史
export function PriceHistoryModal({ open, name, rows, onClose, priceText }: {
  open: boolean;
  name: string;
  rows: PriceHistory[];
  onClose: () => void;
  priceText: (p: number) => string;
}) {
  return (
    <Modal
      title={`价格历史：${name}`}
      open={open}
      onCancel={onClose}
      footer={null}
      destroyOnHidden
      transitionName=""
      maskTransitionName=""
      width={520}
    >
      {rows.length === 0 ? (
        <div className="py-6 text-center text-fg-faint text-[12px]">暂无价格历史（保存时内容变化会自动留档）</div>
      ) : (
        <div className="space-y-1.5 max-h-[46vh] overflow-auto">
          {rows.map((h, i) => (
            <div key={i} className="flex items-center gap-2 px-2 py-1.5 rounded-lg bg-bg-soft/40 text-[12px]">
              <Clock size={12} className="text-fg-faint shrink-0" />
              <span className="text-fg font-medium tabular-nums">{priceText(h.price)}{h.unit ? `/${h.unit}` : ""}</span>
              {h.period && <span className="text-fg-faint">{h.period}</span>}
              {h.source && <span className="text-fg-faint truncate">来源：{h.source}</span>}
              <span className="ml-auto text-fg-faint text-[10.5px] shrink-0">
                {h.fetchedAt ? new Date(h.fetchedAt).toLocaleString("zh-CN", { hour12: false }) : ""}
              </span>
            </div>
          ))}
        </div>
      )}
    </Modal>
  );
}
