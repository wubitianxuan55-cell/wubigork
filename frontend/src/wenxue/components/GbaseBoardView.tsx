import { useMemo, useState } from "react";
import { applyGbaseView, gbaseRowColor, type GbaseSheetModel, type GbaseView } from "../lib/gbase";

// GbaseBoardView — 多维表看板视图（B2 首刀）：groupBy 列值=泳道、行=卡片。
// 数据/筛选/排序/着色复用 lib/gbase 纯函数（与分组视图同源 applyGbaseView）。
// 拖拽改值：卡片拖到目标泳道 = 该行 groupBy 列单元格改值为泳道值（「（空）」
// 泳道=清空），经宿主传入的 onMoveCard 走 XlsxSetCell 直编通道——与表格视图
// 双击直编同口径；拖拽中宿主负责禁用重复触发与刷新预览。
const MAX_CARD_ROWS = 5;

export function GbaseBoardView({
  model,
  view,
  onMoveCard,
  moving,
}: {
  model: GbaseSheetModel;
  view: GbaseView;
  onMoveCard?: (rowIndex: number, laneKey: string) => void;
  moving?: boolean;
}) {
  const { groups, filteredOut } = useMemo(() => applyGbaseView(model, view), [model, view]);
  const [dragOver, setDragOver] = useState<string | null>(null);
  const [dragging, setDragging] = useState<number | null>(null);

  // 卡面字段：cardFields ∩ 实有字段；缺省取前 4 个字段
  const cardFields = useMemo(() => {
    const base = view.cardFields ?? model.fields.slice(0, 4);
    return base.filter((f) => model.fields.includes(f)).slice(0, 8);
  }, [model.fields, view.cardFields]);

  const laneOf = (rowIndex: number): string | null => {
    for (const g of groups) {
      if (g.records.some((r) => r.rowIndex === rowIndex)) return g.key;
    }
    return null;
  };

  const handleDrop = (laneKey: string) => {
    setDragOver(null);
    if (dragging === null || moving || !onMoveCard) return;
    const from = laneOf(dragging);
    setDragging(null);
    if (from === null || from === laneKey) return; // 原地放下不写盘
    onMoveCard(dragging, laneKey);
  };

  const total = groups.reduce((s, g) => s + g.records.length, 0);
  const maxLane = Math.max(1, ...groups.map((g) => g.records.length));

  return (
    <div className="p-3 h-full overflow-auto" data-testid="gbase-board">
      <div className="flex gap-2 items-start">
        {groups.map((g) => (
          <div
            key={g.key}
            data-testid="gbase-lane"
            data-lane-key={g.key}
            className="flex-1 min-w-[160px] max-w-[280px] rounded-lg bg-bg-soft/40 p-1.5"
            style={{
              outline: dragOver === g.key ? "2px dashed var(--accent, #6366f1)" : "none",
              minHeight: 80 + maxLane * 8,
            }}
            onDragOver={(e) => {
              e.preventDefault();
              setDragOver(g.key);
            }}
            onDragLeave={() => setDragOver((prev) => (prev === g.key ? null : prev))}
            onDrop={(e) => {
              e.preventDefault();
              handleDrop(g.key);
            }}
          >
            <div className="px-1.5 py-1 text-[11px] text-fg font-medium flex items-center gap-1">
              <span className="truncate">{g.key || "（空）"}</span>
              <span className="text-fg-faint shrink-0">· {g.records.length}</span>
            </div>
            {g.records.map((r) => {
              const color = gbaseRowColor(r, view);
              const rows = cardFields.map((f) => [f, r.cells[f] ?? ""] as const).filter(([, v]) => v.trim() !== "");
              return (
                <div
                  key={r.rowIndex}
                  data-testid="gbase-card"
                  data-row-index={r.rowIndex}
                  draggable={!moving}
                  onDragStart={() => setDragging(r.rowIndex)}
                  onDragEnd={() => {
                    setDragging(null);
                    setDragOver(null);
                  }}
                  className="rounded-md bg-bg-elevated px-2 py-1.5 mb-1.5 cursor-grab active:cursor-grabbing select-none"
                  style={{
                    border: "1px solid rgba(128,128,140,0.16)",
                    background: color ?? undefined,
                    opacity: dragging === r.rowIndex ? 0.5 : undefined,
                  }}
                >
                  {rows.length === 0 ? (
                    <span className="text-fg-faint text-[11px]">（空行）</span>
                  ) : (
                    <>
                      <div className="text-[12px] text-fg font-medium truncate" title={rows[0]![1]}>
                        {rows[0]![1]}
                      </div>
                      {rows.slice(1, MAX_CARD_ROWS).map(([f, v]) => (
                        <div key={f} className="text-[10.5px] text-fg-faint truncate" title={`${f}：${v}`}>
                          <span className="shrink-0">{f}：</span>
                          {v}
                        </div>
                      ))}
                      {rows.length > MAX_CARD_ROWS && (
                        <div className="text-[10px] text-fg-faint">+{rows.length - MAX_CARD_ROWS} 字段</div>
                      )}
                    </>
                  )}
                </div>
              );
            })}
            {g.records.length === 0 && (
              <div className="text-[10.5px] text-fg-faint px-1.5 py-2">拖卡片到此泳道</div>
            )}
          </div>
        ))}
        {groups.length === 0 && (
          <div className="p-6 text-center text-fg-faint w-full">没有符合条件的记录</div>
        )}
      </div>
      <div className="flex items-center gap-1.5 px-1 py-2 text-[10px] text-fg-faint flex-wrap">
        <span>看板 · 泳道「{view.groupBy}」</span>
        <span>· 共 {total} 条</span>
        {filteredOut > 0 && <span> · 筛选隐藏 {filteredOut} 条</span>}
        <span> · 拖动卡片到泳道即改「{view.groupBy}」单元格值（直编通道）</span>
      </div>
    </div>
  );
}
