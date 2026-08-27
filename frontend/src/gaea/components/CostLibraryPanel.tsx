import { useCallback, useMemo, useState } from "react";
import { Coins, CloudUpload } from "../icons";
import { useComposerInsertStore } from "../lib/store";
import type { CostSummary } from "../lib/types";
import { CostLibraryView } from "./CostLibraryView";
import { PriceSourcesPanel } from "./memoryhub/PriceSourcesPanel";
import { PriceSourcesRepository } from "./memoryhub/PriceSourcesRepository";
import { useToast } from "./Toast";

/**
 * CostLibraryPanel — 办公右侧「成本库」Tab：窄面板复用 CostLibraryView
 * （多级分类路径筛选 + 列表视图），并提供「插入输入框」快捷引用。
 */
export function CostLibraryPanel() {
  const [mode, setMode] = useState<"list" | "sources" | "repository">("list");
  const toast = useToast();

  const priceText = useMemo(() => {
    const fmt = new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 2 });
    return (p: number) => "¥" + fmt.format(p);
  }, []);

  // 一键插入：把单条成本条目转成紧凑结构化行进入输入框（不 dump 整库）。
  const insert = useCallback(
    (e: CostSummary) => {
      const lines = [
        `【成本库】${e.title}`,
        `- name: ${e.name}`,
        `- 分类: ${e.categoryPath || e.category} | 单价: ${priceText(e.price)}${e.unit ? "/" + e.unit : ""} | 规格: ${e.spec || "-"}`,
        `- 来源: ${e.source || "-"} | 状态: ${e.status || "现行"}`,
      ];
      useComposerInsertStore.getState().requestText(lines.join("\n"));
      toast.show(`已把「${e.title}」插入输入框`, "info");
    },
    [priceText, toast],
  );

  const switchMode = (m: "list" | "sources" | "repository") => setMode(m);

  return (
    <div className="flex flex-col h-full text-fg-dim text-xs">
      <div className="flex items-center justify-between px-3 py-2 border-b border-border-soft">
        <span className="flex items-center gap-1.5 font-semibold text-fg text-sm">
          <Coins size={13} className="text-amber-400" />
          成本库
        </span>
        <div className="flex items-center gap-1">
          <button
            className={`px-2 h-6 rounded-full text-[10.5px] transition-colors ${
              mode === "sources" ? "bg-accent text-white" : "text-fg-faint hover:text-fg"
            }`}
            onClick={() => switchMode(mode === "sources" ? "list" : "sources")}
            title="价格源：定时抓取价格更新"
          >
            <CloudUpload size={12} />
          </button>
          {mode === "sources" && (
            <button
              className="px-2 h-6 rounded-full text-[10.5px] transition-colors text-fg-faint hover:text-fg"
              onClick={() => switchMode("repository")}
              title="价格源仓库"
            >
              仓库
            </button>
          )}
          {mode === "repository" && (
            <button
              className="px-2 h-6 rounded-full text-[10.5px] transition-colors bg-accent text-white"
              onClick={() => switchMode("sources")}
              title="返回价格源"
            >
              仓库
            </button>
          )}
        </div>
      </div>
      <div className="flex-1 min-h-0">
        {mode === "sources" ? (
          <PriceSourcesPanel />
        ) : mode === "repository" ? (
          <PriceSourcesRepository />
        ) : (
          <CostLibraryView compact onInsert={insert} />
        )}
      </div>
    </div>
  );
}
