import { useEffect, useState, type RefObject } from "react";
import type { MarkdownHeading } from "../lib/markdownOutline";

export function MarkdownToc({
  headings,
  scrollRef,
  contentKey,
}: {
  headings: MarkdownHeading[];
  scrollRef: RefObject<HTMLElement | null>;
  contentKey: string;
}) {
  const [open, setOpen] = useState(true);
  const [active, setActive] = useState(headings[0]?.id ?? "");

  useEffect(() => {
    const root = scrollRef.current;
    if (!root) return;
    const nodes = Array.from(root.querySelectorAll<HTMLElement>(".md h1, .md h2, .md h3, .md h4, .md h5, .md h6"));
    for (let i = 0; i < headings.length && i < nodes.length; i++) nodes[i].id = headings[i].id;
    const update = () => {
      const top = root.getBoundingClientRect?.().top ?? 0;
      let next = headings[0]?.id ?? "";
      for (let i = 0; i < nodes.length; i++) {
        if ((nodes[i].getBoundingClientRect?.().top ?? Infinity) <= top + 48) next = headings[i]?.id ?? next;
        else break;
      }
      setActive(next);
    };
    update();
    root.addEventListener("scroll", update, { passive: true });
    return () => root.removeEventListener("scroll", update);
  }, [contentKey, headings, scrollRef]);

  if (headings.length < 2) return null;
  return (
    <div
      data-testid="markdown-toc"
      className="sticky top-2 z-20 float-right ml-2 max-h-[65vh] w-[180px] overflow-hidden rounded-lg border border-border-soft bg-bg-elev/95 shadow-sm backdrop-blur"
    >
      <button
        type="button"
        className="flex w-full cursor-pointer items-center border-0 bg-transparent px-2.5 py-1.5 text-[10.5px] font-medium text-fg-dim hover:bg-bg-soft"
        aria-expanded={open}
        onClick={() => setOpen((v) => !v)}
      >
        <span>目录 · {headings.length}</span>
        <span className="ml-auto text-fg-faint">{open ? "−" : "+"}</span>
      </button>
      {open && (
        <nav className="max-h-[calc(65vh-30px)] overflow-auto pb-1" aria-label="Markdown 目录">
          {headings.map((heading) => (
            <button
              key={heading.id}
              type="button"
              data-active={active === heading.id ? "true" : undefined}
              className={`block w-full cursor-pointer truncate border-0 bg-transparent px-2.5 py-1 text-left text-[10.5px] hover:bg-bg-soft ${
                active === heading.id ? "text-accent" : "text-fg-faint"
              }`}
              style={{ paddingLeft: `${10 + (heading.level - 1) * 8}px` }}
              title={heading.text}
              onClick={() => {
                const node = scrollRef.current?.querySelector<HTMLElement>(`#${heading.id}`);
                node?.scrollIntoView?.({ block: "start" });
                setActive(heading.id);
              }}
            >
              {heading.text}
            </button>
          ))}
        </nav>
      )}
    </div>
  );
}
