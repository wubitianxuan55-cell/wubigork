// Find.tsx — 阅读窗的节内查找条与命中定位（2026-11-28 索引优化批）。
//
// 检索命中跳进来时滚到命中那一段并高亮；阅读窗里也能直接查词、上一处 / 下一处翻。
// 高亮走 CSS Custom Highlight API：只登记 Range，不动 React 渲染出来的 DOM——往正文
// 里插 <mark> 的话，React 下次更新这段文字会和我们插的标记打架。不支持该 API 的环境
// （测试用的 jsdom）退成只计数 + 滚动。定位函数见 Find.logic.ts。
import { useCallback, useEffect, useRef, useState, type KeyboardEvent, type RefObject } from "react";
import { ArrowDown, ArrowUp, Search, X } from "../../gaea/icons";
import { findTerm, locateSnippet, textNodes, type FindHit } from "./Find.logic";

const ALL = "proposal-find";
const ACTIVE = "proposal-find-active";

/** 定位请求：needle 取自命中摘要的开头（定位到具体那一段），fallback 是全文命中词。 */
export interface LocateRequest {
  needle: string;
  fallback: string;
}

function overlaps(a: FindHit, b: FindHit): boolean {
  return a.pieces.some((p) => b.pieces.some((q) => p.node === q.node && p.start < q.end && q.start < p.end));
}

function firstOffset(hit: FindHit, nodes: Text[]): number {
  const i = nodes.indexOf(hit.pieces[0]?.node);
  if (i < 0) return Number.MAX_SAFE_INTEGER;
  let pos = 0;
  for (let k = 0; k < i; k++) pos += nodes[k].data.length;
  return pos + hit.pieces[0].start;
}

// TS 的 lib.dom 里 HighlightRegistry 只声明了 forEach（setlike 成员按 lib 组合生成），
// 这里按用到的最小面断言。
interface Highlights {
  set(name: string, value: Highlight): unknown;
  delete(name: string): boolean;
}

function registry(): Highlights | null {
  if (typeof Highlight === "undefined" || typeof CSS === "undefined" || !("highlights" in CSS)) return null;
  return (CSS as unknown as { highlights: Highlights }).highlights;
}

function rangesOf(hit: FindHit): Range[] {
  return hit.pieces.map((p) => {
    const r = document.createRange();
    r.setStart(p.node, p.start);
    r.setEnd(p.node, p.end);
    return r;
  });
}

function paint(hits: FindHit[], active: number) {
  const reg = registry();
  if (!reg) return;
  reg.set(ALL, new Highlight(...hits.flatMap(rangesOf)));
  reg.set(ACTIVE, hits[active] ? new Highlight(...rangesOf(hits[active])) : new Highlight());
}

function clearHighlights() {
  const reg = registry();
  if (!reg) return;
  reg.delete(ALL);
  reg.delete(ACTIVE);
}

function scrollTo(hit: FindHit | undefined) {
  hit?.pieces[0]?.node.parentElement?.scrollIntoView?.({ block: "center" });
}

/**
 * 节内查找条。version 随正文变化（换节、重读、保存后重取），变了就清高亮并按
 * locate（检索命中跳进来的定位请求）或当前查找词重新标一遍。locate 处理完经
 * onLocated 交还页面清掉。
 *
 * 重标走被动副作用而不是同步副作用：正文容器是查找条后面的兄弟节点，同步副作用
 * 跑的时候它的 ref 还没挂上；被动副作用在整次提交之后才跑。
 */
