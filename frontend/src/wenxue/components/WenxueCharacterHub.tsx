import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { message, Modal, Tabs } from "antd";
import {
  Check, Download, Pencil, Plus, RefreshCw, Search, Sparkles,
  Trash2, Users,
} from "../icons";
import { app } from "../lib/bridge";
import { subscribe } from "../../events";
import { useComposerInsertStore } from "../lib/store";
import { WenxueHubEmpty, WenxueHubFrame } from "./WenxueHubFrame";
import {
  countWords, firstParagraph, inferKeywordValue, initialsOf,
} from "./wenxueHubUtils";

// WenxueCharacterHub — 图文板块「角色」标签页。
// 数据面不变：小说工作空间的角色卡 md 文件（角色/|characters/|人物/，一角色
// 一卡）。UI 从工具式 master-detail 升级为人物档案：演员表索引、角色速写、
// 结构化正文、当前状态与资料完整度并列呈现。agent 回写仍通过 filetree
// 通道自动刷新；编辑器继续使用 CAS 裸文本，保证 md 契约与并发安全。

interface CharacterCardRow {
  path: string;
  name: string;
  words: number;
  excerpt: string;
  mtimeMs: number;
  text: string;
}

interface LibChar {
  id: string;
  name: string;
  kind: string;
  gender?: string;
  age?: string;
  tags?: string[];
  roleType?: string;
  personality?: string;
  background?: string;
  appearance?: string;
  figure?: string;
  motivation?: string;
  arc?: string;
  status?: string;
  notes?: string;
  dialogueSamples?: string[];
}

interface CharSection {
  title: string;
  body: string;
}

interface ParsedCard {
  name: string;
  intro: string;
  sections: CharSection[];
  structured: boolean;
}

interface CardAnalysis {
  parsed: ParsedCard;
  filled: Set<string>;
  completeCount: number;
  role: string;
  status: string;
  facts: Array<{ label: string; value: string }>;
}

const DIR_CANDIDATES = ["角色", "characters", "人物"];
const CANON_SECTIONS = ["设定", "外貌", "性格", "口头禅", "关系", "当前状态"];
const STORY_SECTIONS = ["设定", "外貌", "性格", "口头禅"];

const ROLE_TYPE_ZH: Record<string, string> = {
  protagonist: "主角", antagonist: "反派", supporting: "配角", minor: "龙套",
};
const STATUS_ZH: Record<string, string> = {
  Alive: "存活", Dead: "死亡", Missing: "失踪", Transformed: "转变",
};

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

function parseCard(text: string, fallbackName: string): ParsedCard {
  const name = firstHeadingName(text, fallbackName);
  const sections: CharSection[] = [];
  let intro = "";
  let current: CharSection | null = null;
  for (const line of text.split("\n")) {
    const t = line.trim();
    if (t.startsWith("# ")) continue;
    if (t.startsWith("## ")) {
      current = { title: t.slice(3).trim(), body: "" };
      sections.push(current);
      continue;
    }
    if (current) current.body += `${current.body ? "\n" : ""}${line}`;
    else if (t) intro += `${intro ? "\n" : ""}${t}`;
  }
  return {
    name,
    intro: intro.trim(),
    sections: sections.map((section) => ({ ...section, body: section.body.trim() })),
    structured: sections.length > 0,
  };
}

function sectionBody(parsed: ParsedCard, title: string): string {
  return parsed.sections.find((section) => section.title === title)?.body ?? "";
}

function keyFactsOf(body: string): Array<{ label: string; value: string }> {
  const facts: Array<{ label: string; value: string }> = [];
  for (const line of body.split("\n")) {
    const t = line.trim().replace(/^[-*•]\s*/, "");
    const m = /^([^：:\n]{1,12})[：:]\s*(.+)$/.exec(t);
    if (!m) continue;
    const label = m[1].trim();
    const value = m[2].trim();
    if (!label || !value) continue;
    facts.push({ label, value });
    if (facts.length >= 6) break;
  }
  return facts;
}

