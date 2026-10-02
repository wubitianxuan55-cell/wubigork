import { useState } from "react";
import { Modal } from "antd";
import { CloudUpload, RefreshCw } from "../../icons";
import { app } from "../../lib/bridge";
import type { PriceSource } from "../../lib/types";
import { useToast } from "../Toast";
import { PriceSourceCard } from "./PriceSourceCard";
import { PriceSourceFormModal } from "./PriceSourceFormModal";
import { usePriceSources } from "./usePriceSources";

// PriceSourcesRepository 价格源阅览仓库：只读陈列系统里所有已添加的价格源
// 及其抓取地址，支持复制地址 / 浏览器打开；管理（增删改/抓取）仍在价格源页。
// 数据装载与价格源页共用 usePriceSources（FE2-06）；v4.362：加载失败不再静默
// 伪装成空仓库（原 Promise.race 失败吞成空列表），改出「读取失败 + 重试」三态。
export function PriceSourcesRepository() {
  const { sources, loading, loadFailed, load } = usePriceSources();
  const [editOpen, setEditOpen] = useState(false);
  const [editing, setEditing] = useState<PriceSource | null>(null);
  const [deleting, setDeleting] = useState<PriceSource | null>(null);
  const toast = useToast();

  const enabledCount = sources.filter((s) => s.enabled).length;

  const removeSource = async () => {
    if (!deleting) return;
    // v4.362：删除失败不再吞成假成功——原吞错后无条件报「已删除」，条目刷新复活。
    const ok = await app.PriceSourceDelete(deleting.id).then(() => true).catch(() => false);
    if (!ok) {
      toast.show(`删除价格源「${deleting.name}」失败，请重试`, "error");
      return;
    }
    toast.show(`已删除价格源「${deleting.name}」`, "info");
    setDeleting(null);
    load();
  };

  return (
    <div className="flex flex-col h-full text-fg-dim text-xs">
      <div className="flex items-center justify-between px-3 py-2 border-b border-border-soft">
        <span className="flex items-center gap-1.5 font-semibold text-fg text-sm">
          <CloudUpload size={13} className="text-sky-400" />
          价格源阅览仓库
        </span>
        <div className="flex items-center gap-2">
          {!loading && sources.length > 0 && (
            <span className="text-[10px] text-fg-faint">
              共 {sources.length} 个（启用 {enabledCount}）
            </span>
          )}
          <button
            className="flex items-center justify-center w-6 h-6 border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft rounded"
            onClick={load}
            title="刷新仓库"
            disabled={loading}
          >
            <RefreshCw size={12} className={loading ? "animate-spin" : ""} />
          </button>
        </div>
      </div>

      <div className="shrink-0 px-3 pt-2 text-[10.5px] text-fg-faint">
        系统内全部已添加的价格源地址一览（管理/抓取在「价格源」页）
      </div>

      <div className="flex-1 min-h-0 overflow-y-auto p-2">
        {loading ? (
          <div className="py-8 text-center text-fg-faint text-[11px]">加载中…</div>
        ) : loadFailed ? (
          <div className="py-8 text-center text-[11px]" role="status" data-testid="price-sources-load-failed">
            <span className="text-fg-dim">价格源列表读取失败</span>
            <button type="button" className="ml-2 px-2 h-6 rounded-full bg-accent text-accent-fg text-[10.5px] cursor-pointer" onClick={() => load()}>
              重试
            </button>
          </div>
        ) : sources.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full gap-2 px-6 text-center text-fg-faint/50">
            <CloudUpload size={24} className="opacity-40" />
            <span className="text-[11px] leading-relaxed">
              仓库为空
              <br />
              去「价格源」页添加造价信息网地址后，会在这里集中展示
            </span>
          </div>
        ) : (
          <div className="flex flex-col gap-1.5">
            {sources.map((src) => (
              <PriceSourceCard
                key={src.id}
                src={src}
                variant="repository"
                onEdit={(s) => {
                  setEditing(s);
                  setEditOpen(true);
                }}
                onDelete={setDeleting}
              />
            ))}
          </div>
        )}
      </div>

      {/* 编辑价格源（与价格源页共用弹窗） */}
      <PriceSourceFormModal
        open={editOpen}
        editing={editing}
        onClose={() => setEditOpen(false)}
        onSaved={() => {
          setEditOpen(false);
          load();
        }}
      />

      {/* 删除确认 */}
      <Modal
        title="删除价格源"
        open={!!deleting}
        destroyOnHidden
        transitionName=""
        maskTransitionName=""
        onCancel={() => setDeleting(null)}
        onOk={() => void removeSource()}
        okText="删除"
        okButtonProps={{ danger: true }}
        cancelText="取消"
      >
        <p className="text-[13px] text-fg-dim">确定删除价格源「{deleting?.name}」吗？（抓取记录与价格历史保留）</p>
      </Modal>
    </div>
  );
}
