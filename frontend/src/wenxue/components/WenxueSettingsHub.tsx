import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { message, Modal, Tabs } from "antd";
import { Check, Download, FileText, Pencil, Plus, RefreshCw, ScrollText, Search, Sparkles, Trash2, Clock } from "../icons";
import { app } from "../lib/bridge";
import { subscribe } from "../../events";
import { useComposerInsertStore } from "../lib/store";

// WenxueSettingsHub — 图文板块「设定」标签页（2026-11-16 设定批）。
// 数据面=小说工作空间的设定文档（设定/|worldview/|设定集/ 目录，与后端
// wenxueDirCandidates "settings" kind 同源）+ 时间线（时间线/|timeline/，与
// wenxue_timeline_read/append 工具同源）——UI 编辑即 agent 所见；agent 用
// write_file 落设定文档后，gaea-wenxue-filetree 通道自动重载。
// 形态：master-detail（左列表「设定文档」+「时间线」两组，右详情）；
// 设定文档结构化阅读（#/## 标题+列表轻渲染）+ CAS 裸文本编辑；时间线结构化
// 事件视图（「第N章」锚解析按章序）；AI 生成 Modal（预设 世界观/力量体系/
// 规则/势力/地理/自定义 → Composer 注入生成指令，agent 经 write_file 落盘）。

interface DocRow {
  group: "settings" | "timeline";
  path: string;
  name: string;
  words: number;
  excerpt: string;
  mtimeMs: number;
  text: string;
}

const SETTINGS_DIRS = ["设定", "worldview", "设定集"];
const TIMELINE_DIRS = ["时间线", "timeline"];

// AI 生成预设：目标文件 + 生成指令（Composer 注入；agent 经 write_file 落盘）。
const GEN_PRESETS: { key: string; label: string; file: string; brief: string }[] = [
  { key: "worldview", label: "世界观", file: "设定/世界观.md", brief: "世界背景、时代、大陆/城市格局、核心冲突与运行逻辑" },
  { key: "power", label: "力量体系", file: "设定/力量体系.md", brief: "修炼/能力分级、突破条件、代价与限制、彼此克制关系" },
  { key: "rules", label: "规则", file: "设定/规则.md", brief: "世界硬规则（禁忌/底线/不可违背项）与违反后果，成体系可检索" },
  { key: "factions", label: "势力", file: "设定/势力.md", brief: "主要势力（名称/立场/头目/与主角关系/当前态势）" },
  { key: "geo", label: "地理", file: "设定/地理.md", brief: "地图层次（区域/城市/地标）与位置关系、旅途耗时感" },
];

function countWords(s: string): number {
  let n = 0;
  for (const ch of s) if (!/\s/.test(ch)) n++;
  return n;
}

function firstHeadingName(text: string, fallback: string): string {
  for (const line of text.split("\n")) {
    const t = line.trim();
    if (t.startsWith("# ")) return t.slice(2).trim() || fallback;
    if (t) break;
  }
  return fallback;
}

function excerptOf(text: string, max = 70): string {
  const body = text
    .split("\n")
    .map((l) => l.trim())
    .filter((l) => l && !l.startsWith("#"));
  const joined = body.join(" ").replace(/\s+/g, " ").trim();
  const r = [...joined];
  return r.length > max ? r.slice(0, max).join("") + "…" : joined;
}

// ── 时间线结构化解析：「第N章」锚 + 事件文本（与后端 wenxueAnchorNumber 同口径：
// ASCII 数字；中文序号落未锚组并如实展示原文行）──

interface TimelineEvent {
  chapter: number;
  text: string;
  line: number;
}

function parseTimeline(text: string): TimelineEvent[] {
  const out: TimelineEvent[] = [];
  for (let i = 0; i < text.split("\n").length; i++) {
    const raw = text.split("\n")[i];
    const t = raw.trim();
    if (!t || t.startsWith("#")) continue;
    let chapter = 0;
    let body = t.replace(/^[-•*·\s]+/, "");
    const m = /^第(\d+)[章话]\s*[·:：\-—]?\s*/.exec(body);
    if (m) {
      chapter = parseInt(m[1], 10) || 0;
      body = body.slice(m[0].length).trim();
    } else if (/^第[零一二三四五六七八九十百]+[章话]/.test(body)) {
      body = `${body}（章号未解析：中文序号不可锚定，请改用阿拉伯数字）`;
    }
    out.push({ chapter, text: body, line: i + 1 });
  }
  // 章序稳定排序（未锚=0 落最前，与后端 wenxueTimelineEntries 同语义）。
  return out.map((e, idx) => ({ e, idx }))
    .sort((a, b) => a.e.chapter !== b.e.chapter ? a.e.chapter - b.e.chapter : a.idx - b.idx)
    .map(({ e }) => e);
}

