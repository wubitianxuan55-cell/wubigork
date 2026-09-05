import { Fragment, useEffect, useMemo, useState } from "react";
import { ChevronDown, ChevronRight } from "../icons";
import { applyGbaseView, gbaseRowColor, type GbaseSheetModel, type GbaseView } from "../lib/gbase";

const MAX_FIELDS = 30;
const ROW_H = 28;

// GbaseGroupedView — 多维表分组视图（B1 v1：分组/筛选/排序/条件着色的浏览层）。
// 数据与计算全部来自 lib/gbase 纯函数；此处只做呈现：
//   - groupBy 列值分块（首现顺序，空值归「（空）」），块头可折叠；
//   - 行背景按 colorRules 首条命中着色（色值来自配置，渲染不引入新硬编码）；
//   - 只读浏览：编辑数据请切回「表格」视图（Plan→Apply / 直编通道不变）。
export function GbaseGroupedView({ model, view }: { model: GbaseSheetModel; view: GbaseView }) {
  const { groups, filteredOut } = useMemo(() => applyGbaseView(model, view), [model, view]);
  const fields = useMemo(() => model.fields.slice(0, MAX_FIELDS), [model.fields]);
  const [collapsed, setCollapsed] = useState<ReadonlySet<string>>(() => new Set<string>());

  // 切视图重置折叠态（分组键含义随视图变化）
  useEffect(() => {
    setCollapsed(new Set<string>());
  }, [view]);

  const total = groups.reduce((s, g) => s + g.records.length, 0);

  return (
    <div className="p-3 h-full overflow-auto" data-testid="gbase-grouped">
      <table className="border-collapse text-[12px] leading-tight w-full">
        <thead>
          <tr>
            {fields.map((f) => (
              <th
                key={f}
                className="sticky top-0 z-20 px-2 py-1 text-left text-[10.5px] font-normal text-fg-faint whitespace-nowrap"
                style={{ background: "var(--bg-elevated, #181b21)", border: "1px solid rgba(128,128,140,0.16)" }}
              >
                {f}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {groups.map((g) => {
            const isCollapsed = collapsed.has(g.key);
            return (
              <Fragment key={g.key}>
                <tr data-testid="gbase-group" data-group-key={g.key}>
                  <td
                    colSpan={fields.length}
                    className="px-2 py-1 cursor-pointer select-none bg-bg-soft/60 text-fg-dim"
                    style={{ border: "1px solid rgba(128,128,140,0.16)", height: ROW_H }}
                    onClick={() =>
                      setCollapsed((prev) => {
                        const next = new Set(prev);
                        if (next.has(g.key)) next.delete(g.key);
                        else next.add(g.key);
                        return next;
                      })
                    }
                  >
                    <span className="inline-flex items-center gap-1">
                      {isCollapsed ? <ChevronRight size={10} /> : <ChevronDown size={10} />}
                      <span className="text-fg font-medium">{g.key || "（全部）"}</span>
                      <span className="text-fg-faint">· {g.records.length} 条</span>
                    </span>
                  </td>
                </tr>
                {!isCollapsed &&
                  g.records.map((r) => {
                    const color = gbaseRowColor(r, view);
                    return (
                      <tr key={`${g.key}-${r.rowIndex}`} style={{ height: ROW_H }}>
                        {fields.map((f) => (
                          <td
                            key={f}
                            className="px-2 py-1 text-fg whitespace-nowrap overflow-hidden text-ellipsis max-w-[240px]"
                            style={{
                              border: "1px solid rgba(128,128,140,0.16)",
                              background: color ?? undefined,
                              fontVariantNumeric: "tabular-nums",
                            }}
                            title={r.cells[f] ?? ""}
                          >
                            {r.cells[f] ?? ""}
                          </td>
                        ))}
                      </tr>
                    );
                  })}
              </Fragment>
            );
          })}
          {groups.length === 0 && (
            <tr>
              <td colSpan={Math.max(1, fields.length)} className="p-6 text-center text-fg-faint">
                没有符合条件的记录
              </td>
            </tr>
          )}
        </tbody>
      </table>
      <div className="flex items-center gap-1.5 px-1 py-2 text-[10px] text-fg-faint flex-wrap">
        {view.groupBy ? <span>已按「{view.groupBy}」分组 · </span> : null}
        <span>共 {total} 条</span>
        {filteredOut > 0 && <span> · 筛选隐藏 {filteredOut} 条</span>}
        {model.fields.length > MAX_FIELDS && <span> · 仅展示前 {MAX_FIELDS} 列</span>}
        <span> · 编辑数据请切回「表格」视图</span>
      </div>
    </div>
  );
}