function analyzeCard(row: CharacterCardRow): CardAnalysis {
  const parsed = parseCard(row.text, row.name);
  const filled = new Set(
    parsed.sections
      .filter((section) => section.body.trim())
      .map((section) => section.title),
  );
  const setting = sectionBody(parsed, "设定");
  const status = sectionBody(parsed, "当前状态");
  const explicitRole = inferKeywordValue(setting, ["定位", "身份", "角色"]);
  const explicitStatus = inferKeywordValue(status, ["状态", "近况"]) || firstParagraph(status, 42);
  const facts = keyFactsOf(setting).slice(0, 4);
  return {
    parsed,
    filled,
    completeCount: CANON_SECTIONS.filter((title) => filled.has(title)).length,
    role: explicitRole || "人物档案",
    status: explicitStatus || "状态未记",
    facts,
  };
}

function cardFromLibraryChar(c: LibChar): string {
  const roleTypeZh = c.roleType ? ROLE_TYPE_ZH[c.roleType] ?? c.roleType : "";
  const statusZh = c.status ? STATUS_ZH[c.status] ?? c.status : "";
  const bodyOf = (lines: string[], fallback: string): string => {
    const filled = lines.map((line) => line.trim()).filter(Boolean);
    return filled.length ? filled.join("\n") : fallback;
  };
  const setting: string[] = [];
  const meta = [roleTypeZh, c.gender, c.age].filter(Boolean).join(" · ");
  if (meta) setting.push(`定位：${meta}`);
  if (c.tags?.length) setting.push(`标签：${c.tags.join("、")}`);
  if (c.background) setting.push(c.background);
  if (c.motivation) setting.push(`动机：${c.motivation}`);
  const personality: string[] = [];
  if (c.personality) personality.push(c.personality);
  if (c.arc) personality.push(`弧线：${c.arc}`);
  const status: string[] = [];
  if (statusZh) status.push(`状态：${statusZh}`);
  if (c.notes) status.push(c.notes);
  const samples = (c.dialogueSamples ?? []).map((sample) => `- ${sample}`).filter((sample) => sample !== "- ");
  const section = (title: string, body: string): string => `## ${title}\n\n${body}`;
  return [
    `# ${c.name}`,
    section("设定", bodyOf(setting, "（待补）")),
    section("外貌", bodyOf([c.appearance ?? "", c.figure ?? ""], "（待补）")),
    section("性格", bodyOf(personality, "（待补）")),
    section("口头禅", samples.length ? samples.join("\n") : "（待补）"),
    section("关系", "（留待书中发展）"),
    section("当前状态", bodyOf(status, "（待补）")),
  ].join("\n\n") + "\n";
}

function libSummary(c: LibChar): string {
  const base = c.background || c.personality || c.appearance || "";
  const chars = [...base.replace(/\s+/g, " ").trim()];
  return chars.length > 60 ? `${chars.slice(0, 60).join("")}…` : chars.join("");
}

function SectionCard({
  title,
  body,
  index,
  onAdd,
}: {
  title: string;
  body: string;
  index: number;
  onAdd?: () => void;
}) {
  const empty = !body.trim();
  return (
    <section className={`wx-hub-paper wx-character-section${empty ? " is-empty" : ""}`}>
      <div className="wx-character-section__head">
        <h2>{title}</h2>
        <span className="wx-character-section__index">{String(index + 1).padStart(2, "0")}</span>
      </div>
      {empty ? (
        <>
          <span className="sr-only">{title} · 未填写</span>
          <div className="wx-character-section__body">这一节还空着。角色不必一次写满，但关键选择要能在这里找到依据。</div>
          {onAdd && <button type="button" className="wx-character-section__add" onClick={onAdd}>补写这一节</button>}
        </>
      ) : (
        <div className="wx-character-section__body">{body}</div>
      )}
    </section>
  );
}

