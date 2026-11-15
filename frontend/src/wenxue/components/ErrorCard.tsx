import { X } from "../icons";
import type { Item } from "../lib/store";
import { useT } from "../lib/i18n";

/**
 * ErrorCard — a dismissible error display for turn_done failures.
 * Renders a red-bordered card with the error message and a close button.
 */
export function ErrorCard({
  item,
  onDismiss,
}: {
  item: Extract<Item, { kind: "notice" }>;
  onDismiss: (id: string) => void;
}) {
  const t = useT();
  return (
    <div className="mx-2 my-1.5 p-2 rounded-lg border border-err/25 border-l-[3px] border-l-err flex gap-2 items-start bg-err/[0.04]">
      <span className="flex-1 text-xs text-err leading-snug break-words">{item.text}</span>
      <button
        type="button"
        className="shrink-0 bg-transparent border-0 text-fg-faint cursor-pointer p-0.5 rounded hover:text-err hover:bg-bg-soft transition-colors"
        onClick={() => onDismiss(item.id)}
        aria-label={t("msg.dismissError")}
      >
        <X size={14} />
      </button>
    </div>
  );
}
