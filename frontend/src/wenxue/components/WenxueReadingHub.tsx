import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { CSSProperties, UIEvent } from "react";
import {
  BookOpen, ChevronRight, Clock, FileText, ListTree, Maximize2,
  Minimize2, RefreshCw, Search,
} from "../icons";
import { app } from "../lib/bridge";
import { subscribe } from "../../events";
import { Markdown } from "./Markdown";
import { WenxueHubEmpty, WenxueHubFrame } from "./WenxueHubFrame";
import {
  countWords, markdownHeadings, readingMinutes, withoutLeadingH1,
} from "./wenxueHubUtils";

// WenxueReadingHub — 文学板块「阅读」标签页。
// 数据面仍是工作空间「定稿」目录，不新增持久化格式。UI 从原来的双栏文件
// 浏览器升级为阅读室：章节目录、书页正文、章内导航与阅读偏好三层分工。
// agent 通过 wenxue_final_write 入库后，gaea-wenxue-filetree 通道会触发重载。

interface ReadingChapter {
  path: string;
  name: string;
  number: number;
  title: string;
  words: number;
}

const DIR_CANDIDATES = ["定稿", "final", "finalized"];

// 阅读排版偏好（2026-11-23 用户拍板：字体可选/字号滑动改/首字不放大）。
// 字体全部走系统自带栈（Windows 优先），默认档沿用书宋 --wx-serif；
// 偏好存 localStorage（键名空间化，读写失败静默回默认）。
const READING_FONTS = [
  { key: "default", label: "书宋 · 默认", stack: "var(--wx-serif)" },
  { key: "song", label: "宋体", stack: '"SimSun", "Songti SC", "Noto Serif CJK SC", serif' },
  { key: "yahei", label: "雅黑", stack: '"Microsoft YaHei", "PingFang SC", "Noto Sans CJK SC", sans-serif' },
  { key: "kai", label: "楷体", stack: '"KaiTi", "Kaiti SC", "STKaiti", serif' },
  { key: "fangsong", label: "仿宋", stack: '"FangSong", "Fangsong SC", "STFangsong", serif' },
] as const;

type ReadingFontKey = (typeof READING_FONTS)[number]["key"];

const READING_SIZE_MIN = 14;
const READING_SIZE_MAX = 24;
const READING_PREFS_KEY = "gaea.wenxue.reading.prefs";

interface ReadingPrefs {
  fontSize: number;
  fontKey: ReadingFontKey;
}

function clampReadingSize(value: unknown): number {
  const n = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(n)) return 17;
  return Math.min(READING_SIZE_MAX, Math.max(READING_SIZE_MIN, Math.round(n)));
}

function loadReadingPrefs(): ReadingPrefs {
  const fallback: ReadingPrefs = { fontSize: 17, fontKey: "default" };
  try {
    const raw = localStorage.getItem(READING_PREFS_KEY);
    if (!raw) return fallback;
    const parsed = JSON.parse(raw) as Partial<ReadingPrefs>;
    const font = READING_FONTS.find((f) => f.key === parsed.fontKey);
    return {
      fontSize: clampReadingSize(parsed.fontSize),
      fontKey: font ? font.key : "default",
    };
  } catch {
    return fallback;
  }
}

function readingFontStack(key: ReadingFontKey): string {
  return READING_FONTS.find((f) => f.key === key)?.stack ?? READING_FONTS[0].stack;
}

function parseChapterName(name: string): { number: number; title: string } {
  const stem = name.replace(/\.(md|markdown|txt)$/i, "");
  const m = stem.match(/^(\d+)[-_\s.：:]*(.*)$/);
  if (m) return { number: parseInt(m[1], 10), title: m[2].trim() || stem };
  return { number: 0, title: stem };
}

function chapterLabel(chapter: ReadingChapter): string {
  if (chapter.number > 0) return `第${chapter.number}章`;
  return chapter.title.includes("序") ? "序" : "章";
}

