import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { BookOpen, ChevronRight, RefreshCw } from "../icons";
import { app } from "../lib/bridge";
import { Markdown } from "./Markdown";

// WenxueReadingHub — 图文板块「阅读」标签页（2026-11-15 阅读刀）。
// 数据面=小说工作空间定稿目录（定稿|final|finalized，只放发布稿不放草稿），
// 与 agent 的 wenxue_final_list/read/write 专属工具共享同一数据源：agent
// 入库（wenxue_final_write）→ 点刷新即见；目录导航 + 阅读窗（Markdown 渲染，
// 本地相对图片按章节路径解析）。与「角色」tab 同款移植纪律：形态新写、
// 数据层锚工作空间文件。

interface ReadingChapter {
  path: string;
  name: string;
  number: number;
  title: string;
  words: number;
}

const DIR_CANDIDATES = ["定稿", "final", "finalized"];

function countWords(s: string): number {
  let n = 0;
  for (const ch of s) if (!/\s/.test(ch)) n++;
  return n;
}

function parseChapterName(name: string): { number: number; title: string } {
  const stem = name.replace(/\.(md|markdown|txt)$/i, "");
  const m = stem.match(/^(\d+)[-_\s.：:]*(.*)$/);
  if (m) return { number: parseInt(m[1], 10), title: m[2].trim() || stem };
  return { number: 0, title: stem };
}