// ── 设定文档轻渲染：#/## 标题 + 列表 + 段落 ──

function DocRender({ text }: { text: string }) {
  const blocks: { kind: "h1" | "h2" | "li" | "p"; body: string }[] = [];
  for (const raw of text.split("\n")) {
    const t = raw.trim();
    if (!t) continue;
    if (t.startsWith("## ")) blocks.push({ kind: "h2", body: t.slice(3).trim() });
    else if (t.startsWith("# ")) blocks.push({ kind: "h1", body: t.slice(2).trim() });
    else if (t.startsWith("- ") || t.startsWith("* ") || t.startsWith("• ")) blocks.push({ kind: "li", body: t.slice(2).trim() });
    else blocks.push({ kind: "p", body: t });
  }
  return (
    <div className="flex flex-col gap-1.5" data-testid="wenxue-settings-doc">
      {blocks.map((b, i) => {
        if (b.kind === "h1") return <div key={i} className="mt-1 text-[16px] font-semibold text-fg">{b.body}</div>;
        if (b.kind === "h2") return <div key={i} className="mt-2 text-[13px] font-medium text-accent border-b pb-1" style={{ borderColor: "var(--md-sys-color-outline-variant)" }}>{b.body}</div>;
        if (b.kind === "li") return (
          <div key={i} className="flex gap-2 text-[12px] text-fg leading-relaxed pl-2">
            <span className="text-fg-faint shrink-0">·</span>
            <span>{b.body}</span>
          </div>
        );
        return <p key={i} className="text-[12px] text-fg leading-relaxed whitespace-pre-wrap">{b.body}</p>;
      })}
    </div>
  );
}

