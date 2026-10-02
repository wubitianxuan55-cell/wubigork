import { useCallback } from "react";
import { Coins, Copy, ExternalLink, Pencil, Trash2 } from "../../icons";
import { openExternal } from "../../lib/bridge";
import { classifyExternalLink } from "../../lib/browserPolicy";
import type { PriceSource } from "../../lib/types";
import { useToast } from "../Toast";
import { freqLabel, timeText } from "./usePriceSources";

// PriceSourceCard 单个价格源卡片（FE2-06 抽取，价格源页与阅览仓库共用）：
// 名称/频率徽标/启用态/最近抓取/抓取地址 + 复制/外链操作单源实现。两容器历史
// 漂移按 variant 参数化，DOM 与抽取前逐容器等价：
// - variant="panel"（价格源页）：Coins 图标；仅停用时出灰色徽标；编辑/删除在
//   头部行（w-6 h-6）；「抓取」按钮；地址行操作 hover 显现。
// - variant="repository"（阅览仓库）：地区徽标；启用态徽标常显（启用绿/停用灰）；
//   「最近抓取：」前缀；编辑/删除并入地址行（w-5 h-5）常显。
export function PriceSourceCard({
  src,
  variant,
  fetching = false,
  onFetch,
  onEdit,
  onDelete,
}: {
  src: PriceSource;
  variant: "panel" | "repository";
  fetching?: boolean;
  onFetch?: (src: PriceSource) => void;
  onEdit: (src: PriceSource) => void;
  onDelete: (src: PriceSource) => void;
}) {
  const toast = useToast();
  const panel = variant === "panel";

  const copyUrl = useCallback(
    async (url: string) => {
      try {
        await navigator.clipboard.writeText(url);
        toast.show("已复制抓取地址", "info");
      } catch {
        toast.show("复制失败：剪贴板不可用", "warn");
      }
    },
    [toast],
  );

  return (
    <div className="p-2 rounded-lg border border-border-soft/70 bg-bg-soft/30">
      <div className="flex items-center gap-1.5">
        {panel && <Coins size={12} className="text-sky-400 shrink-0" />}
        <span className="truncate text-fg text-[12px] font-medium">{src.name}</span>
        <span className="px-1.5 py-px rounded bg-bg-elev text-fg-faint text-[9.5px] shrink-0">{freqLabel(src.frequencyHours)}</span>
        {panel ? (
          !src.enabled && (
            <span className="px-1.5 py-px rounded bg-bg-elev text-fg-faint text-[9.5px] shrink-0">停用</span>
          )
        ) : (
          <>
            {src.area && (
              <span className="px-1.5 py-px rounded bg-bg-elev text-fg-faint text-[9.5px] shrink-0">
                {src.area}
              </span>
            )}
            <span className={`px-1.5 py-px rounded text-[9.5px] shrink-0 ${src.enabled ? "bg-ok/15 text-ok" : "bg-bg-elev text-fg-faint"}`}>
              {src.enabled ? "启用" : "停用"}
            </span>
          </>
        )}
        <span className="ml-auto shrink-0 text-fg-faint text-[10px]">
          {panel ? timeText(src.lastFetchAt) : `最近抓取：${timeText(src.lastFetchAt)}`}
        </span>
        {panel && onFetch && (
          <button
            className="shrink-0 px-2 h-6 rounded-md bg-sky-400/15 text-sky-300 text-[11px] cursor-pointer hover:bg-sky-400/25 transition-colors disabled:opacity-50"
            disabled={fetching}
            onClick={() => onFetch(src)}
            title="立即抓取该价格源"
          >
            {fetching ? "抓取中…" : "抓取"}
          </button>
        )}
        {panel && (
          <>
            <button
              className="shrink-0 w-6 h-6 inline-flex items-center justify-center rounded text-fg-faint hover:text-fg hover:bg-bg-elev"
              onClick={() => onEdit(src)}
              title="编辑价格源"
            >
              <Pencil size={11} />
            </button>
            <button
              className="shrink-0 w-6 h-6 inline-flex items-center justify-center rounded text-fg-faint hover:text-red-400 hover:bg-bg-elev"
              onClick={() => onDelete(src)}
              title="删除价格源"
            >
              <Trash2 size={11} />
            </button>
          </>
        )}
      </div>
      <div className="mt-1 flex items-start gap-1">
        <span
          className="min-w-0 flex-1 break-all text-fg-faint text-[9.5px] font-mono leading-snug"
          title={src.url}
        >
          抓取地址：{src.url}
        </span>
        <span className={`shrink-0 flex items-center gap-0.5 ${panel ? "opacity-0 group-hover:opacity-100 transition-opacity" : ""}`}>
          {!panel && (
            <>
              <button
                type="button"
                className="flex items-center justify-center w-5 h-5 rounded border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft"
                onClick={() => onEdit(src)}
                title="编辑价格源"
              >
                <Pencil size={10} />
              </button>
              <button
                type="button"
                className="flex items-center justify-center w-5 h-5 rounded border-0 bg-transparent text-fg-faint cursor-pointer hover:text-red-400 hover:bg-bg-soft"
                onClick={() => onDelete(src)}
                title="删除价格源"
              >
                <Trash2 size={10} />
              </button>
            </>
          )}
          <button
            type="button"
            className="flex items-center justify-center w-5 h-5 rounded border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft"
            onClick={() => void copyUrl(src.url)}
            title="复制抓取地址"
          >
            <Copy size={10} />
          </button>
          <button
            type="button"
            className="flex items-center justify-center w-5 h-5 rounded border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft"
            onClick={() => { const d = classifyExternalLink(src.url); if (d.kind === "open") openExternal(d.url); }}
            title="在浏览器打开抓取地址"
          >
            <ExternalLink size={10} />
          </button>
        </span>
      </div>
    </div>
  );
}