export default function ReaderFind({ bodyRef, version, locate, onLocated, open, onOpenChange }: {
  bodyRef: RefObject<HTMLElement | null>;
  version: string;
  locate: LocateRequest | null;
  onLocated: () => void;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [query, setQuery] = useState("");
  const [count, setCount] = useState(0);
  const [active, setActive] = useState(0);
  const hitsRef = useRef<FindHit[]>([]);
  const activeRef = useRef(0);
  const locateRef = useRef(locate);
  locateRef.current = locate;
  const queryRef = useRef("");
  const inputRef = useRef<HTMLInputElement>(null);
  // 页面的回调走 ref 读：副作用只随正文版本重跑，不随页面每次渲染重跑。
  const openChangeRef = useRef(onOpenChange);
  openChangeRef.current = onOpenChange;
  const locatedRef = useRef(onLocated);
  locatedRef.current = onLocated;

  const show = useCallback((hits: FindHit[], index: number, scroll: boolean) => {
    hitsRef.current = hits;
    activeRef.current = index;
    setCount(hits.length);
    setActive(index);
    paint(hits, index);
    if (scroll) scrollTo(hits[index]);
  }, []);

  useEffect(() => {
    const root = bodyRef.current;
    if (!root) return;
    const req = locateRef.current;
    if (req) {
      // 先定位到命中那一段；有全文命中词时把它的全部出现处一并标出，落在那一段上。
      const needle = req.needle ? locateSnippet(root, req.needle) : null;
      let hits: FindHit[] = [];
      let at = 0;
      if (req.fallback) {
        hits = findTerm(root, req.fallback);
        if (needle) {
          const i = hits.findIndex((h) => overlaps(h, needle));
          if (i >= 0) {
            at = i;
          } else {
            const nodes = textNodes(root);
            hits = [...hits, needle].sort((a, b) => firstOffset(a, nodes) - firstOffset(b, nodes));
            at = hits.indexOf(needle);
          }
        }
      } else if (needle) {
        hits = [needle];
      }
      queryRef.current = req.fallback || req.needle.slice(0, 12);
      setQuery(queryRef.current);
      openChangeRef.current(true);
      show(hits, at, hits.length > 0);
      locatedRef.current();
    } else if (queryRef.current.trim()) {
      const hits = findTerm(root, queryRef.current);
      show(hits, 0, hits.length > 0);
    } else {
      show([], 0, false);
    }
    return clearHighlights;
  }, [version, bodyRef, show]);

  useEffect(() => {
    if (open) inputRef.current?.focus();
  }, [open]);

  const close = () => {
    queryRef.current = "";
    setQuery("");
    onOpenChange(false);
    show([], 0, false);
  };

  const step = (delta: 1 | -1) => {
    if (!hitsRef.current.length) return;
    show(hitsRef.current, (activeRef.current + delta + hitsRef.current.length) % hitsRef.current.length, true);
  };

  const onKey = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      e.preventDefault();
      step(e.shiftKey ? -1 : 1);
    } else if (e.key === "Escape") {
      e.preventDefault();
      e.stopPropagation();
      close();
    }
  };

  if (!open) return null;
  return (
    <div className="pl-find" role="search" aria-label="节内查找" data-testid="proposal-find">
      <Search size={13} aria-hidden />
      <input
        ref={inputRef}
        aria-label="在本节中查找"
        placeholder="在本节中查找"
        value={query}
        onChange={(e) => {
          const value = e.target.value;
          queryRef.current = value;
          setQuery(value);
          const term = value.trim();
          const hits = term && bodyRef.current ? findTerm(bodyRef.current, term) : [];
          show(hits, 0, hits.length > 0);
        }}
        onKeyDown={onKey}
      />
      <span className="pl-find-count" aria-live="polite">
        {count ? `第 ${active + 1} 处，共 ${count} 处` : query.trim() ? "本节没有" : ""}
      </span>
      <button type="button" className="pl-icon-btn" aria-label="上一处" disabled={!count} onClick={() => step(-1)}><ArrowUp size={13} /></button>
      <button type="button" className="pl-icon-btn" aria-label="下一处" disabled={!count} onClick={() => step(1)}><ArrowDown size={13} /></button>
      <button type="button" className="pl-icon-btn" aria-label="关闭查找" onClick={close}><X size={13} /></button>
    </div>
  );
}
