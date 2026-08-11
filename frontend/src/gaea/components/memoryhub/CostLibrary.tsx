import { useState } from "react";
import { CostLibraryView } from "../CostLibraryView";
import { PriceSourcesPanel } from "./PriceSourcesPanel";
import { PriceSourcesRepository } from "./PriceSourcesRepository";

/**
 * CostLibrary 成本库（记忆中枢扩展库）：
 * 多级分类树 + 列表/表格双视图（详见 CostLibraryView），
 * 另提供「价格源 / 价格源仓库」两个子视图。
 */
export function CostLibrary() {
  const [mode, setMode] = useState<"list" | "sources" | "repository">("list");

  return (
    <div className="h-full flex flex-col">
      <div className="shrink-0 flex items-center gap-1 px-4 pt-3">
        {(["list", "sources", "repository"] as const).map((m) => (
          <button
            key={m}
            onClick={() => setMode(m)}
            className={`px-2.5 h-7 rounded-full text-[11.5px] transition-colors ${
              mode === m ? "bg-accent text-white" : "bg-bg-elev text-fg-faint hover:text-fg border border-border"
            }`}
          >
            {m === "list" ? "成本条目" : m === "sources" ? "价格源" : "价格源仓库"}
          </button>
        ))}
      </div>
      <div className="flex-1 min-h-0">
        {mode === "sources" ? (
          <PriceSourcesPanel />
        ) : mode === "repository" ? (
          <PriceSourcesRepository />
        ) : (
          <CostLibraryView />
        )}
      </div>
    </div>
  );
}
