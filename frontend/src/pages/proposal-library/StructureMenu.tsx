// StructureMenu.tsx — 阅读窗的「调整结构」菜单（2026-11-28 编辑优化批）：升降级、
// 上下移、并入、新增与删除小节。这里只列菜单项与可用态；要不要确认、新增小节叫
// 什么名字，由页面做。拆分要取编辑框里的光标，放在编辑态里，不在这个菜单。
import { useEffect, useRef, useState } from "react";
import { ListTree } from "../../gaea/icons";
import type { ProposalSectionOp } from "../../gaea/lib/types/proposal";
import { STRUCTURE_ITEMS } from "./structureItems";
import { formatChars, type StructureFlags } from "./logic";

export function StructureMenu({ flags, onPick }: {
  flags: StructureFlags;
  onPick: (op: ProposalSectionOp) => void;
}) {
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const label = flags.subtreeCount > 1 ? `删除本节（连同 ${flags.subtreeCount - 1} 个子节，${formatChars(flags.subtreeChars)}）` : "删除本节";

  useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    document.addEventListener("mousedown", onDown);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDown);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  return (
    <div ref={rootRef} className="pl-menu-root">
      <button
        type="button"
        className="pl-btn pl-btn-ghost"
        aria-haspopup="menu"
        aria-expanded={open}
        data-testid="proposal-structure-open"
        title="调整标题层级、顺序，或拆分、并入、增删小节"
        onClick={() => setOpen((v) => !v)}
      >
        <ListTree size={12} aria-hidden />调整结构
      </button>
      {open && (
        <div className="pl-menu" role="menu" aria-label="调整结构" data-testid="proposal-structure-menu">
          {STRUCTURE_ITEMS.map((item) => {
            const usable = item.enabled(flags);
            return (
              <button
                key={item.op}
                type="button"
                role="menuitem"
                className={`pl-menu-item${item.danger ? " is-danger" : ""}`}
                data-testid={`proposal-structure-${item.op}`}
                disabled={!usable}
                title={usable ? item.hint : item.why}
                onClick={() => {
                  setOpen(false);
                  onPick(item.op);
                }}
              >
                <span>{item.op === "remove" ? label : item.label}</span>
                <small>{item.hint}</small>
              </button>
            );
          })}
          <p className="pl-menu-note">调整层级或位置后标题里的编号不会改；与新层级不符时标题旁会有提示。</p>
        </div>
      )}
    </div>
  );
}
