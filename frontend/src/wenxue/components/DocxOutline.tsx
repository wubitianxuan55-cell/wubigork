import { AlertCircle, ListTree, Pencil, X } from "../icons";
import type { DocxOutlineItem } from "../lib/docxOutline";

/**
 * DocxOutline 是 docx 预览右侧的「目录」侧栏（对标 Word/WPS 导航窗格的
 * 文档结构图）：按标题层级缩进列出全文章节，点条目滚动定位到版式中对应
 * 段落（锚点由 DocxPreview 在渲染完成后链接）；条目右侧的「定位后编辑」
 * 入口把「请修改 <文件名> 中『<节名>』一节：」模板插入输入框（与
 * PptxOutline「针对第 N 页修改」同一 composer 插入通道，不直接发送）。
 *
 * 文档没有可解析标题 / 解析失败 → 诚实提示原因，绝不假装有目录；版式预览
 * 本身不受影响。
 */
export function DocxOutline({
  items,
  error,
  onNavigate,
  onInsertModify,
  onClose,
}: {
  items: DocxOutlineItem[];
  /** 目录解析失败信息（非空时展示诚实降级） */
  error?: string;
  onNavigate: (index: number) => void;
  onInsertModify: (item: DocxOutlineItem) => void;
  onClose: () => void;
}) {
  return (
    <aside
      data-testid="docx-outline"
      className="w-60 shrink-0 border-l border-border-soft flex flex-col min-h-0 bg-bg-elev-2/60"
    >
      <div className="flex items-center gap-1.5 px-3 py-2 border-b border-border-soft text-fg-dim text-[11px] shrink-0">
        <ListTree size={12} className="text-accent shrink-0" />
        <span className="font-medium">目录</span>
        {!error && <span className="text-fg-faint ml-auto">{items.length} 节</span>}
        <button
          type="button"
          className="flex items-center justify-center w-5 h-5 border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg rounded"
          onClick={onClose}
          title="关闭目录"
          aria-label="关闭目录"
        >
          <X size={11} />
        </button>
      </div>
      {error ? (
        <div className="flex items-start gap-1.5 px-3 py-2.5 text-fg-faint text-[11px] leading-relaxed">
          <AlertCircle size={12} className="text-amber-500/70 shrink-0 mt-px" />
          <span>目录不可用（{error}），仍可全文预览。</span>
        </div>
      ) : items.length === 0 ? (
        <div className="px-3 py-2.5 text-fg-faint text-[11px] leading-relaxed">
          未检测到标题结构：本文档标题未使用 Word 标题样式或大纲级别，暂无法打开目录。
        </div>
      ) : (
        <ul className="flex-1 min-h-0 overflow-auto px-1.5 py-1.5 flex flex-col gap-px">
          {items.map((item, i) => (
            <li
              key={`${i}-${item.level}-${item.text}`}
              className="rounded-md hover:bg-bg-soft/70 group"
              style={{ paddingLeft: Math.min(item.level - 1, 8) * 9 }}
            >
              <div className="flex items-center gap-1 min-w-0 py-1 pr-0.5">
                <button
                  type="button"
                  className="flex items-baseline gap-1.5 min-w-0 flex-1 text-left bg-transparent border-0 p-0 cursor-pointer"
                  data-testid={`docx-outline-item-${i}`}
                  onClick={() => onNavigate(i)}
                  title="点击定位到该节"
                >
                  <span className="text-[9.5px] text-accent/70 font-mono shrink-0">
                    {item.level}
                  </span>
                  <span className="text-[11.5px] text-fg truncate">{item.text}</span>
                </button>
                <button
                  type="button"
                  className="inline-flex items-center justify-center w-5 h-5 rounded border border-transparent bg-transparent text-fg-faint cursor-pointer opacity-0 group-hover:opacity-100 hover:text-accent hover:border-accent/30 hover:bg-accent/10 transition-opacity shrink-0"
                  data-testid={`docx-outline-edit-${i}`}
                  onClick={() => onInsertModify(item)}
                  title={`插入「修改${item.text}」指令模板`}
                  aria-label={`插入修改${item.text}的指令模板`}
                >
                  <Pencil size={9} />
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </aside>
  );
}
