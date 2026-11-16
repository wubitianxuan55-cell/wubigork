import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { CSSProperties } from "react";
import { message, Modal, Tabs } from "antd";
import {
  Check, Clock, Download, FileText, Pencil, Plus, RefreshCw,
  ScrollText, Search, Sparkles, Trash2,
} from "../icons";
import { app } from "../lib/bridge";
import { subscribe } from "../../events";
import { useComposerInsertStore } from "../lib/store";
import { Markdown } from "./Markdown";
import { WenxueHubEmpty, WenxueHubFrame } from "./WenxueHubFrame";
import {
  countWords, firstParagraph, markdownHeadings, settingCategory,
  withoutLeadingH1,
} from "./wenxueHubUtils";

// WenxueSettingsHub — 文学板块「设定」标签页。
// 数据面=工作空间设定文档（设定/|worldview/|设定集/）+ 时间线
// （时间线/|timeline/），与 agent 的 write_file / wenxue_timeline_* 工具同源。
// UI 从双组文件列表升级为世界观资料库：分类索引、正文阅读、文档大纲与
// 章序时间线分层呈现。agent 落盘后通过 filetree 通道自动刷新。

interface DocRow {
  group: "settings" | "timeline";
  path: string;
  name: string;
  words: number;
  excerpt: string;
  mtimeMs: number;
  text: string;
}

interface TimelineEvent {
  chapter: number;
  text: string;
  line: number;
}

interface TimelineGroup {
  chapter: number;
  label: string;
  events: TimelineEvent[];
}

const SETTINGS_DIRS = ["设定", "worldview", "设定集"];
const TIMELINE_DIRS = ["时间线", "timeline"];

const GEN_PRESETS: { key: string; label: string; file: string; brief: string }[] = [
  { key: "worldview", label: "世界观", file: "设定/世界观.md", brief: "世界背景、时代、大陆/城市格局、核心冲突与运行逻辑" },
  { key: "power", label: "力量体系", file: "设定/力量体系.md", brief: "修炼/能力分级、突破条件、代价与限制、彼此克制关系" },
  { key: "rules", label: "规则", file: "设定/规则.md", brief: "世界硬规则（禁忌/底线/不可违背项）与违反后果，成体系可检索" },
  { key: "factions", label: "势力", file: "设定/势力.md", brief: "主要势力（名称/立场/头目/与主角关系/当前态势）" },
  { key: "geo", label: "地理", file: "设定/地理.md", brief: "地图层次（区域/城市/地标）与位置关系、旅途耗时感" },
];

function firstHeadingName(text: string, fallback: string): string {
  for (const line of text.split("\n")) {
    const t = line.trim();
    if (t.startsWith("# ")) return t.slice(2).trim() || fallback;
    if (t) break;
  }
  return fallback;
}

function excerptOf(text: string, max = 90): string {
  const body = text
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => !!line && !line.startsWith("#"));
  const joined = body.join(" ").replace(/\s+/g, " ").trim();
  const chars = [...joined];
  return chars.length > max ? `${chars.slice(0, max).join("")}…` : joined;
}

function parseTimeline(text: string): TimelineEvent[] {
  const out: TimelineEvent[] = [];
  const lines = text.split("\n");
  for (let index = 0; index < lines.length; index++) {
    const raw = lines[index];
    const t = raw.trim();
    if (!t || t.startsWith("#")) continue;
    let chapter = 0;
    let body = t.replace(/^[-•*·\s]+/, "");
    const match = /^第(\d+)[章话]\s*[·:：\-—]?\s*/.exec(body);
    if (match) {
      chapter = parseInt(match[1], 10) || 0;
      body = body.slice(match[0].length).trim();
    } else if (/^第[零一二三四五六七八九十百]+[章话]/.test(body)) {
      body = `${body}（章号未解析：中文序号不可锚定，请改用阿拉伯数字）`;
    }
    out.push({ chapter, text: body, line: index + 1 });
  }
  return out
    .map((event, index) => ({ event, index }))
    .sort((a, b) => (a.event.chapter !== b.event.chapter
      ? a.event.chapter - b.event.chapter
      : a.index - b.index))
    .map(({ event }) => event);
}