export function WenxueSettingsHub() {
  const [dirRel, setDirRel] = useState("");
  const [docs, setDocs] = useState<DocRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");
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

  const load = useCallback(async () => {
    const token = ++reloadRef.current;
    setLoading(true);
    setError("");
    try {
      const rootEntries = await app.ListDir("");
      let sDir = "";
      for (const cand of SETTINGS_DIRS) {
        if (rootEntries.some((e) => e.isDir && e.name === cand)) { sDir = cand; break; }
      }
      let tDir = "";
      for (const cand of TIMELINE_DIRS) {
        if (rootEntries.some((e) => e.isDir && e.name === cand)) { tDir = cand; break; }
      }
      setDirRel(sDir);
      const rows: DocRow[] = [];
      const readDocs = async (dir: string, group: DocRow["group"]) => {
        if (!dir) return;
        const entries = await app.ListDir(dir);
        for (const e of entries) {
          if (e.isDir || !/\.(md|markdown|txt)$/i.test(e.name)) continue;
          const rel = `${dir}/${e.name}`;
          const tf = await app.GaeaReadTextFile(rel);
          rows.push({
            group, path: rel,
            name: firstHeadingName(tf.text, e.name.replace(/\.(md|markdown|txt)$/i, "")),
            words: countWords(tf.text), excerpt: excerptOf(tf.text),
            mtimeMs: tf.mtimeMs, text: tf.text,
          });
        }
      };
      await readDocs(sDir, "settings");
      await readDocs(tDir, "timeline");
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
  // 换书后重载。
  useEffect(() => {
    const handler = () => { void load(); };
    window.addEventListener("wenxue:workspace-changed", handler);
    return () => window.removeEventListener("wenxue:workspace-changed", handler);
  }, [load]);

  // agent 落盘即见：filetree 通道变更路径落在设定/时间线目录下时自动重载。
  useEffect(() => {
    return subscribe("gaea-wenxue-filetree", (payload) => {
      const paths = (payload as { paths?: unknown } | null)?.paths;
      if (!Array.isArray(paths)) return;
      const hit = paths.some((p) => {
        if (typeof p !== "string") return false;
        const norm = p.replaceAll("\\", "/");
        return [...SETTINGS_DIRS, ...TIMELINE_DIRS].some((d) => norm.startsWith(`${d}/`));
      });
      if (hit) void load();
    });
  }, [load]);

  const selectedRow = useMemo(() => docs.find((d) => d.path === selected) ?? null, [docs, selected]);

  const openDoc = (row: DocRow) => {
    setSelected(row.path);
    setDraft(row.text);
    setBaseline(row.mtimeMs);
    setDirty(false);
    setEditing(false);
  };

  // 列表重载后：未脏且外部已更新 → 静默跟随最新（CAS 基线同步；脏草稿不动）。
  useEffect(() => {
    if (!selectedRow || dirty) return;
    if (selectedRow.mtimeMs !== baseline) {
      setDraft(selectedRow.text);
      setBaseline(selectedRow.mtimeMs);
    }
  }, [selectedRow, baseline, dirty]);

  const save = async () => {
    if (!selected) return;
    setSaving(true);
    try {
      await app.GaeaWriteWorkspaceFile(selected, draft, baseline);
      const tf = await app.GaeaReadTextFile(selected);
      setBaseline(tf.mtimeMs);
      setDraft(tf.text);
      setDirty(false);
      setDocs((rows) => rows.map((r) => (r.path === selected
        ? { ...r, text: tf.text, mtimeMs: tf.mtimeMs, words: countWords(tf.text), excerpt: excerptOf(tf.text) }
        : r)));
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

  const nameTaken = (rel: string): boolean => docs.some((d) => d.path.toLowerCase() === rel.toLowerCase());

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
      await app.GaeaWriteWorkspaceFile(rel, `# ${name}\n\n（待补——可在右侧「编辑」手写，或用「AI 生成」让 agent 基于现有章节/角色卡/时间线起草。）\n`, 0);
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

  // AI 生成：Composer 注入生成指令（agent 经 write_file 落设定/<名>.md；落盘后
  // filetree 通道让本页自动出现该文档）。
  const requestGen = (file: string, brief: string) => {
    useComposerInsertStore.getState().requestText(
      `请为这部小说生成设定文档并保存到工作空间相对路径 ${file}（用 write_file；` +
      `若该文件已存在，先 read_file 再在其基础上修订扩充）。内容要求：${brief}。` +
      `生成前先用 wenxue_chapter_list 与 wenxue_character_list 摸清已有章节与角色，` +
      `必要时参考 wenxue_timeline_read，保证设定与既有正文/角色一致；` +
      `文档用 Markdown（首行 # 标题，## 小节组织），与「时间线」目录中的事实不冲突。`,
    );
    setCreating(false);
    message.success("已写入对话输入框——发送后 agent 生成落盘，本页自动出现该文档");
  };

  const askAiImprove = (row: DocRow) => {
    useComposerInsertStore.getState().requestText(
      `请阅读设定文档 @${row.path}，在保持既有事实一致的前提下补全缺失小节、修订含糊表述；` +
      `与「时间线」目录中的事件冲突处以时间线为准；改完用 write_file 保存（先 read_file 再整体重写）。`,
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
          if (selected === row.path) { setSelected(""); setDraft(""); setDirty(false); setEditing(false); }
          await load();
          message.success("已删除");
        } catch (err) {
          message.error(err instanceof Error ? err.message : "删除失败");
        }
      },
    });
  };

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return docs;
    return docs.filter((d) => `${d.name} ${d.excerpt} ${d.path}`.toLowerCase().includes(q));
  }, [docs, query]);

  const settingsDocs = filtered.filter((d) => d.group === "settings");
  const timelineDocs = filtered.filter((d) => d.group === "timeline");
  const parsedTimeline = useMemo(() => {
    if (!selectedRow || selectedRow.group !== "timeline" || editing) return null;
    return parseTimeline(selectedRow.text);
  }, [selectedRow, editing]);

  const docListRow = (d: DocRow) => (
    <div
      key={d.path}
      role="button"
      tabIndex={0}
      data-testid={`wenxue-settings-doc-${d.name}`}
      className={`rounded-md border px-2.5 py-2 cursor-pointer transition-colors ${selected === d.path ? "border-accent" : "hover:border-accent/40"}`}
      style={{ background: "var(--md-sys-color-surface-container)", borderColor: selected === d.path ? "var(--gaea-glow)" : "var(--md-sys-color-outline-variant)" }}
      onClick={() => openDoc(d)}
      onKeyDown={(e) => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); openDoc(d); } }}
    >
      <div className="flex items-center justify-between gap-2">
        <span className="text-[13px] font-medium text-fg truncate">{d.name}</span>
        <span className="text-[10px] text-fg-faint shrink-0">{d.words} 字</span>
      </div>
      <div className="mt-0.5 text-[11px] text-fg-dim leading-relaxed" style={{ display: "-webkit-box", WebkitLineClamp: 2, WebkitBoxOrient: "vertical", overflow: "hidden" }}>
        {d.excerpt || "（空文档）"}
      </div>
    </div>
  );

  return (
    <div className="flex h-full min-h-0" data-testid="wenxue-settings-hub">
      {/* ── 左栏 ── */}
      <div
        className="flex w-64 shrink-0 flex-col gap-2 border-r px-3 py-3"
        style={{ borderColor: "var(--md-sys-color-outline-variant)" }}
      >
        <div className="flex items-center gap-1.5 px-0.5 text-[11px] text-fg-dim">
          <ScrollText size={13} aria-hidden className="text-accent shrink-0" />
          <span>与 agent 的 write_file / 时间线工具同源</span>
        </div>
        <div className="flex items-center gap-1.5">
          <div className="relative flex-1 min-w-0">
            <Search size={12} aria-hidden className="absolute left-2 top-1/2 -translate-y-1/2 text-fg-faint" />
            <input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="搜索设定…"
              aria-label="搜索设定"
              className="w-full rounded-md py-1.5 pl-7 pr-2 text-[12px] outline-none"
              style={{ background: "var(--md-sys-color-surface-container)", border: "1px solid var(--md-sys-color-outline-variant)", color: "var(--md-sys-color-text)" }}
            />
          </div>
          <button type="button" className={HEAD_BTN} onClick={() => void load()} title="刷新（agent 侧落盘后同步）">
            <RefreshCw size={11} />
          </button>
        </div>
        <div className="flex items-center gap-1.5">
          <button
            type="button"
            className="flex flex-1 items-center justify-center gap-1 rounded-md px-2 py-1.5 text-[12px] bg-accent text-accent-fg cursor-pointer hover:opacity-90"
            onClick={openCreate}
            data-testid="wenxue-settings-new"
          >
            <Plus size={12} /> 新建文档
          </button>
          <button
            type="button"
            className="flex flex-1 items-center justify-center gap-1 rounded-md px-2 py-1.5 text-[12px] border cursor-pointer hover:opacity-80"
            style={{ borderColor: "var(--md-sys-color-outline-variant)", color: "var(--md-sys-color-text)" }}
            onClick={() => { setGenTab("preset"); setGenName(""); setCreating(true); }}
            data-testid="wenxue-settings-gen"
            title="AI 生成设定文档（世界观/力量体系/规则/势力/地理/自定义）"
          >
            <Sparkles size={12} /> AI 生成
          </button>
        </div>

        {error && (
          <div className="px-2 py-1.5 rounded-md border border-amber-500/30 bg-amber-500/5 text-amber-500 text-[11px]">{error}</div>
        )}
        <div className="min-h-0 flex-1 overflow-auto flex flex-col gap-2">
          {loading ? (
            <div className="py-8 text-center text-fg-faint text-[12px]">读取中…</div>
          ) : (
            <>
              <div className="flex items-center gap-1.5 px-0.5 text-[10px] text-fg-faint">
                <FileText size={11} aria-hidden /> 设定文档{settingsDocs.length > 0 ? `（${settingsDocs.length}）` : ""}
              </div>
              {settingsDocs.length === 0 && (
                <div className="px-1 text-[10px] text-fg-faint leading-relaxed">
                  尚无设定文档（工作空间「设定」目录为空）——点「AI 生成」让 agent 起草，或「新建文档」手写。
                </div>
              )}
              {settingsDocs.map(docListRow)}
              <div className="flex items-center gap-1.5 px-0.5 mt-1 text-[10px] text-fg-faint">
                <Clock size={11} aria-hidden /> 时间线{timelineDocs.length > 0 ? `（${timelineDocs.length}）` : ""}
              </div>
              {timelineDocs.length === 0 && (
                <div className="px-1 text-[10px] text-fg-faint leading-relaxed">
                  时间线目录为空——agent 写章后的回写动作（wenxue_timeline_append）会自动建立。
                </div>
              )}
              {timelineDocs.map(docListRow)}
            </>
          )}
        </div>
        <div className="text-[10px] text-fg-faint leading-relaxed px-0.5">
          agent 落盘（write_file / wenxue_timeline_append）后本列表自动刷新。
        </div>
      </div>

      {/* ── 右栏详情 ── */}
      <div className="min-w-0 flex-1 overflow-auto px-5 py-4">
        {!selectedRow ? (
          <div className="flex h-full flex-col items-center justify-center gap-2 text-fg-faint text-[12px]">
            <ScrollText size={28} aria-hidden className="opacity-40" />
            <span>左侧选择设定文档或时间线查看详情</span>
            <span className="text-[11px]">设定=工作空间「设定」目录 md 文档；时间线=「时间线」目录事件</span>
          </div>
        ) : (
          <div>
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <div className="flex items-center gap-2">
                  <span className="text-[17px] font-semibold text-fg truncate">{selectedRow.name}</span>
                  <span className="px-1 rounded bg-bg-soft text-[9px] text-fg-dim shrink-0">{selectedRow.group === "timeline" ? "时间线" : "设定文档"}</span>
                  <span className="text-[10px] text-fg-faint shrink-0">{selectedRow.words} 字</span>
                  {dirty && <span className="w-1.5 h-1.5 rounded-full bg-amber-400" title="未保存" />}
                </div>
                <div className="mt-0.5 text-[10px] text-fg-faint truncate">{selectedRow.path}</div>
              </div>
              <div className="flex items-center gap-1 shrink-0">
                {!editing && (
                  <button type="button" className={HEAD_BTN} onClick={() => { setDraft(selectedRow.text); setBaseline(selectedRow.mtimeMs); setDirty(false); setEditing(true); }}>
                    <Pencil size={11} /> 编辑
                  </button>
                )}
                {editing && (
                  <>
                    <button
                      type="button"
                      className="flex items-center gap-1 px-2 py-1 rounded bg-accent text-accent-fg text-[11px] cursor-pointer hover:opacity-90 disabled:opacity-50"
                      onClick={() => void save()}
                      disabled={!dirty || saving}
                    >
                      {saving ? "保存中…" : <><Check size={11} /> 保存（CAS）</>}
                    </button>
                    <button type="button" className={HEAD_BTN} onClick={() => { if (!dirty) { setEditing(false); } else { message.warning("有未保存改动——保存或刷新丢弃后再退出编辑"); } }}>
                      完成
                    </button>
                  </>
                )}
                <button type="button" className={HEAD_BTN} title="让 AI 在既有事实一致的前提下补全修订（写入输入框）" onClick={() => askAiImprove(selectedRow)}>
                  <Sparkles size={10} /> 让 AI 完善
                </button>
                <button type="button" className={HEAD_BTN + " hover:!text-red-400"} title="删除设定文档" onClick={() => remove(selectedRow)}>
                  <Trash2 size={10} />
                </button>
              </div>
            </div>

            {editing ? (
              <textarea
                value={draft}
                onChange={(e) => { setDraft(e.target.value); setDirty(true); }}
                spellCheck={false}
                aria-label={`编辑设定 ${selected}`}
                className="mt-3 w-full h-[calc(100vh-260px)] min-h-72 p-3 text-[12px] font-mono leading-relaxed bg-transparent outline-none resize-y rounded-md border"
                style={{ color: "var(--md-sys-color-text)", borderColor: "var(--md-sys-color-outline-variant)" }}
              />
            ) : selectedRow.group === "timeline" && parsedTimeline ? (
              <div className="mt-3 flex flex-col gap-1.5" data-testid="wenxue-settings-timeline">
                {parsedTimeline.length === 0 && (
                  <div className="text-[11px] text-fg-faint">（空时间线——agent 回写章后事件经 wenxue_timeline_append 自动追加）</div>
                )}
                {parsedTimeline.map((ev, i) => (
                  <div
                    key={i}
                    className="flex items-start gap-2 rounded-md border px-3 py-1.5"
                    style={{ borderColor: "var(--md-sys-color-outline-variant)", background: "var(--md-sys-color-surface-container)" }}
                  >
                    <span
                      className={`px-1.5 rounded text-[10px] shrink-0 mt-0.5 ${ev.chapter > 0 ? "text-accent" : "text-fg-faint"}`}
                      style={{ background: "var(--md-sys-color-surface-container-high, rgba(127,127,127,0.12))" }}
                    >
                      {ev.chapter > 0 ? `第${ev.chapter}章` : "未锚"}
                    </span>
                    <span className="text-[12px] text-fg leading-relaxed">{ev.text}</span>
                  </div>
                ))}
                <div className="mt-1 text-[10px] text-fg-faint">排序=章号（未锚事件列最前，与 agent 的 wenxue_timeline_read 同口径）。</div>
              </div>
            ) : (
              <div className="mt-3">
                <DocRender text={selectedRow.text} />
              </div>
            )}
          </div>
        )}
      </div>

      {/* ── 新建 / AI 生成 ── */}
      <Modal
        title="新建设定文档"
        open={creating}
        onCancel={() => setCreating(false)}
        footer={null}
        destroyOnClose
        width={560}
      >
        <Tabs
          activeKey={genTab}
          onChange={(k) => setGenTab(k)}
          items={[
            {
              key: "blank",
              label: "空白文档",
              children: (
                <div>
                  <input
                    autoFocus
                    value={newName}
                    onChange={(e) => setNewName(e.target.value)}
                    onKeyDown={(e) => { if (e.key === "Enter") void createBlank(); }}
                    placeholder="文档名（如：世界观）"
                    aria-label="文档名"
                    className="w-full rounded-md px-2.5 py-1.5 text-[13px] outline-none"
                    style={{ background: "var(--md-sys-color-surface-container)", border: "1px solid var(--md-sys-color-outline-variant)", color: "var(--md-sys-color-text)" }}
                  />
                  <div className="mt-2 text-[11px] text-fg-faint">
                    将在「{dirRel || SETTINGS_DIRS[0]}/」下创建 &lt;文档名&gt;.md。
                  </div>
                  <button
                    type="button"
                    className="mt-3 w-full flex items-center justify-center gap-1 rounded-md px-2 py-1.5 text-[12px] bg-accent text-accent-fg cursor-pointer hover:opacity-90 disabled:opacity-50"
                    onClick={() => void createBlank()}
                    disabled={!newName.trim()}
                  >
                    <Plus size={12} /> 创建
                  </button>
                </div>
              ),
            },
            {
              key: "preset",
              label: <span className="flex items-center gap-1"><Sparkles size={12} /> AI 生成</span>,
              children: (
                <div>
                  <div className="flex flex-col gap-1.5">
                    {GEN_PRESETS.map((p) => (
                      <button
                        key={p.key}
                        type="button"
                        data-testid={`wenxue-settings-gen-${p.key}`}
                        className="rounded-md border px-2.5 py-2 text-left cursor-pointer hover:border-accent/60 transition-colors"
                        style={{ borderColor: "var(--md-sys-color-outline-variant)", background: "var(--md-sys-color-surface-container)" }}
                        onClick={() => requestGen(p.file, p.brief)}
                      >
                        <div className="flex items-center gap-2">
                          <Sparkles size={12} aria-hidden className="text-accent shrink-0" />
                          <span className="text-[12px] font-medium text-fg">{p.label}</span>
                          <span className="text-[10px] text-fg-faint">{p.file}</span>
                        </div>
                        <div className="mt-0.5 text-[10px] text-fg-dim leading-relaxed">{p.brief}</div>
                      </button>
                    ))}
                  </div>
                  <div className="mt-2 flex items-center gap-1.5">
                    <input
                      value={genName}
                      onChange={(e) => setGenName(e.target.value)}
                      placeholder="自定义文档名（如：货币与商路）"
                      aria-label="自定义文档名"
                      className="flex-1 min-w-0 rounded-md px-2.5 py-1.5 text-[12px] outline-none"
                      style={{ background: "var(--md-sys-color-surface-container)", border: "1px solid var(--md-sys-color-outline-variant)", color: "var(--md-sys-color-text)" }}
                    />
                    <button
                      type="button"
                      className="flex items-center gap-1 rounded-md px-2 py-1.5 text-[11px] border cursor-pointer hover:opacity-80 disabled:opacity-50"
                      style={{ borderColor: "var(--md-sys-color-outline-variant)", color: "var(--md-sys-color-text)" }}
                      onClick={() => requestGen(`${dirRel || SETTINGS_DIRS[0]}/${genName.trim()}.md`, `围绕「${genName.trim()}」的成体系设定`)}
                      disabled={!genName.trim()}
                      data-testid="wenxue-settings-gen-custom"
                    >
                      <Download size={11} /> 生成
                    </button>
                  </div>
                  <div className="mt-2 rounded-md px-2.5 py-1.5 text-[10px] text-fg-faint leading-relaxed" style={{ background: "var(--md-sys-color-surface-container)" }}>
                    生成指令写入对话输入框，agent 先摸清已有章节/角色/时间线再起草，经 write_file 落
                    「设定」目录——落盘后本页自动出现该文档；生成内容随时可在此编辑修订。
                  </div>
                </div>
              ),
            },
          ]}
        />
      </Modal>
    </div>
  );
}

const HEAD_BTN =
  "flex items-center gap-1 px-1.5 py-0.5 border-0 rounded bg-transparent text-fg-dim text-[10px] cursor-pointer hover:bg-bg-soft";
