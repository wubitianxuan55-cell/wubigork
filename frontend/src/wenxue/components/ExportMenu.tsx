import { useEffect, useState } from "react";
import { useT } from "../lib/i18n";
import { ChevronDown, Download, FileText, FileWord, FilePdf } from "../icons";

// 导出收拢菜单（v4.29「化繁为简」刀，对标 Devin/Linear「新动作只进菜单不加常驻按钮」
// 与 VS Code「顶栏单点溢出」）：原顶栏「导出 / Word / PDF」三个常驻文字钮收进
// 一个「导出 ⌄」下拉——三个出口（Markdown / Word / PDF）与底层管线原样保留，
// 只改呈现，功能零删除（用户红线：简化界面 ≠ 删除功能）。

export type ExportFormat = "md" | "docx" | "pdf";

export function ExportMenu({
  disabled,
  onPick,
}: {
  /** 会话无内容时整钮禁用（与原三个导出钮的 disabled 语义一致）。 */
  disabled?: boolean;
  /** 选择出口；md=下载 Markdown，docx/pdf=统一交付管线。 */
  onPick: (format: ExportFormat) => void;
}) {
  const t = useT();
  const [open, setOpen] = useState(false);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [open]);

  const pick = (format: ExportFormat) => {
    setOpen(false);
    onPick(format);
  };

  const items: { format: ExportFormat; label: string; hint?: string; Icon: typeof FileText }[] = [
    { format: "md", label: t("topbar.exportMarkdown"), Icon: FileText },
    { format: "docx", label: t("topbar.exportWordShort"), hint: t("topbar.exportWord"), Icon: FileWord },
    { format: "pdf", label: t("topbar.exportPdfShort"), hint: t("topbar.exportPdf"), Icon: FilePdf },
  ];

  return (
    <span className="relative inline-flex">
      <button
        type="button"
        className="toolbar-btn no-drag"
        disabled={disabled}
        aria-haspopup="menu"
        aria-expanded={open}
        title={t("topbar.export")}
        data-testid="export-menu-trigger"
        onClick={() => setOpen((o) => !o)}
      >
        <Download size={13} />
        <span>{t("topbar.export")}</span>
        <ChevronDown size={11} />
      </button>
      {open && (
        <>
          {/* 透明遮罩：点击菜单外部即关闭（同 WorkspaceTabs 设置弹层交互） */}
          <span className="fixed inset-0 z-10 cursor-default" aria-hidden onClick={() => setOpen(false)} />
          <span
            role="menu"
            aria-label={t("topbar.export")}
            data-testid="export-menu"
            className="absolute right-0 top-[calc(100%+2px)] z-20 flex min-w-40 flex-col rounded-lg border border-border-soft bg-bg-elev p-1"
            style={{ boxShadow: "var(--ds-shadow-dropdown)" }}
          >
            {items.map(({ format, label, hint, Icon }) => (
              <button
                key={format}
                type="button"
                role="menuitem"
                title={hint}
                className="flex items-center gap-2 rounded-md border-0 bg-transparent px-2 py-1.5 text-left text-[12px] text-fg cursor-pointer hover:bg-(color:--md-sys-color-surface-container-high)"
                onClick={() => pick(format)}
              >
                <Icon size={13} aria-hidden style={{ color: "var(--gaea-glow)" }} />
                {label}
              </button>
            ))}
          </span>
        </>
      )}
    </span>
  );
}