function groupTimeline(events: TimelineEvent[]): TimelineGroup[] {
  const groups = new Map<number, TimelineEvent[]>();
  for (const event of events) {
    const list = groups.get(event.chapter) ?? [];
    list.push(event);
    groups.set(event.chapter, list);
  }
  return [...groups.entries()]
    .sort(([a], [b]) => a - b)
    .map(([chapter, list]) => ({
      chapter,
      label: chapter > 0 ? `第${chapter}章` : "未锚",
      events: list,
    }));
}

function categoryGlyph(category: ReturnType<typeof settingCategory>): string {
  switch (category) {
    case "世界": return "世";
    case "体系": return "体";
    case "规则": return "则";
    case "势力": return "势";
    case "人物": return "人";
    default: return "资";
  }
}

export function WenxueSettingsHub() {
  const [dirRel, setDirRel] = useState("");
  const [docs, setDocs] = useState<DocRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState("all");
  const [selected, setSelected] = useState<string>("");
  const [draft, setDraft] = useState("");
  const [baseline, setBaseline] = useState(0);
  const [dirty, setDirty] = useState(false);
  const [editing, setEditing] = useState(false);
  const [saving, setSaving] = useState(false);
  const [creating, setCreating] = useState(false);
  const [newName, setNewName] = useState("");
  const [genTab, setGenTab] = useState("preset");
  const [genName, setGenName] = useState("");
  const reloadRef = useRef(0);
  const paperRef = useRef<HTMLElement>(null);
  const editorRef = useRef<HTMLTextAreaElement>(null);
  const detailRef = useRef<HTMLElement>(null);

  const load = useCallback(async () => {
    const token = ++reloadRef.current;
    setLoading(true);
    setError("");
    try {
      const rootEntries = await app.ListDir("");
      let settingsDir = "";
      for (const candidate of SETTINGS_DIRS) {
        if (rootEntries.some((entry) => entry.isDir && entry.name === candidate)) {
          settingsDir = candidate;
          break;
        }
      }
      let timelineDir = "";
      for (const candidate of TIMELINE_DIRS) {
        if (rootEntries.some((entry) => entry.isDir && entry.name === candidate)) {
          timelineDir = candidate;
          break;
        }
      }
      setDirRel(settingsDir);

      const rows: DocRow[] = [];
      const readDocs = async (dir: string, group: DocRow["group"]) => {
        if (!dir) return;
        const entries = await app.ListDir(dir);
        for (const entry of entries) {
          if (entry.isDir || !/\.(md|markdown|txt)$/i.test(entry.name)) continue;
          const rel = `${dir}/${entry.name}`;
          const file = await app.GaeaReadTextFile(rel);
          rows.push({
            group,
            path: rel,
            name: firstHeadingName(file.text, entry.name.replace(/\.(md|markdown|txt)$/i, "")),
            words: countWords(file.text),
            excerpt: excerptOf(file.text),
            mtimeMs: file.mtimeMs,
            text: file.text,
          });
        }
      };
      await readDocs(settingsDir, "settings");
      await readDocs(timelineDir, "timeline");
      rows.sort((a, b) => a.path.localeCompare(b.path, "zh-Hans-CN"));
      if (token !== reloadRef.current) return;
      setDocs(rows);
    } catch (err) {
      if (token === reloadRef.current) setError(err instanceof Error ? err.message : String(err));
    } finally {
      if (token === reloadRef.current) setLoading(false);
    }
  }, []);

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
      const hit = paths.some((path) => {
        if (typeof path !== "string") return false;
        const norm = path.replaceAll("\\", "/");
        return [...SETTINGS_DIRS, ...TIMELINE_DIRS].some((dir) => norm.startsWith(`${dir}/`));
      });
      if (hit) void load();
    });
  }, [load]);

  const selectedRow = useMemo(
    () => docs.find((doc) => doc.path === selected) ?? null,
    [docs, selected],
  );

  const openDoc = (row: DocRow) => {
    if (detailRef.current) detailRef.current.scrollTop = 0;
    setSelected(row.path);
    setDraft(row.text);
    setBaseline(row.mtimeMs);
    setDirty(false);
    setEditing(false);
  };

  useEffect(() => {
    if (docs.length === 0) return;
    if (selectedRow) return;
    openDoc(docs[0]);
  }, [docs, selectedRow]);

  useEffect(() => {
    if (!selectedRow || dirty) return;
    if (selectedRow.mtimeMs !== baseline) {
      setDraft(selectedRow.text);
      setBaseline(selectedRow.mtimeMs);
    }
  }, [selectedRow, baseline, dirty]);

  useEffect(() => {
    if (editing) editorRef.current?.focus();
  }, [editing]);

  useEffect(() => {
    requestAnimationFrame(() => {
      if (detailRef.current) detailRef.current.scrollTop = 0;
    });
  }, [selected]);

  const save = async () => {
    if (!selected) return;
    setSaving(true);
    try {
      await app.GaeaWriteWorkspaceFile(selected, draft, baseline);
      const file = await app.GaeaReadTextFile(selected);
      setBaseline(file.mtimeMs);
      setDraft(file.text);
      setDirty(false);
      setDocs((rows) => rows.map((row) => (row.path === selected
        ? {
            ...row,
            text: file.text,
            mtimeMs: file.mtimeMs,
            words: countWords(file.text),
            excerpt: excerptOf(file.text),
          }
        : row)));
      message.success("设定已保存（agent 侧工具所见即所得）");
    } catch (err) {
      message.error(err instanceof Error ? err.message : "保存失败（可能已被外部修改，请刷新后重试）");
    } finally {
      setSaving(false);
    }
  };

  const openCreate = () => {
    setNewName("");
    setGenTab("blank");
    setGenName("");
    setCreating(true);
  };

  const openGenerate = () => {
    setNewName("");
    setGenTab("preset");
    setGenName("");
    setCreating(true);
  };

  const nameTaken = (rel: string): boolean =>
    docs.some((doc) => doc.path.toLowerCase() === rel.toLowerCase());

  const createBlank = async () => {
    const name = newName.trim();
    if (!name) return;
    const dir = dirRel || SETTINGS_DIRS[0];
    const rel = `${dir}/${name}.md`;
    if (nameTaken(rel)) {
      message.error(`同名设定文档已存在（${rel}）`);
      return;
    }
    try {
      await app.GaeaWriteWorkspaceFile(
        rel,
        `# ${name}\n\n（待补——可在阅读页进入编辑手写，或用「AI 生成」让 agent 基于现有章节、角色卡与时间线起草。）\n`,
        0,
      );
      message.success("设定文档已创建");
      setCreating(false);
      await load();
      const fresh = await app.GaeaReadTextFile(rel);
      setSelected(rel);
      setDraft(fresh.text);
      setBaseline(fresh.mtimeMs);
      setDirty(false);
      setEditing(true);
    } catch (err) {
      message.error(err instanceof Error ? err.message : "创建失败");
    }
  };

  const requestGen = (file: string, brief: string) => {
    useComposerInsertStore.getState().requestText(
      `请为这部小说生成设定文档并保存到工作空间相对路径 ${file}（用 write_file；`
      + `若该文件已存在，先 read_file 再在其基础上修订扩充）。内容要求：${brief}。`
      + `生成前先用 wenxue_chapter_list 与 wenxue_character_list 摸清已有章节与角色，`
      + `必要时参考 wenxue_timeline_read，保证设定与既有正文/角色一致；`
      + `文档用 Markdown（首行 # 标题，## 小节组织），与「时间线」目录中的事实不冲突。`,
    );
    setCreating(false);
    message.success("已写入对话输入框——发送后 agent 生成落盘，本页自动出现该文档");
  };

  const askAiImprove = (row: DocRow) => {
    useComposerInsertStore.getState().requestText(
      `请阅读设定文档 @${row.path}，在保持既有事实一致的前提下补全缺失小节、修订含糊表述；`
      + `与「时间线」目录中的事件冲突处以时间线为准；改完用 write_file 保存（先 read_file 再整体重写）。`,
    );
  };

  const remove = (row: DocRow) => {
    Modal.confirm({
      title: `删除设定文档「${row.name}」？`,
      content: `${row.path} 将被删除，不可恢复。引用该文档的章节不会自动改写。`,
      okText: "删除",
      okButtonProps: { danger: true },
      onOk: async () => {
        try {
          await app.GaeaDeleteWorkspacePaths([row.path]);
          if (selected === row.path) {
            setSelected("");
            setDraft("");
            setDirty(false);
            setEditing(false);
          }
          await load();
          message.success("已删除");
        } catch (err) {
          message.error(err instanceof Error ? err.message : "删除失败");
        }
      },
    });
  };

  const docsWithCategory = useMemo(
    () => docs.map((doc) => ({
      doc,
      category: doc.group === "timeline" ? "时间线" : settingCategory(doc.name, doc.text),
    })),
    [docs],
  );
  const categories = useMemo(() => {
    const counts = new Map<string, number>();
    for (const item of docsWithCategory) {
      counts.set(item.category, (counts.get(item.category) ?? 0) + 1);
    }
    return [...counts.entries()];
  }, [docsWithCategory]);
  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return docsWithCategory.filter(({ doc, category }) => {
      if (filter !== "all" && category !== filter) return false;
      if (!q) return true;
      return `${doc.name} ${doc.excerpt} ${doc.path} ${category}`.toLowerCase().includes(q);
    });
  }, [docsWithCategory, filter, query]);
  const settingsDocs = filtered.filter(({ doc }) => doc.group === "settings");
  const timelineDocs = filtered.filter(({ doc }) => doc.group === "timeline");

  const allTimelineEvents = useMemo(
    () => docs
      .filter((doc) => doc.group === "timeline")
      .flatMap((doc) => parseTimeline(doc.text)),
    [docs],
  );
  const unanchored = allTimelineEvents.filter((event) => event.chapter <= 0).length;
  const coveredCategories = categories.filter(([category]) => category !== "时间线").length;

  const selectedCategory = selectedRow
    ? selectedRow.group === "timeline"
      ? "时间线"
      : settingCategory(selectedRow.name, selectedRow.text)
    : "";
  const displayText = useMemo(
    () => (selectedRow ? withoutLeadingH1(selectedRow.text) : ""),
    [selectedRow],
  );
  const selectedHeadings = useMemo(
    () => (selectedRow ? markdownHeadings(displayText) : []),
    [displayText, selectedRow],
  );
  const parsedTimeline = useMemo(() => {
    if (!selectedRow || selectedRow.group !== "timeline" || editing) return [];
    return parseTimeline(selectedRow.text);
  }, [editing, selectedRow]);
  const parsedTimelineGroups = useMemo(() => groupTimeline(parsedTimeline), [parsedTimeline]);
  const timelineAnchored = parsedTimeline.filter((event) => event.chapter > 0).length;
  const timelineMaxChapter = parsedTimeline.reduce(
    (max, event) => Math.max(max, event.chapter),
    0,
  );

  const startEditing = () => {
    if (!selectedRow) return;
    setDraft(selectedRow.text);
    setBaseline(selectedRow.mtimeMs);
    setDirty(false);
    setEditing(true);
  };

  const jumpToHeading = (headingIndex: number) => {
    const nodes = paperRef.current?.querySelectorAll<HTMLElement>(
      ".wx-settings-doc-paper .md h1, .wx-settings-doc-paper .md h2, .wx-settings-doc-paper .md h3",
    );
    nodes?.[headingIndex]?.scrollIntoView({ behavior: "smooth", block: "start" });
  };

  const docListRow = ({
    doc,
    category,
  }: {
    doc: DocRow;
    category: string;
  }) => (
    <button
      key={doc.path}
      type="button"
      role="option"
      aria-selected={selected === doc.path}
      data-testid={`wenxue-settings-doc-${doc.name}`}
      className="wx-hub-list-row"
      onClick={() => openDoc(doc)}
    >
      <span className="wx-hub-monogram" aria-hidden>
        {doc.group === "timeline" ? "序" : categoryGlyph(category as ReturnType<typeof settingCategory>)}
      </span>
      <span className="wx-settings-row__copy">
        <span className="wx-settings-row__top">
          <span className="wx-settings-row__name">{doc.name}</span>
          <span className="wx-settings-row__meta">{doc.words}</span>
        </span>
        <span className="mt-1 flex items-center gap-1">
          <span className="wx-settings-row__badge">{category}</span>
          <span className="truncate text-[9px] text-fg-faint">{doc.path}</span>
        </span>
        <span className="wx-settings-row__excerpt">{doc.excerpt || "（待补）"}</span>
      </span>
    </button>
  );

  return (
    <WenxueHubFrame
      tone="settings"
      eyebrow="世界观资料库"
      title="世界设定志"
      description="让世界观从散落的文件变成可检索的档案：先找维度，再读全文，最后核对时间。"
      metrics={[
        { label: "设定", value: docs.filter((doc) => doc.group === "settings").length, hint: "篇文档" },
        { label: "门类", value: coveredCategories, hint: categories.map(([name]) => name).filter((name) => name !== "时间线").join(" / ") || "待建立" },
        { label: "时间线", value: allTimelineEvents.length, hint: `${unanchored} 条未锚` },
      ]}
      actions={[
        {
          key: "new",
          label: "新建文档",
          icon: Plus,
          onClick: openCreate,
          testId: "wenxue-settings-new",
        },
        {
          key: "generate",
          label: "AI 生成",
          icon: Sparkles,
          onClick: openGenerate,
          primary: true,
          testId: "wenxue-settings-gen",
          title: "AI 生成设定文档（世界观/力量体系/规则/势力/地理/自定义）",
        },
      ]}
    >
      <div className="wx-settings__workspace" data-testid="wenxue-settings-hub">
        <aside className="wx-hub-sidebar" aria-label="设定索引">
          <div className="wx-hub-sidebar__head">
            <span className="wx-hub-sidebar__title"><ScrollText size={12} className="mr-1 inline" aria-hidden />资料索引</span>
            <span className="wx-hub-sidebar__count">{filtered.length}/{docs.length}</span>
          </div>
          <div className="wx-hub-search">
            <Search size={12} aria-hidden />
            <input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="搜索设定、势力、规则…"
              aria-label="搜索设定"
            />
            <button
              type="button"
              className="wx-hub-search__action"
              onClick={() => void load()}
              title="刷新（agent 侧落盘后同步）"
              aria-label="刷新设定"
            >
              <RefreshCw size={11} />
            </button>
          </div>
          <div className="wx-hub-chips" aria-label="设定分类">
            <button
              type="button"
              className="wx-hub-chip"
              aria-pressed={filter === "all"}
              onClick={() => setFilter("all")}
            >
              全部 {docs.length}
            </button>
            {categories.map(([category, count]) => (
              <button
                key={category}
                type="button"
                className="wx-hub-chip"
                aria-pressed={filter === category}
                onClick={() => setFilter(category)}
              >
                {category} {count}
              </button>
            ))}
          </div>

          {error && (
            <div className="mx-3 mb-2 rounded-md border border-warning/30 bg-warning/5 px-2 py-1.5 text-[10px] text-warning" role="alert">
              {error}
            </div>
          )}

          <div className="wx-settings__groups" role="listbox" aria-label="设定文档列表">
            <div className="wx-settings__group-label">
              <span><FileText size={10} className="mr-1 inline" aria-hidden />设定文档</span>
              <span>{settingsDocs.length}</span>
            </div>
            {settingsDocs.length === 0 ? (
              <div className="wx-settings__empty-group">
                尚无匹配的设定文档。可用「AI 生成」让 agent 先铺世界骨架，或手写第一份。
              </div>
            ) : settingsDocs.map(docListRow)}

            <div className="wx-settings__group-label">
              <span><Clock size={10} className="mr-1 inline" aria-hidden />时间线</span>
              <span>{timelineDocs.length}</span>
            </div>
            {timelineDocs.length === 0 ? (
              <div className="wx-settings__empty-group">
                时间线目录为空。agent 写章后可用 wenxue_timeline_append 记录事件。
              </div>
            ) : timelineDocs.map(docListRow)}
          </div>

          <div className="wx-hub-sidebar__foot">
            与 agent 的 write_file / 时间线工具同源。落盘（write_file / wenxue_timeline_append）后会自动刷新。
          </div>
        </aside>

        <main key={selected} className="wx-settings-detail" ref={detailRef}>
          {!selectedRow ? (
            <WenxueHubEmpty
              icon={ScrollText}
              title={loading ? "正在整理世界档案" : "从第一份设定开始"}
              description="设定文档保存在工作空间「设定」目录，时间线保存在「时间线」目录。它们都是 Markdown，agent 与这里读写同一份事实。"
              action={(
                <div className="flex items-center gap-2">
                  <button type="button" className="wx-hub-action" onClick={openCreate}>
                    <Plus size={13} aria-hidden /> 新建文档
                  </button>
                  <button type="button" className="wx-hub-action is-primary" onClick={openGenerate}>
                    <Sparkles size={13} aria-hidden /> AI 起草
                  </button>
                </div>
              )}
            />
          ) : (
            <div className="wx-settings-dossier">
              <section className="wx-hub-paper wx-settings-hero">
                <div className="wx-settings-hero__copy">
                  <div className="wx-settings-hero__kicker">{selectedCategory.toUpperCase()} · {selectedRow.group === "timeline" ? "CHRONICLE" : "WORLD ARCHIVE"}</div>
                  <h1>{selectedRow.name}</h1>
                  <div className="wx-settings-hero__meta">
                    <span className="wx-hub-badge is-accent">{selectedCategory}</span>
                    <span className="wx-hub-badge">{selectedRow.words.toLocaleString("zh-CN")} 字</span>
                    <span>{selectedRow.path}</span>
                    {dirty && <span className="text-warning">有未保存修改</span>}
                  </div>
                </div>
                <div className="wx-settings-hero__actions">
                  {!editing ? (
                    <button type="button" className="wx-hub-icon-btn" onClick={startEditing}>
                      <Pencil size={11} aria-hidden /> 编辑
                    </button>
                  ) : (
                    <>
                      <button
                        type="button"
                        className="wx-hub-action is-primary !min-h-[30px]"
                        onClick={() => void save()}
                        disabled={!dirty || saving}
                      >
                        {saving ? "保存中…" : <><Check size={11} aria-hidden /> 保存（CAS）</>}
                      </button>
                      <button
                        type="button"
                        className="wx-hub-icon-btn"
                        onClick={() => {
                          if (!dirty) setEditing(false);
                          else message.warning("有未保存改动——保存后再退出编辑");
                        }}
                      >
                        完成
                      </button>
                    </>
                  )}
                  <button
                    type="button"
                    className="wx-hub-icon-btn"
                    title="让 AI 在既有事实一致的前提下补全修订（写入输入框）"
                    onClick={() => askAiImprove(selectedRow)}
                  >
                    <Sparkles size={10} aria-hidden /> 让 AI 完善
                  </button>
                  <button
                    type="button"
                    className="wx-hub-icon-btn is-danger"
                    title="删除设定文档"
                    aria-label={`删除设定文档 ${selectedRow.name}`}
                    onClick={() => remove(selectedRow)}
                  >
                    <Trash2 size={10} aria-hidden />
                  </button>
                </div>
              </section>

              {editing ? (
                <section className="wx-hub-paper wx-settings-editor">
                  <div className="wx-hub-meta-row justify-between">
                    <span className="text-[10px] text-fg-faint">Markdown 原文 · 保存时执行 CAS，外部改动不会被静默覆盖</span>
                    <span className="text-[9.5px] text-fg-faint">{[...draft].length.toLocaleString("zh-CN")} 字符</span>
                  </div>
                  <textarea
                    ref={editorRef}
                    value={draft}
                    onChange={(event) => {
                      setDraft(event.target.value);
                      setDirty(true);
                    }}
                    spellCheck={false}
                    aria-label={`编辑设定 ${selected}`}
                  />
                </section>
              ) : selectedRow.group === "timeline" ? (
                <section className="wx-hub-paper wx-settings-timeline" data-testid="wenxue-settings-timeline">
                  <div className="wx-settings-timeline__stats">
                    <div className="wx-settings-timeline__stat">
                      <span>事件</span>
                      <strong>{parsedTimeline.length}</strong>
                    </div>
                    <div className="wx-settings-timeline__stat">
                      <span>已锚定</span>
                      <strong>{timelineAnchored}</strong>
                    </div>
                    <div className="wx-settings-timeline__stat">
                      <span>未锚定</span>
                      <strong>{parsedTimeline.length - timelineAnchored}</strong>
                    </div>
                    <div className="wx-settings-timeline__stat">
                      <span>最远章序</span>
                      <strong>{timelineMaxChapter || "—"}</strong>
                    </div>
                  </div>

                  {parsedTimeline.some((event) => event.chapter <= 0) && (
                    <p className="wx-settings-timeline__warn">
                      有 {parsedTimeline.length - timelineAnchored} 条事件无法按章号定位。它们会保留在「未锚」组，不会被静默丢弃；建议改为「第N章 · 事件」格式。
                    </p>
                  )}

                  {parsedTimeline.length === 0 ? (
                    <WenxueHubEmpty
                      icon={Clock}
                      title="时间线还没有事件"
                      description="agent 写章后可用 wenxue_timeline_append 追加事件，或在编辑器中按「第N章 · 一句话事件」逐行记录。"
                      action={(
                        <button type="button" className="wx-hub-action" onClick={startEditing}>
                          <Pencil size={13} aria-hidden /> 开始记录
                        </button>
                      )}
                    />
                  ) : (
                    <div className="wx-settings-timeline__axis">
                      {parsedTimelineGroups.map((group) => (
                        <section
                          key={group.chapter}
                          className={`wx-settings-timeline__group${group.chapter > 0 ? "" : " is-unanchored"}`}
                        >
                          <span className="wx-settings-timeline__chapter">{group.label}</span>
                          <div className="wx-settings-timeline__events">
                            {group.events.map((event) => (
                              <article key={`${event.line}-${event.text}`} className="wx-settings-timeline__event">
                                <p>{event.text}</p>
                                <span>第 {event.line} 行 · {event.chapter > 0 ? `锚定至第 ${event.chapter} 章` : "未锚定"}</span>
                              </article>
                            ))}
                          </div>
                        </section>
                      ))}
                    </div>
                  )}
                </section>
              ) : (
                <div className="wx-settings-doc-layout">
                  <article className="wx-hub-paper wx-settings-doc-paper" ref={paperRef}>
                    <div data-testid="wenxue-settings-doc">
                      <Markdown text={displayText} imageBasePath={selectedRow.path} />
                    </div>
                  </article>

                  <aside className="wx-hub-paper wx-settings-outline">
                    <h2>文档大纲</h2>
                    {selectedHeadings.length > 0 ? (
                      <nav aria-label="设定文档大纲">
                        {selectedHeadings.map((heading, index) => (
                          <button
                            key={`${heading.title}-${index}`}
                            type="button"
                            style={{ "--wx-outline-level": heading.level } as CSSProperties}
                            onClick={() => jumpToHeading(index)}
                            title={heading.title}
                          >
                            {heading.title}
                          </button>
                        ))}
                      </nav>
                    ) : (
                      <p className="mt-2 text-[9.5px] leading-relaxed text-fg-faint">
                        这份文档还没有标题层级。用 ## 小节组织，会更容易被作者和 agent 检索。
                      </p>
                    )}
                    <div className="wx-settings-outline__meta">
                      {firstParagraph(selectedRow.text, 110) || "暂无摘要。"}
                    </div>
                  </aside>
                </div>
              )}
            </div>
          )}
        </main>
      </div>

      <Modal
        title="新建世界档案"
        open={creating}
        onCancel={() => setCreating(false)}
        footer={null}
        destroyOnHidden
        width={590}
      >
        <Tabs
          activeKey={genTab}
          onChange={(key) => setGenTab(key)}
          items={[
            {
              key: "blank",
              label: "空白文档",
              children: (
                <div>
                  <input
                    autoFocus
                    value={newName}
                    onChange={(event) => setNewName(event.target.value)}
                    onKeyDown={(event) => {
                      if (event.key === "Enter") void createBlank();
                    }}
                    placeholder="文档名（如：世界观）"
                    aria-label="文档名"
                    className="w-full rounded-md px-2.5 py-2 text-[13px] outline-none"
                    style={{
                      background: "var(--md-sys-color-surface-container)",
                      border: "1px solid var(--md-sys-color-outline-variant)",
                      color: "var(--md-sys-color-text)",
                    }}
                  />
                  <div className="mt-2 text-[11px] text-fg-faint">
                    将在「{dirRel || SETTINGS_DIRS[0]}/」下创建 &lt;文档名&gt;.md。
                  </div>
                  <button
                    type="button"
                    className="wx-hub-action is-primary mt-3 w-full justify-center"
                    onClick={() => void createBlank()}
                    disabled={!newName.trim()}
                  >
                    <Plus size={12} aria-hidden /> 创建
                  </button>
                </div>
              ),
            },
            {
              key: "preset",
              label: <span className="flex items-center gap-1"><Sparkles size={12} /> AI 生成</span>,
              children: (
                <div>
                  <div className="grid grid-cols-2 gap-1.5">
                    {GEN_PRESETS.map((preset) => (
                      <button
                        key={preset.key}
                        type="button"
                        data-testid={`wenxue-settings-gen-${preset.key}`}
                        className="rounded-lg border px-3 py-2.5 text-left transition-colors hover:border-accent/60"
                        style={{
                          borderColor: "var(--md-sys-color-outline-variant)",
                          background: "var(--md-sys-color-surface-container)",
                        }}
                        onClick={() => requestGen(preset.file, preset.brief)}
                      >
                        <div className="flex items-center gap-2">
                          <span className="flex h-7 w-7 items-center justify-center rounded-md bg-accent/10 text-accent">
                            <Sparkles size={12} aria-hidden />
                          </span>
                          <span className="text-[12px] font-medium text-fg">{preset.label}</span>
                        </div>
                        <div className="mt-1.5 text-[10px] leading-relaxed text-fg-faint">{preset.brief}</div>
                        <div className="mt-1 font-mono text-[9px] text-fg-faint/70">{preset.file}</div>
                      </button>
                    ))}
                  </div>
                  <div className="mt-3 flex items-center gap-1.5">
                    <input
                      value={genName}
                      onChange={(event) => setGenName(event.target.value)}
                      placeholder="自定义文档名（如：货币与商路）"
                      aria-label="自定义文档名"
                      className="min-w-0 flex-1 rounded-md px-2.5 py-2 text-[12px] outline-none"
                      style={{
                        background: "var(--md-sys-color-surface-container)",
                        border: "1px solid var(--md-sys-color-outline-variant)",
                        color: "var(--md-sys-color-text)",
                      }}
                    />
                    <button
                      type="button"
                      className="wx-hub-icon-btn"
                      onClick={() => requestGen(
                        `${dirRel || SETTINGS_DIRS[0]}/${genName.trim()}.md`,
                        `围绕「${genName.trim()}」的成体系设定`,
                      )}
                      disabled={!genName.trim()}
                      data-testid="wenxue-settings-gen-custom"
                    >
                      <Download size={11} aria-hidden /> 生成
                    </button>
                  </div>
                  <div className="mt-2 rounded-md px-2.5 py-2 text-[10px] leading-relaxed text-fg-faint" style={{ background: "var(--md-sys-color-surface-container)" }}>
                    生成指令写入对话输入框。agent 会先摸清已有章节、角色与时间线，再经 write_file 落盘；生成后这里会自动出现文档。
                  </div>
                </div>
              ),
            },
          ]}
        />
      </Modal>
    </WenxueHubFrame>
  );
}