export function WenxueReadingHub() {
  const [dirRel, setDirRel] = useState("");
  const [chapters, setChapters] = useState<ReadingChapter[]>([]);
  const [selected, setSelected] = useState<string>("");
  const [body, setBody] = useState("");
  const [loading, setLoading] = useState(true);
  const [reading, setReading] = useState(false);
  const [error, setError] = useState("");
  const [bookTitle, setBookTitle] = useState("当前作品");
  const [tocQuery, setTocQuery] = useState("");
  const [focusMode, setFocusMode] = useState(false);
  const [prefs, setPrefs] = useState<ReadingPrefs>(loadReadingPrefs);
  const [scrollPct, setScrollPct] = useState(0);
  const loadToken = useRef(0);
  const readToken = useRef(0);
  const scrollRef = useRef<HTMLDivElement>(null);
  const paperRef = useRef<HTMLElement>(null);

  const load = useCallback(async () => {
    const token = ++loadToken.current;
    setLoading(true);
    setError("");
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
      rows.sort((a, b) => (a.number !== b.number
        ? a.number - b.number
        : a.name.localeCompare(b.name, "zh-Hans-CN")));
      if (token !== loadToken.current) return;
      setChapters(rows);
      setSelected((cur) => (rows.some((r) => r.path === cur) ? cur : rows[0]?.path ?? ""));
    } catch (err) {
      if (token === loadToken.current) {
        setChapters([]);
        setSelected("");
        setError(err instanceof Error ? err.message : String(err));
      }
    } finally {
      if (token === loadToken.current) setLoading(false);
    }
  }, []);

  const readChapter = useCallback(async (path: string) => {
    const token = ++readToken.current;
    setReading(true);
    setError("");
    try {
      const tf = await app.GaeaReadTextFile(path);
      if (token === readToken.current) setBody(tf.text);
    } catch (err) {
      if (token === readToken.current) {
        setBody("");
        setError(err instanceof Error ? err.message : "章节读取失败");
      }
    } finally {
      if (token === readToken.current) setReading(false);
    }
  }, []);

  useEffect(() => {
    if (!selected) {
      setBody("");
      return;
    }
    void readChapter(selected);
    requestAnimationFrame(() => {
      if (scrollRef.current) scrollRef.current.scrollTop = 0;
      setScrollPct(0);
    });
  }, [readChapter, selected]);

  useEffect(() => { void load(); }, [load]);

  useEffect(() => {
    const handler = () => { void load(); };
    window.addEventListener("wenxue:workspace-changed", handler);
    return () => window.removeEventListener("wenxue:workspace-changed", handler);
  }, [load]);

  useEffect(() => {
    return subscribe("gaea-wenxue-filetree", (payload) => {
      const paths = (payload as { paths?: unknown } | null)?.paths;
      if (!Array.isArray(paths)) return;
      const hit = paths.some((p) =>
        typeof p === "string"
        && DIR_CANDIDATES.some((dir) => p.replaceAll("\\", "/").startsWith(`${dir}/`)),
      );
      if (hit) void load();
    });
  }, [load]);

  useEffect(() => {
    let live = true;
    const meta = (app as typeof app & { WenxueMeta?: () => Promise<{ cwd?: string }> }).WenxueMeta;
    if (typeof meta === "function") {
      void meta()
        .then((value) => {
          const name = value?.cwd?.split(/[\\/]/).filter(Boolean).pop();
          if (live && name) setBookTitle(name);
        })
        .catch(() => {});
    }
    return () => { live = false; };
  }, []);

  const idx = chapters.findIndex((c) => c.path === selected);
  const current = idx >= 0 ? chapters[idx] : undefined;
  const fontSize = prefs.fontSize;
  const fontKey = prefs.fontKey;
  const fontLabel = READING_FONTS.find((f) => f.key === fontKey)?.label ?? READING_FONTS[0].label;

  // 排版偏好持久化：字号/字体跨重启保留（读写失败静默——隐私模式等场景）。
  useEffect(() => {
    try {
      localStorage.setItem(READING_PREFS_KEY, JSON.stringify(prefs));
    } catch { /* 忽略持久化失败 */ }
  }, [prefs]);

  const totalWords = useMemo(() => chapters.reduce((sum, c) => sum + c.words, 0), [chapters]);
  const displayBody = useMemo(() => withoutLeadingH1(body), [body]);
  const headings = useMemo(
    () => markdownHeadings(displayBody).filter((h) => h.level <= 3),
    [displayBody],
  );
  const filteredChapters = useMemo(() => {
    const q = tocQuery.trim().toLowerCase();
    if (!q) return chapters;
    return chapters.filter((c) => `${c.number} ${c.title} ${c.name}`.toLowerCase().includes(q));
  }, [chapters, tocQuery]);
  const bookProgress = chapters.length === 0
    ? 0
    : Math.round(((Math.max(0, idx) + (current ? 1 : 0)) / chapters.length) * 100);

  const jump = (delta: number) => {
    const next = chapters[idx + delta];
    if (next) setSelected(next.path);
  };

  const handleScroll = (event: UIEvent<HTMLDivElement>) => {
    const el = event.currentTarget;
    const max = el.scrollHeight - el.clientHeight;
    const next = max > 0 ? Math.min(100, Math.max(0, Math.round((el.scrollTop / max) * 100))) : 0;
    setScrollPct((prev) => (Math.abs(prev - next) >= 1 ? next : prev));
  };

  const jumpToHeading = (headingIndex: number) => {
    const nodes = paperRef.current?.querySelectorAll<HTMLElement>(
      ".wx-reading-prose .md h1, .wx-reading-prose .md h2, .wx-reading-prose .md h3",
    );
    nodes?.[headingIndex]?.scrollIntoView({ behavior: "smooth", block: "start" });
  };

  // Ctrl+滚轮缩放字号（2026-11-23 用户反馈）：书页上滚放大/下滚缩小，与滑杆
  // 同源同一偏好。普通滚轮保持原生滚动不劫持；必须拦 Ctrl 态默认行为，否则
  // WebView2 会把它当页面缩放。监听挂原生非 passive（React onWheel 是被动
  // 监听，preventDefault 不生效）；书页容器随章节挂载，依赖章节路径重挂。
  const chapterPath = current?.path;
  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    const onWheel = (event: WheelEvent) => {
      if (!event.ctrlKey) return;
      event.preventDefault();
      const dir = event.deltaY < 0 ? 1 : -1;
      setPrefs((p) => ({ ...p, fontSize: clampReadingSize(p.fontSize + dir) }));
    };
    el.addEventListener("wheel", onWheel, { passive: false });
    return () => el.removeEventListener("wheel", onWheel);
  }, [chapterPath]);

  return (
    <WenxueHubFrame
      tone="reading"
      eyebrow="书稿阅览"
      title="沉浸阅读室"
      description="把定稿铺成一本书来读：先看全书进度与章节结构，再专注于正文。"
      metrics={[
        { label: "定稿", value: chapters.length, hint: chapters.length ? "章" : "待入库" },
        { label: "全书", value: totalWords.toLocaleString("zh-CN"), hint: "字" },
        { label: "进度", value: `${bookProgress}%`, hint: current ? "已到当前章" : "暂无章节" },
      ]}
      actions={[
        {
          key: "refresh",
          label: "同步定稿",
          icon: RefreshCw,
          onClick: () => void load(),
          title: "刷新（agent 的 wenxue_final_write 入库后同步）",
        },
        {
          key: "focus",
          label: focusMode ? "退出专注" : "专注阅读",
          icon: focusMode ? Minimize2 : Maximize2,
          onClick: () => setFocusMode((value) => !value),
          primary: focusMode,
          title: "隐藏目录与侧栏，只保留书页正文",
        },
      ]}
    >
      <div className={`wx-reading${focusMode ? " is-focus" : ""}`} data-testid="wenxue-reading-hub">
        <div className="wx-reading__workspace">
          <aside className="wx-reading__toc" aria-label="定稿目录">
            <div className="wx-reading__book">
              <span className="wx-reading__book-mark" aria-hidden>书</span>
              <div className="wx-reading__book-copy">
                <p>定稿目录 · 当前作品 {chapters.length} 章</p>
                <h2 title={bookTitle}>{bookTitle}</h2>
              </div>
            </div>

            <div className="wx-reading__toc-tools">
              <div className="wx-hub-search">
                <Search size={12} aria-hidden />
                <input
                  value={tocQuery}
                  onChange={(event) => setTocQuery(event.target.value)}
                  placeholder="查找章节…"
                  aria-label="查找章节"
                />
                <button
                  type="button"
                  className="wx-hub-search__action"
                  onClick={() => void load()}
                  title="刷新定稿目录"
                  aria-label="刷新定稿目录"
                >
                  <RefreshCw size={11} />
                </button>
              </div>
            </div>

            <div
              className="wx-reading__chapters"
              role="listbox"
              aria-label="章节目录"
            >
              {loading ? (
                <div className="px-2 py-8 text-center text-[11px] text-fg-faint">读取定稿目录中…</div>
              ) : filteredChapters.length === 0 ? (
                <div className="px-3 py-8 text-center text-[10.5px] leading-relaxed text-fg-faint">
                  {chapters.length === 0 ? "定稿目录暂时为空。" : "没有匹配的章节。"}
                </div>
              ) : (
                filteredChapters.map((chapter) => {
                  const active = chapter.path === selected;
                  return (
                    <button
                      key={chapter.path}
                      type="button"
                      role="option"
                      aria-selected={active}
                      data-testid={`wenxue-reading-item-${chapter.number || chapter.name}`}
                      className="wx-reading__chapter"
                      onClick={() => setSelected(chapter.path)}
                    >
                      <span className="wx-reading__chapter-no">{chapterLabel(chapter)}</span>
                      <span className="wx-reading__chapter-title">{chapter.title}</span>
                      <span className="wx-reading__chapter-words">{chapter.words}</span>
                    </button>
                  );
                })
              )}
            </div>

            <div className="wx-reading__toc-foot">
              {current
                ? `当前第 ${idx + 1} / ${chapters.length} 篇 · 全书约 ${readingMinutes(totalWords)} 分钟`
                : "定稿入库后，目录与阅读进度会自动出现。"}
            </div>
          </aside>

          <main className="wx-reading__main">
            {error && (
              <div className="wx-reading__error" role="alert">
                读取失败：{error}
                <button type="button" className="wx-hub-icon-btn ml-2" onClick={() => selected && void readChapter(selected)}>
                  重试
                </button>
              </div>
            )}

            {current ? (
              <>
                <div className="wx-reading__topline">
                  <button
                    type="button"
                    className="wx-hub-icon-btn"
                    onClick={() => jump(-1)}
                    disabled={idx <= 0}
                    title="上一章"
                    aria-label="上一章"
                  >
                    <ChevronRight size={12} style={{ transform: "rotate(180deg)" }} />
                  </button>
                  <div className="wx-reading__topline-copy">
                    <p>{bookTitle} · {chapterLabel(current)}</p>
                    <h2>{current.title}</h2>
                  </div>
                  <div
                    className="wx-reading__toolgroup wx-reading__typeset"
                    aria-label="阅读排版"
                  >
                    <select
                      className="wx-reading__fontselect"
                      value={fontKey}
                      onChange={(event) => setPrefs((p) => ({
                        ...p,
                        fontKey: event.target.value as ReadingFontKey,
                      }))}
                      aria-label="正文字体"
                      title="正文字体"
                    >
                      {READING_FONTS.map((font) => (
                        <option key={font.key} value={font.key}>{font.label}</option>
                      ))}
                    </select>
                    <input
                      type="range"
                      className="wx-reading__sizeslider"
                      min={READING_SIZE_MIN}
                      max={READING_SIZE_MAX}
                      step={1}
                      value={fontSize}
                      onChange={(event) => setPrefs((p) => ({
                        ...p,
                        fontSize: clampReadingSize(event.target.value),
                      }))}
                    aria-label="正文字号"
                    title={`正文字号 ${fontSize}px（Ctrl+滚轮也可缩放）`}
                    />
                    <span className="wx-reading__sizevalue" title={`当前 ${fontSize}px`}>
                      {fontSize}
                    </span>
                  </div>
                  <button
                    type="button"
                    className="wx-hub-icon-btn"
                    onClick={() => jump(1)}
                    disabled={idx < 0 || idx >= chapters.length - 1}
                    title="下一章"
                    aria-label="下一章"
                  >
                    <ChevronRight size={12} />
                  </button>
                </div>
                <div className="wx-reading__progress" aria-hidden>
                  <div className="wx-reading__progress-fill" style={{ width: `${scrollPct}%` }} />
                </div>
                <div className="wx-reading__scroll" ref={scrollRef} onScroll={handleScroll}>
                  <article className="wx-reading__paper" ref={paperRef}>
                    <header className="wx-reading__chapter-head">
                      <span>{chapterLabel(current)}</span>
                      <h1>{current.title}</h1>
                      <p>
                        {current.words.toLocaleString("zh-CN")} 字
                        <span aria-hidden> · </span>
                        约 {readingMinutes(current.words)} 分钟
                        <span aria-hidden> · </span>
                        章内进度 {scrollPct}%
                      </p>
                    </header>
                    <div
                      className="wx-reading-prose"
                      style={{
                        "--wx-reading-size": `${fontSize}px`,
                        "--wx-reading-leading": "2.02",
                        "--wx-reading-font": readingFontStack(fontKey),
                      } as CSSProperties}
                    >
                      {reading ? (
                        <div className="py-12 text-center text-[11px] text-fg-faint">正在铺开书页…</div>
                      ) : (
                        <Markdown text={displayBody} imageBasePath={current.path} />
                      )}
                    </div>
                  </article>
                </div>
              </>
            ) : (
              <WenxueHubEmpty
                icon={BookOpen}
                title={loading ? "正在检查定稿" : dirRel ? "定稿目录尚无章节" : "这部作品还没有定稿"}
                description={
                  loading
                    ? "正在读取当前小说的定稿目录。"
                    : dirRel
                      ? "把发布稿放进「定稿」目录，或让 agent 用 wenxue_final_write 入库。落盘后这里会自动同步。"
                      : "先在章节里完成正文，再把发布稿放进「定稿」目录。阅读室只收定稿，不混入草稿。"
                }
                action={(
                  <button type="button" className="wx-hub-action is-primary" onClick={() => void load()}>
                    <RefreshCw size={13} aria-hidden />
                    <span>重新检查定稿</span>
                  </button>
                )}
              />
            )}
          </main>

          <aside className="wx-reading__aux" aria-label="阅读辅助">
            {current ? (
              <>
                <section className="wx-hub-paper wx-reading__aux-card">
                  <h3>本章信息</h3>
                  <div className="wx-reading__aux-stats">
                    <div className="wx-reading__aux-stat">
                      <span>字数</span>
                      <strong>{current.words.toLocaleString("zh-CN")}</strong>
                    </div>
                    <div className="wx-reading__aux-stat">
                      <span>阅读</span>
                      <strong>{readingMinutes(current.words)} 分</strong>
                    </div>
                    <div className="wx-reading__aux-stat">
                      <span>章序</span>
                      <strong>{idx + 1}/{chapters.length}</strong>
                    </div>
                    <div className="wx-reading__aux-stat">
                      <span>小节</span>
                      <strong>{headings.length}</strong>
                    </div>
                  </div>
                </section>

                <section className="wx-hub-paper wx-reading__aux-card">
                  <h3><ListTree size={11} className="mr-1 inline" aria-hidden />章内导航</h3>
                  {headings.length > 0 ? (
                    <div className="wx-reading__outline">
                      {headings.map((heading, headingIndex) => (
                        <button
                          key={`${heading.title}-${headingIndex}`}
                          type="button"
                          style={{ "--wx-outline-level": heading.level } as CSSProperties}
                          onClick={() => jumpToHeading(headingIndex)}
                          title={heading.title}
                        >
                          {heading.title}
                        </button>
                      ))}
                    </div>
                  ) : (
                    <p className="mt-2 text-[10px] leading-relaxed text-fg-faint">
                      本章没有小标题。连续正文更适合沉浸在书中阅读。
                    </p>
                  )}
                </section>

                <section className="wx-hub-paper wx-reading__aux-card">
                  <h3>阅读节奏</h3>
                  <p className="mt-2 text-[10px] leading-relaxed text-fg-faint">
                    正文字号 {fontSize}px · {fontLabel}，行距 2.02；书页上 Ctrl+滚轮可缩放字号。建议每 {Math.max(1, Math.round(readingMinutes(current.words) / 2))} 分钟停一次，检查节奏与信息密度。
                  </p>
                  <div className="mt-2 flex items-center gap-1.5 text-[9.5px] text-fg-faint">
                    <Clock size={10} aria-hidden />
                    <span>滚动进度 {scrollPct}%</span>
                    <FileText size={10} className="ml-auto" aria-hidden />
                    <span>{bookProgress}% 全书</span>
                  </div>
                </section>
              </>
            ) : (
              <section className="wx-hub-paper wx-reading__aux-card">
                <h3>阅读提示</h3>
                <p className="mt-2 text-[10px] leading-relaxed text-fg-faint">
                  定稿只承载发布版本。章节草稿、设定和角色卡保留在各自目录，阅读时不会被未定稿内容打断。
                </p>
              </section>
            )}

            <div className="wx-reading__navline">
              <button
                type="button"
                className="wx-hub-icon-btn"
                onClick={() => jump(-1)}
                disabled={idx <= 0}
              >
                <ChevronRight size={11} style={{ transform: "rotate(180deg)" }} aria-hidden />
                上一章
              </button>
              <button
                type="button"
                className="wx-hub-icon-btn"
                onClick={() => jump(1)}
                disabled={idx < 0 || idx >= chapters.length - 1}
              >
                下一章
                <ChevronRight size={11} aria-hidden />
              </button>
            </div>
          </aside>
        </div>
      </div>
    </WenxueHubFrame>
  );
}