export function WenxueCharacterHub() {
  const [dirRel, setDirRel] = useState("");
  const [cards, setCards] = useState<CharacterCardRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState<"all" | "ready" | "todo">("all");
  const [selected, setSelected] = useState<string>("");
  const [draft, setDraft] = useState("");
  const [baseline, setBaseline] = useState(0);
  const [dirty, setDirty] = useState(false);
  const [editing, setEditing] = useState(false);
  const [saving, setSaving] = useState(false);
  const [creating, setCreating] = useState(false);
  const [newName, setNewName] = useState("");
  const [libTab, setLibTab] = useState("blank");
  const [libQuery, setLibQuery] = useState("");
  const [libItems, setLibItems] = useState<LibChar[]>([]);
  const [libTotal, setLibTotal] = useState(0);
  const [libPage, setLibPage] = useState(1);
  const [libLoading, setLibLoading] = useState(false);
  const [libPicked, setLibPicked] = useState<LibChar | null>(null);
  const reloadRef = useRef(0);
  const editorRef = useRef<HTMLTextAreaElement>(null);
  const detailRef = useRef<HTMLElement>(null);

  const load = useCallback(async () => {
    const token = ++reloadRef.current;
    setLoading(true);
    setError("");
    try {
      const rootEntries = await app.ListDir("");
      let dir = "";
      for (const candidate of DIR_CANDIDATES) {
        if (rootEntries.some((entry) => entry.isDir && entry.name === candidate)) {
          dir = candidate;
          break;
        }
      }
      setDirRel(dir);
      if (!dir) {
        setCards([]);
        return;
      }
      const entries = await app.ListDir(dir);
      const rows: CharacterCardRow[] = [];
      for (const entry of entries) {
        if (entry.isDir || !/\.(md|markdown|txt)$/i.test(entry.name)) continue;
        const rel = `${dir}/${entry.name}`;
        const file = await app.GaeaReadTextFile(rel);
        rows.push({
          path: rel,
          name: firstHeadingName(file.text, entry.name.replace(/\.(md|markdown|txt)$/i, "")),
          words: countWords(file.text),
          excerpt: excerptOf(file.text),
          mtimeMs: file.mtimeMs,
          text: file.text,
        });
      }
      rows.sort((a, b) => a.name.localeCompare(b.name, "zh-Hans-CN"));
      if (token !== reloadRef.current) return;
      setCards(rows);
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
      const hit = paths.some((p) => {
        if (typeof p !== "string") return false;
        const norm = p.replaceAll("\\", "/");
        return DIR_CANDIDATES.some((dir) => norm === dir || norm.startsWith(`${dir}/`));
      });
      if (hit) void load();
    });
  }, [load]);

  const selectedRow = useMemo(
    () => cards.find((card) => card.path === selected) ?? null,
    [cards, selected],
  );

  const openCard = (row: CharacterCardRow) => {
    if (detailRef.current) detailRef.current.scrollTop = 0;
    setSelected(row.path);
    setDraft(row.text);
    setBaseline(row.mtimeMs);
    setDirty(false);
    setEditing(false);
  };

  useEffect(() => {
    if (cards.length === 0) return;
    if (selectedRow) return;
    openCard(cards[0]);
  }, [cards, selectedRow]);

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
      setCards((rows) => rows.map((row) => (row.path === selected
        ? {
            ...row,
            text: file.text,
            mtimeMs: file.mtimeMs,
            words: countWords(file.text),
            excerpt: excerptOf(file.text),
          }
        : row)));
      message.success("角色卡已保存（agent 的 wenxue_character_* 工具所见即所得）");
    } catch (err) {
      message.error(err instanceof Error ? err.message : "保存失败（可能已被外部修改，请刷新后重试）");
    } finally {
      setSaving(false);
    }
  };

  const openCreate = () => {
    setNewName("");
    setLibQuery("");
    setLibItems([]);
    setLibTotal(0);
    setLibPage(1);
    setLibPicked(null);
    setLibTab("blank");
    setCreating(true);
  };

  const searchLibrary = useCallback(async (keyword: string, page: number) => {
    setLibLoading(true);
    try {
      // 只读联动红线：仅 CharacterList 查询，不调用任何写向角色库绑定。
      const result = await app.CharacterList(keyword, "", false, page, 50);
      const items = Array.isArray(result?.items) ? (result.items as LibChar[]) : [];
      const total = typeof result?.total === "number" ? result.total : 0;
      setLibItems((prev) => (page > 1 ? [...prev, ...items] : items));
      setLibTotal(total);
      setLibPage(page);
    } catch (err) {
      message.error(err instanceof Error ? err.message : "角色库读取失败");
    } finally {
      setLibLoading(false);
    }
  }, []);

  const nameTaken = (name: string): boolean =>
    cards.some((card) => card.name === name)
    || cards.some((card) =>
      card.path.toLowerCase() === `${dirRel || DIR_CANDIDATES[0]}/${name}.md`.toLowerCase());

  const createBlank = async () => {
    const name = newName.trim();
    if (!name) return;
    if (nameTaken(name)) {
      message.error(`同名角色卡已存在（${name}）`);
      return;
    }
    const dir = dirRel || DIR_CANDIDATES[0];
    const rel = `${dir}/${name}.md`;
    try {
      await app.GaeaWriteWorkspaceFile(
        rel,
        `# ${name}\n\n## 设定\n\n## 外貌\n\n## 性格\n\n## 口头禅\n\n## 关系\n\n## 当前状态\n`,
        0,
      );
      message.success("角色卡已创建");
      setCreating(false);
      setNewName("");
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

  const createFromLibrary = async () => {
    if (!libPicked) return;
    if (nameTaken(libPicked.name)) {
      message.error(`同名角色卡已存在（${libPicked.name}）——导入前请先处理工作空间同名卡`);
      return;
    }
    const dir = dirRel || DIR_CANDIDATES[0];
    const rel = `${dir}/${libPicked.name}.md`;
    try {
      await app.GaeaWriteWorkspaceFile(rel, cardFromLibraryChar(libPicked), 0);
      message.success(`已从角色库导入「${libPicked.name}」为工作空间卡（库卡不受影响）`);
      setCreating(false);
      setLibPicked(null);
      await load();
      const fresh = await app.GaeaReadTextFile(rel);
      setSelected(rel);
      setDraft(fresh.text);
      setBaseline(fresh.mtimeMs);
      setDirty(false);
      setEditing(false);
    } catch (err) {
      message.error(err instanceof Error ? err.message : "导入失败");
    }
  };

  const remove = (row: CharacterCardRow) => {
    Modal.confirm({
      title: `删除角色卡「${row.name}」？`,
      content: `${row.path} 将被删除，不可恢复。agent 侧引用该卡的章节不会自动改写。`,
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

  const askAi = (row: CharacterCardRow) => {
    useComposerInsertStore.getState().requestText(
      `请阅读角色卡 @${row.path}，按「设定 / 外貌 / 性格 / 口头禅 / 关系 / 当前状态」六节补全缺失小节；`
      + `「当前状态」必须与最新章节一致；写完用 wenxue_character_write 保存（overwrite=true）。`,
    );
  };

  const analyses = useMemo(
    () => new Map(cards.map((card) => [card.path, analyzeCard(card)])),
    [cards],
  );
  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return cards.filter((card) => {
      const analysis = analyses.get(card.path);
      const complete = (analysis?.completeCount ?? 0) >= 5;
      if (filter === "ready" && !complete) return false;
      if (filter === "todo" && complete) return false;
      if (!q) return true;
      return `${card.name} ${card.excerpt} ${card.path} ${analysis?.role ?? ""}`
        .toLowerCase()
        .includes(q);
    });
  }, [analyses, cards, filter, query]);

  const selectedAnalysis = useMemo(
    () => (selectedRow ? analyses.get(selectedRow.path) ?? analyzeCard(selectedRow) : null),
    [analyses, selectedRow],
  );
  const readyCount = useMemo(
    () => cards.filter((card) => (analyses.get(card.path)?.completeCount ?? 0) >= 5).length,
    [analyses, cards],
  );
  const parsed = selectedAnalysis?.parsed ?? null;

  const orderedSections = useMemo(() => {
    if (!parsed) return [] as CharSection[];
    const known = STORY_SECTIONS.map((title) => ({
      title,
      body: sectionBody(parsed, title),
    }));
    const custom = parsed.sections.filter((section) => !CANON_SECTIONS.includes(section.title));
    return [...known, ...custom];
  }, [parsed]);

  const startEditing = () => {
    if (!selectedRow) return;
    setDraft(selectedRow.text);
    setBaseline(selectedRow.mtimeMs);
    setDirty(false);
    setEditing(true);
  };

  return (
    <WenxueHubFrame
      tone="characters"
      eyebrow="人物档案"
      title="角色人物志"
      description="把角色当作会随剧情变化的人，而不是一份静态字段清单。"
      metrics={[
        { label: "角色", value: cards.length, hint: cards.length ? "张工作空间卡" : "待建立" },
        { label: "完整", value: readyCount, hint: "六节完成五项以上" },
        { label: "待补", value: Math.max(0, cards.length - readyCount), hint: "可逐步完善" },
      ]}
      actions={[
        {
          key: "new-character",
          label: "新建角色",
          icon: Plus,
          onClick: openCreate,
          primary: true,
          testId: "wenxue-char-new",
        },
      ]}
    >
      <div className="wx-characters__workspace" data-testid="wenxue-character-hub">
        <aside className="wx-hub-sidebar" aria-label="角色索引">
          <div className="wx-hub-sidebar__head">
            <span className="wx-hub-sidebar__title"><Users size={12} className="mr-1 inline" aria-hidden />演员表</span>
            <span className="wx-hub-sidebar__count">{filtered.length}/{cards.length}</span>
          </div>
          <div className="wx-hub-search">
            <Search size={12} aria-hidden />
            <input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="搜索姓名、定位、摘要…"
              aria-label="搜索角色"
            />
            <button
              type="button"
              className="wx-hub-search__action"
              onClick={() => void load()}
              title="刷新（agent 侧回写后同步）"
              aria-label="刷新角色卡"
            >
              <RefreshCw size={11} />
            </button>
          </div>
          <div className="wx-hub-chips" aria-label="角色筛选">
            {([
              ["all", `全部 ${cards.length}`],
              ["ready", `已完善 ${readyCount}`],
              ["todo", `待补 ${Math.max(0, cards.length - readyCount)}`],
            ] as const).map(([key, label]) => (
              <button
                key={key}
                type="button"
                className="wx-hub-chip"
                aria-pressed={filter === key}
                onClick={() => setFilter(key)}
              >
                {label}
              </button>
            ))}
          </div>

          {error && (
            <div className="mx-3 mb-2 rounded-md border border-warning/30 bg-warning/5 px-2 py-1.5 text-[10px] text-warning" role="alert">
              {error}
            </div>
          )}

          <div className="wx-hub-list">
            {loading ? (
              <div className="py-10 text-center text-[11px] text-fg-faint">正在整理演员表…</div>
            ) : cards.length === 0 ? (
              <div className="px-3 py-8 text-center text-[10.5px] leading-relaxed text-fg-faint">
                尚无角色卡。新建第一张，或从角色库只读导入一个已有角色。
              </div>
            ) : filtered.length === 0 ? (
              <div className="px-3 py-8 text-center text-[10.5px] leading-relaxed text-fg-faint">
                当前筛选没有角色。换一个关键词或切回「全部」。
              </div>
            ) : (
              filtered.map((card) => {
                const analysis = analyses.get(card.path) ?? analyzeCard(card);
                const pct = Math.round((analysis.completeCount / CANON_SECTIONS.length) * 100);
                return (
                  <button
                    key={card.path}
                    type="button"
                    role="option"
                    aria-selected={selected === card.path}
                    data-testid={`wenxue-char-card-${card.name}`}
                    className="wx-hub-list-row"
                    onClick={() => openCard(card)}
                  >
                    <span className="wx-hub-monogram" aria-hidden>{initialsOf(card.name)}</span>
                    <span className="wx-character-row__copy">
                      <span className="wx-character-row__top">
                        <span className="wx-character-row__name">{card.name}</span>
                        <span className="wx-character-row__meta">{card.words} 字</span>
                      </span>
                      <span className="mt-1 block text-[9.5px] text-fg-faint">{analysis.role} · {analysis.status}</span>
                      <span className="wx-character-row__excerpt">{card.excerpt || "（空卡，先写下角色最重要的矛盾）"}</span>
                      <span className="wx-character-row__progress">
                        <span className="wx-character-row__track">
                          <span className="wx-character-row__fill" style={{ width: `${pct}%` }} />
                        </span>
                        <span className="wx-character-row__pct">{analysis.completeCount}/6</span>
                      </span>
                    </span>
                  </button>
                );
              })
            )}
          </div>
          <div className="wx-hub-sidebar__foot">
            与 agent 的 wenxue_character_* 工具同源。agent 回写（wenxue_character_write）后，演员表会自动刷新。
          </div>
        </aside>

        <main key={selected} className="wx-character-detail" ref={detailRef}>
          {!selectedRow || !parsed || !selectedAnalysis ? (
            <WenxueHubEmpty
              icon={Users}
              title={loading ? "正在整理人物志" : "先建立第一张人物卡"}
              description="角色卡保存在工作空间「角色」目录，每张卡是一份可编辑的 Markdown。它和 agent 使用的是同一份事实。"
              action={(
                <button type="button" className="wx-hub-action is-primary" onClick={openCreate}>
                  <Plus size={13} aria-hidden />
                  <span>新建角色</span>
                </button>
              )}
            />
          ) : (
            <div className="wx-character-dossier">
              <section className="wx-hub-paper wx-character-hero">
                <div className="wx-character-portrait" aria-hidden>
                  <strong>{initialsOf(parsed.name)}</strong>
                </div>
                <div className="wx-character-identity">
                  <div className="wx-character-kicker">CHARACTER DOSSIER · {selectedRow.path}</div>
                  <h1 className="wx-character-name">{parsed.name}</h1>
                  <p className="wx-character-subtitle">
                    {parsed.intro
                      || firstParagraph(sectionBody(parsed, "设定"))
                      || firstParagraph(sectionBody(parsed, "性格"))
                      || "这个角色还没有人物速写。先写下他最想要什么，又最怕失去什么。"}
                  </p>
                  <div className="wx-hub-meta-row mt-2">
                    <span className="wx-hub-badge is-accent">{selectedAnalysis.role}</span>
                    <span className="wx-hub-badge">{selectedAnalysis.status}</span>
                    <span className="wx-hub-badge">{selectedRow.words} 字</span>
                    <span className="wx-hub-badge">{parsed.structured ? "结构化角色卡" : "自由格式卡"}</span>
                    {dirty && <span className="wx-hub-badge" style={{ color: "var(--warn)" }}>有未保存修改</span>}
                  </div>
                </div>
                <div className="wx-character-hero__side">
                  <div className="wx-character-hero__actions">
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
                      title="让 AI 按六节结构补全此卡（写入输入框）"
                      onClick={() => askAi(selectedRow)}
                    >
                      <Sparkles size={10} aria-hidden /> 让 AI 完善
                    </button>
                    <button
                      type="button"
                      className="wx-hub-icon-btn is-danger"
                      title="删除角色卡"
                      aria-label={`删除角色卡 ${parsed.name}`}
                      onClick={() => remove(selectedRow)}
                    >
                      <Trash2 size={10} aria-hidden />
                    </button>
                  </div>
                  <div className="wx-character-completion">
                    <div className="wx-character-completion__label">
                      <span>资料完整度</span>
                      <strong>{Math.round((selectedAnalysis.completeCount / CANON_SECTIONS.length) * 100)}%</strong>
                    </div>
                    <div className="wx-character-completion__track">
                      <div
                        className="wx-character-completion__fill"
                        style={{ width: `${(selectedAnalysis.completeCount / CANON_SECTIONS.length) * 100}%` }}
                      />
                    </div>
                    <div className="wx-character-completion__hint">
                      {selectedAnalysis.completeCount >= 5 ? "关键档案已成形" : "补齐缺失小节会更稳定"}
                    </div>
                  </div>
                </div>
              </section>

              {editing ? (
                <section className="wx-hub-paper wx-character-editor">
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
                    aria-label={`编辑角色卡 ${selected}`}
                  />
                </section>
              ) : parsed.structured ? (
                <div className="wx-character-body" data-testid="wenxue-char-detail">
                  <div className="wx-character-story">
                    {orderedSections.map((section, index) => (
                      <SectionCard
                        key={section.title}
                        title={section.title}
                        body={section.body}
                        index={index}
                        onAdd={startEditing}
                      />
                    ))}
                  </div>

                  <aside className="wx-character-aside">
                    <section className="wx-hub-paper wx-character-facts" aria-label="角色速览">
                      {(selectedAnalysis.facts.length > 0
                        ? selectedAnalysis.facts
                        : [
                            { label: "定位", value: selectedAnalysis.role },
                            { label: "状态", value: selectedAnalysis.status },
                            { label: "角色卡", value: `${selectedAnalysis.completeCount}/6 节` },
                            { label: "初稿", value: selectedRow.words ? `${selectedRow.words} 字` : "待写" },
                          ]).map((fact) => (
                        <dl key={`${fact.label}-${fact.value}`} className="wx-character-fact">
                          <dt>{fact.label}</dt>
                          <dd>{fact.value}</dd>
                        </dl>
                      ))}
                    </section>

                    <section className="wx-hub-paper wx-character-status-card">
                      <h2>当前状态</h2>
                      {sectionBody(parsed, "当前状态") ? (
                        <p>{sectionBody(parsed, "当前状态")}</p>
                      ) : (
                        <>
                          <span className="sr-only">当前状态 · 未填写</span>
                          <p className="text-fg-faint">还没有记录角色最近的变化。</p>
                          <button type="button" className="wx-character-section__add" onClick={startEditing}>补写当前状态</button>
                        </>
                      )}
                    </section>

                    <section className="wx-hub-paper wx-character-status-card">
                      <h2>关系</h2>
                      {sectionBody(parsed, "关系") ? (
                        <p>{sectionBody(parsed, "关系")}</p>
                      ) : (
                        <>
                          <span className="sr-only">关系 · 未填写</span>
                          <p className="text-fg-faint">关系尚未记录。写下谁会影响他的选择，关系才会真正进入剧情。</p>
                          <button type="button" className="wx-character-section__add" onClick={startEditing}>补写关系</button>
                        </>
                      )}
                    </section>

                    <section className="wx-hub-paper wx-character-section-ledger">
                      <h2>档案目录</h2>
                      <div className="wx-character-section-ledger__grid">
                        {CANON_SECTIONS.map((title) => (
                          <span
                            key={title}
                            className={`wx-character-section-ledger__item${selectedAnalysis.filled.has(title) ? " is-filled" : ""}`}
                          >
                            <span className="wx-character-section-ledger__dot" aria-hidden />
                            {title}
                          </span>
                        ))}
                      </div>
                    </section>

                    <div className="px-1 text-[9.5px] leading-relaxed text-fg-faint">
                      角色卡与 agent 的人物工具共用同一文件。这里看到的「当前状态」就是续写时会读取的事实。
                    </div>
                  </aside>
                </div>
              ) : (
                <section className="wx-hub-paper wx-character-section" data-testid="wenxue-char-detail">
                  <div className="wx-character-section__head">
                    <h2>自由格式角色卡</h2>
                    <span className="wx-character-section__index">原文</span>
                  </div>
                  <div className="mb-2 text-[10px] text-fg-faint">
                    这张卡没有按六节结构组织。原文完整保留，可点「编辑」整理，或让 AI 按六节补全。
                  </div>
                  <pre className="whitespace-pre-wrap font-mono text-[12px] leading-relaxed text-fg">
                    {selectedRow.text}
                  </pre>
                </section>
              )}
            </div>
          )}
        </main>
      </div>

      <Modal
        title="新建角色"
        open={creating}
        onCancel={() => setCreating(false)}
        footer={null}
        destroyOnHidden
        width={580}
      >
        <Tabs
          activeKey={libTab}
          onChange={(key) => setLibTab(key)}
          items={[
            {
              key: "blank",
              label: "空白卡",
              children: (
                <div>
                  <input
                    autoFocus
                    value={newName}
                    onChange={(event) => setNewName(event.target.value)}
                    onKeyDown={(event) => {
                      if (event.key === "Enter") void createBlank();
                    }}
                    placeholder="角色名（如：林晚）"
                    aria-label="角色名"
                    className="w-full rounded-md px-2.5 py-2 text-[13px] outline-none"
                    style={{
                      background: "var(--md-sys-color-surface-container)",
                      border: "1px solid var(--md-sys-color-outline-variant)",
                      color: "var(--md-sys-color-text)",
                    }}
                  />
                  <div className="mt-2 text-[11px] text-fg-faint">
                    将在「{dirRel || DIR_CANDIDATES[0]}/」下创建 &lt;角色名&gt;.md，并预置六节骨架。
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
              key: "library",
              label: <span className="flex items-center gap-1"><Download size={12} /> 从角色库导入</span>,
              children: (
                <div>
                  <div className="flex items-center gap-1.5">
                    <div className="relative min-w-0 flex-1">
                      <Search size={12} aria-hidden className="absolute left-2 top-1/2 -translate-y-1/2 text-fg-faint" />
                      <input
                        value={libQuery}
                        onChange={(event) => setLibQuery(event.target.value)}
                        onKeyDown={(event) => {
                          if (event.key === "Enter") void searchLibrary(libQuery.trim(), 1);
                        }}
                        placeholder="搜索角色库（名字/设定）…"
                        aria-label="搜索角色库"
                        className="w-full rounded-md py-1.5 pl-7 pr-2 text-[12px] outline-none"
                        style={{
                          background: "var(--md-sys-color-surface-container)",
                          border: "1px solid var(--md-sys-color-outline-variant)",
                          color: "var(--md-sys-color-text)",
                        }}
                      />
                    </div>
                    <button type="button" className="wx-hub-icon-btn is-accent" onClick={() => void searchLibrary(libQuery.trim(), 1)}>
                      搜索
                    </button>
                  </div>
                  <div className="mt-2 flex min-h-24 max-h-64 flex-col gap-1.5 overflow-auto">
                    {libItems.length === 0 && !libLoading && (
                      <div className="py-6 text-center text-[11px] text-fg-faint">
                        {libTotal === 0 && libPage > 1 ? "没有更多了。" : "输入关键词搜索角色库，或直接搜索看全部。"}
                      </div>
                    )}
                    {libLoading && <div className="py-6 text-center text-[11px] text-fg-faint">读取角色库中…</div>}
                    {libItems.map((character) => (
                      <button
                        key={character.id}
                        type="button"
                        data-testid={`wenxue-lib-char-${character.name}`}
                        className={`wx-hub-list-row${libPicked?.id === character.id ? " is-selected" : ""}`}
                        style={libPicked?.id === character.id
                          ? { borderColor: "var(--wx-accent-line)", background: "var(--wx-accent-wash)" }
                          : undefined}
                        onClick={() => setLibPicked(character)}
                      >
                        <span className="wx-hub-monogram" aria-hidden>{initialsOf(character.name)}</span>
                        <span className="wx-settings-row__copy">
                          <span className="wx-settings-row__top">
                            <span className="wx-settings-row__name">{character.name}</span>
                            <span className="wx-settings-row__meta">
                              {character.roleType ? ROLE_TYPE_ZH[character.roleType] ?? character.roleType : character.kind}
                            </span>
                          </span>
                          <span className="wx-settings-row__excerpt">{libSummary(character) || "（无摘要）"}</span>
                        </span>
                      </button>
                    ))}
                    {libItems.length < libTotal && !libLoading && (
                      <button
                        type="button"
                        className="wx-hub-icon-btn justify-center"
                        onClick={() => void searchLibrary(libQuery.trim(), libPage + 1)}
                      >
                        加载更多（{libTotal - libItems.length}）
                      </button>
                    )}
                  </div>
                  <div className="mt-2 rounded-md px-2.5 py-2 text-[10px] leading-relaxed text-fg-faint" style={{ background: "var(--md-sys-color-surface-container)" }}>
                    只读联动：导入会把库卡字段映射为工作空间六节卡。
                    <span className="text-warning">不会写回、不登记角色库</span>；此后两边各自演进。
                  </div>
                  <button
                    type="button"
                    className="wx-hub-action is-primary mt-2.5 w-full justify-center"
                    onClick={() => void createFromLibrary()}
                    disabled={!libPicked}
                    data-testid="wenxue-lib-import"
                  >
                    <Download size={12} aria-hidden />
                    导入为工作空间卡{libPicked ? `（${libPicked.name}）` : ""}
                  </button>
                </div>
              ),
            },
          ]}
        />
      </Modal>
    </WenxueHubFrame>
  );
}