export function WenxueReadingHub() {
  const [dirRel, setDirRel] = useState("");
  const [chapters, setChapters] = useState<ReadingChapter[]>([]);
  const [selected, setSelected] = useState<string>("");
  const [body, setBody] = useState("");
  const [loading, setLoading] = useState(true);
  const [reading, setReading] = useState(false);
  const loadToken = useRef(0);
  const readToken = useRef(0);

  const load = useCallback(async () => {
    const token = ++loadToken.current;
    setLoading(true);
    try {
      const rootEntries = await app.ListDir("");
      let dir = "";
      for (const cand of DIR_CANDIDATES) {
        if (rootEntries.some((e) => e.isDir && e.name === cand)) {
          dir = cand;
          break;
        }
      }
      setDirRel(dir);
      if (!dir) {
        setChapters([]);
        setSelected("");
        setBody("");
        return;
      }
      const entries = await app.ListDir(dir);
      const rows: ReadingChapter[] = [];
      for (const e of entries) {
        if (e.isDir || !/\.(md|markdown|txt)$/i.test(e.name)) continue;
        const rel = `${dir}/${e.name}`;
        const tf = await app.GaeaReadTextFile(rel);
        const { number, title } = parseChapterName(e.name);
        rows.push({ path: rel, name: e.name, number, title, words: countWords(tf.text) });
      }
      rows.sort((a, b) => (a.number !== b.number ? a.number - b.number : a.name.localeCompare(b.name, "zh-Hans-CN")));
      if (token !== loadToken.current) return;
      setChapters(rows);
      setSelected((cur) => (rows.some((r) => r.path === cur) ? cur : rows[0]?.path ?? ""));
    } catch {
      if (token === loadToken.current) { setChapters([]); setSelected(""); }
    } finally {
      if (token === loadToken.current) setLoading(false);
    }
  }, []);

  // 选中章节正文加载
  useEffect(() => {
    if (!selected) { setBody(""); return; }
    const token = ++readToken.current;
    setReading(true);
    app.GaeaReadTextFile(selected)
      .then((tf) => { if (token === readToken.current) setBody(tf.text); })
      .catch(() => { if (token === readToken.current) setBody("（读取失败）"); })
      .finally(() => { if (token === readToken.current) setReading(false); });
  }, [selected]);

  useEffect(() => { void load(); }, [load]);
  // 切书刷新（与文件树/角色 tab 同款事件）
  useEffect(() => {
    const handler = () => { void load(); };
    window.addEventListener("wenxue:workspace-changed", handler);
    return () => window.removeEventListener("wenxue:workspace-changed", handler);
  }, [load]);

  const idx = chapters.findIndex((c) => c.path === selected);
  const current = idx >= 0 ? chapters[idx] : undefined;
  const totalWords = useMemo(() => chapters.reduce((s, c) => s + c.words, 0), [chapters]);

  const jump = (delta: number) => {
    const next = chapters[idx + delta];
    if (next) setSelected(next.path);
  };

  return (
    <div className="h-full flex min-h-0" data-testid="wenxue-reading-hub">
      {/* 目录导航 */}
      <div
        className="w-60 shrink-0 flex flex-col border-r overflow-hidden"
        style={{ borderColor: "var(--md-sys-color-outline-variant)" }}
      >
        <div className="px-3 py-2 border-b flex items-center justify-between gap-1" style={{ borderColor: "var(--md-sys-color-outline-variant)" }}>
          <span className="text-[11px] font-semibold text-fg-dim flex items-center gap-1.5">
            <BookOpen size={12} aria-hidden className="text-accent" /> 定稿目录
            <span className="text-fg-faint font-normal">（{chapters.length} 章 · {totalWords} 字）</span>
          </span>
          <button
            type="button"
            className="flex items-center gap-1 px-1.5 py-0.5 border-0 rounded bg-transparent text-fg-dim text-[10px] cursor-pointer hover:bg-bg-soft"
            onClick={() => void load()}
            title="刷新（agent wenxue_final_write 入库后同步）"
          >
            <RefreshCw size={10} />
          </button>
        </div>
        <div className="flex-1 overflow-auto py-1">
          {loading ? (
            <div className="py-8 text-center text-fg-faint text-[12px]">读取定稿目录中…</div>
          ) : chapters.length === 0 ? (
            <div className="px-3 py-8 text-center text-fg-faint text-[11px] leading-relaxed">
              定稿目录为空。
              <br />
              让 agent 用 wenxue_final_write 入库发布稿，
              <br />
              或从「章节」目录把定稿文本放入「定稿」文件夹。
            </div>
          ) : (
            chapters.map((c) => {
              const active = c.path === selected;
              return (
                <button
                  key={c.path}
                  type="button"
                  role="button"
                  data-testid={`wenxue-reading-item-${c.number || c.name}`}
                  aria-selected={active}
                  className={`w-full text-left px-3 py-1.5 border-0 bg-transparent cursor-pointer text-[12px] transition-colors ${active ? "text-accent bg-bg-soft" : "text-fg-dim hover:text-fg hover:bg-bg-soft"}`}
                  onClick={() => setSelected(c.path)}
                >
                  <span className="block truncate">
                    {c.number > 0 ? `${c.number}. ` : ""}{c.title}
                  </span>
                  <span className="block text-[10px] text-fg-faint">{c.words} 字</span>
                </button>
              );
            })
          )}
        </div>
      </div>

      {/* 阅读窗 */}
      <div className="flex-1 min-w-0 flex flex-col overflow-hidden">
        {current ? (
          <>
            <div
              className="flex items-center justify-between gap-2 px-5 py-2 border-b"
              style={{ borderColor: "var(--md-sys-color-outline-variant)" }}
            >
              <button
                type="button"
                className="flex items-center gap-1 px-2 py-1 rounded border-0 bg-transparent text-fg-dim text-[11px] cursor-pointer hover:bg-bg-soft disabled:opacity-40"
                onClick={() => jump(-1)}
                disabled={idx <= 0}
              >
                <span aria-hidden style={{ transform: "rotate(180deg)", display: "inline-flex" }}><ChevronRight size={12} /></span> 上一章
              </button>
              <span className="text-[12px] font-medium text-fg truncate">
                {current.number > 0 ? `第${current.number}章 ` : ""}{current.title}
                <span className="ml-2 text-[10px] text-fg-faint">{current.words} 字</span>
              </span>
              <button
                type="button"
                className="flex items-center gap-1 px-2 py-1 rounded border-0 bg-transparent text-fg-dim text-[11px] cursor-pointer hover:bg-bg-soft disabled:opacity-40"
                onClick={() => jump(1)}
                disabled={idx < 0 || idx >= chapters.length - 1}
              >
                下一章 <ChevronRight size={12} />
              </button>
            </div>
            <div className="flex-1 overflow-auto px-8 py-6">
              <div className="max-w-[760px] mx-auto reading-prose">
                {reading ? (
                  <div className="py-10 text-center text-fg-faint text-[12px]">加载中…</div>
                ) : (
                  <Markdown text={body} imageBasePath={current.path} />
                )}
              </div>
            </div>
          </>
        ) : (
          <div className="flex-1 flex flex-col items-center justify-center text-fg-faint text-[12px] gap-2">
            <BookOpen size={28} aria-hidden className="opacity-40" />
            {loading ? "加载中…" : dirRel ? "定稿目录暂无章节——从「章节」目录把发布稿放入「定稿」文件夹，或让 agent 用 wenxue_final_write 入库。" : "当前小说尚无定稿目录。"}
          </div>
        )}
      </div>
    </div>
  );
}
