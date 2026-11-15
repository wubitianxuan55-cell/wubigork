import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { message, Modal, Tabs } from "antd";
import { Check, Download, Pencil, Plus, RefreshCw, Search, Sparkles, Trash2, Users } from "../icons";
import { app } from "../lib/bridge";
import { subscribe } from "../../events";
import { useComposerInsertStore } from "../lib/store";

// WenxueCharacterHub — 图文板块「角色」标签页 v2（2026-11-16 重设计）。
// 数据面不变：小说工作空间的角色卡 md 文件（角色/|characters/|人物/ 目录，
// 一角色一卡），与 agent 的 wenxue_character_* 专属工具共享同一数据源——
// UI 编辑即 agent 所见，agent 回写（章节推进更新「当前状态」）UI 即见
// （gaea-wenxue-filetree 通道自动重载，不再依赖手动刷新）。
// v2 形态：左列表右详情（master-detail）+ 结构化阅读态（解析六节 md）+
// 编辑态保留 CAS 裸文本（保 md 契约与并发安全）。新建角色双通道：空白卡
// 或「从角色库导入」——只读联动 gaea 角色库（CharacterList 查询），字段
// 映射六节落工作空间卡；**红线=只读，不写回/不登记角色库**（工作空间卡
// 与库卡同源不同命，互不污染）。
// 卡片结构约定见内嵌技能 wenxue-workbench（首行一级标题=角色名，六节：
// 设定/外貌/性格/口头禅/关系/当前状态）。

interface CharacterCardRow {
  path: string;
  name: string;
  words: number;
  excerpt: string;
  mtimeMs: number;
  text: string;
}

// 角色库卡（CharacterList items 的消费字段全集；Record 转型面）。
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

const DIR_CANDIDATES = ["角色", "characters", "人物"];
const CANON_SECTIONS = ["设定", "外貌", "性格", "口头禅", "关系", "当前状态"];

const ROLE_TYPE_ZH: Record<string, string> = {
  protagonist: "主角", antagonist: "反派", supporting: "配角", minor: "龙套",
};
const STATUS_ZH: Record<string, string> = {
  Alive: "存活", Dead: "死亡", Missing: "失踪", Transformed: "转变",
};

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

function excerptOf(text: string, max = 90): string {
  const body = text
    .split("\n")
    .map((l) => l.trim())
    .filter((l) => l && !l.startsWith("#"));
  const joined = body.join(" ").replace(/\s+/g, " ").trim();
  const r = [...joined];
  return r.length > max ? r.slice(0, max).join("") + "…" : joined;
}

// ── 结构化解析（阅读态）：首行 # 名 + ## 节；自由格式卡 structured=false ──

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

function parseCard(text: string, fallbackName: string): ParsedCard {
  const name = firstHeadingName(text, fallbackName);
  const sections: CharSection[] = [];
  let intro = "";
  let cur: CharSection | null = null;
  for (const line of text.split("\n")) {
    const t = line.trim();
    if (t.startsWith("# ")) continue; // 首行标题（名）
    if (t.startsWith("## ")) {
      cur = { title: t.slice(3).trim(), body: "" };
      sections.push(cur);
      continue;
    }
    if (cur) cur.body += (cur.body ? "\n" : "") + line;
    else if (t) intro += (intro ? "\n" : "") + t;
  }
  return {
    name,
    intro: intro.trim(),
    sections: sections.map((s) => ({ ...s, body: s.body.trim() })),
    structured: sections.length > 0,
  };
}

// ── 角色库卡 → 工作空间六节卡（只读映射；仅非空字段落卡）──

function cardFromLibraryChar(c: LibChar): string {
  const roleTypeZh = c.roleType ? ROLE_TYPE_ZH[c.roleType] ?? c.roleType : "";
  const statusZh = c.status ? STATUS_ZH[c.status] ?? c.status : "";
  const bodyOf = (lines: string[], fallback: string): string => {
    const filled = lines.map((l) => l.trim()).filter(Boolean);
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
  const samples = (c.dialogueSamples ?? []).map((s) => `- ${s}`).filter((s) => s !== "- ");
  const sec = (title: string, body: string): string => `## ${title}\n\n${body}`;
  return [
    `# ${c.name}`,
    sec("设定", bodyOf(setting, "（待补）")),
    sec("外貌", bodyOf([c.appearance ?? "", c.figure ?? ""], "（待补）")),
    sec("性格", bodyOf(personality, "（待补）")),
    sec("口头禅", samples.length ? samples.join("\n") : "（待补）"),
    sec("关系", "（留待书中发展）"),
    sec("当前状态", bodyOf(status, "（待补）")),
  ].join("\n\n") + "\n";
}

function libSummary(c: LibChar): string {
  const base = c.background || c.personality || c.appearance || "";
  const r = [...base.replace(/\s+/g, " ").trim()];
  return r.length > 60 ? r.slice(0, 60).join("") + "…" : base.trim();
}

export function WenxueCharacterHub() {
  const [dirRel, setDirRel] = useState("");
  const [cards, setCards] = useState<CharacterCardRow[]>([]);
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
  // 角色库导入面（只读联动）。
  const [libTab, setLibTab] = useState("blank");
  const [libQuery, setLibQuery] = useState("");
  const [libItems, setLibItems] = useState<LibChar[]>([]);
  const [libTotal, setLibTotal] = useState(0);
  const [libPage, setLibPage] = useState(1);
  const [libLoading, setLibLoading] = useState(false);
  const [libPicked, setLibPicked] = useState<LibChar | null>(null);
  const reloadRef = useRef(0);

  const load = useCallback(async () => {
    const token = ++reloadRef.current;
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
        setCards([]);
        return;
      }
      const entries = await app.ListDir(dir);
      const rows: CharacterCardRow[] = [];
      for (const e of entries) {
        if (e.isDir || !/\.(md|markdown|txt)$/i.test(e.name)) continue;
        const rel = `${dir}/${e.name}`;
        const tf = await app.GaeaReadTextFile(rel);
        rows.push({
          path: rel,
          name: firstHeadingName(tf.text, e.name.replace(/\.(md|markdown|txt)$/i, "")),
          words: countWords(tf.text),
          excerpt: excerptOf(tf.text),
          mtimeMs: tf.mtimeMs,
          text: tf.text,
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
  // 切换工作空间（换书）后重载——文件树监视同款事件（wenxue:workspace-changed）。
  useEffect(() => {
    const handler = () => { void load(); };
    window.addEventListener("wenxue:workspace-changed", handler);
    return () => window.removeEventListener("wenxue:workspace-changed", handler);
  }, [load]);

  // agent 回写即见：文件树监视通道（gaea-wenxue-filetree）变更路径落在角色
  // 目录下时自动重载列表——wenxue_character_write 的落盘 UI 无需手动刷新。
  useEffect(() => {
    return subscribe("gaea-wenxue-filetree", (payload) => {
      const paths = (payload as { paths?: unknown } | null)?.paths;
      if (!Array.isArray(paths)) return;
      const hit = paths.some((p) => {
        if (typeof p !== "string") return false;
        const norm = p.replaceAll("\\", "/");
        return DIR_CANDIDATES.some((d) => norm === d || norm.startsWith(`${d}/`));
      });
      if (hit) void load();
    });
  }, [load]);

  const selectedRow = useMemo(
    () => cards.find((c) => c.path === selected) ?? null,
    [cards, selected],
  );

  const openCard = (row: CharacterCardRow) => {
    setSelected(row.path);
    setDraft(row.text);
    setBaseline(row.mtimeMs);
    setDirty(false);
    setEditing(false);
  };

  // 列表重载后：选中卡未脏且外部已更新（agent 回写）→ 静默跟随最新内容
  // （CAS 基线同步；脏草稿不动，保存时 CAS 兜底）。
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
      setCards((rows) => rows.map((r) => (r.path === selected
        ? { ...r, text: tf.text, mtimeMs: tf.mtimeMs, words: countWords(tf.text), excerpt: excerptOf(tf.text) }
        : r)));
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

  const searchLibrary = useCallback(async (kw: string, page: number) => {
    setLibLoading(true);
    try {
      // 只读联动红线：这里（以及整个面板）只允许 CharacterList 查询——
      // 不触任何写向/登记绑定（Save/Delete/Import*/Set*/Associate*/Sync）。
      const res = await app.CharacterList(kw, "", false, page, 50);
      const items = Array.isArray(res?.items) ? (res.items as LibChar[]) : [];
      const total = typeof res?.total === "number" ? res.total : 0;
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
    cards.some((c) => c.name === name) ||
    cards.some((c) => c.path.toLowerCase() === `${dirRel || DIR_CANDIDATES[0]}/${name}.md`.toLowerCase());

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
      await app.GaeaWriteWorkspaceFile(rel, `# ${name}\n\n## 设定\n\n## 外貌\n\n## 性格\n\n## 口头禅\n\n## 关系\n\n## 当前状态\n`, 0);
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
          if (selected === row.path) { setSelected(""); setDraft(""); setDirty(false); setEditing(false); }
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
      `请阅读角色卡 @${row.path}，按「设定 / 外貌 / 性格 / 口头禅 / 关系 / 当前状态」六节补全缺失小节；` +
      `「当前状态」必须与最新章节一致；写完用 wenxue_character_write 保存（overwrite=true）。`,
    );
  };

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return cards;
    return cards.filter((c) => `${c.name} ${c.excerpt} ${c.path}`.toLowerCase().includes(q));
  }, [cards, query]);

  const parsed = useMemo(
    () => (selectedRow ? parseCard(selectedRow.text, selectedRow.name) : null),
    [selectedRow],
  );

  return (
    <div className="flex h-full min-h-0" data-testid="wenxue-character-hub">
      {/* ── 左栏：搜索 + 角色卡列表 ── */}
      <div
        className="flex w-64 shrink-0 flex-col gap-2 border-r px-3 py-3"
        style={{ borderColor: "var(--md-sys-color-outline-variant)" }}
      >
        <div className="flex items-center gap-1.5 px-0.5 text-[11px] text-fg-dim">
          <Users size={13} aria-hidden className="text-accent shrink-0" />
          <span>与 agent 的 wenxue_character_* 工具同源</span>
        </div>
        <div className="flex items-center gap-1.5">
          <div className="relative flex-1 min-w-0">
            <Search size={12} aria-hidden className="absolute left-2 top-1/2 -translate-y-1/2 text-fg-faint" />
            <input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="搜索角色…"
              aria-label="搜索角色"
              className="w-full rounded-md py-1.5 pl-7 pr-2 text-[12px] outline-none"
              style={{ background: "var(--md-sys-color-surface-container)", border: "1px solid var(--md-sys-color-outline-variant)", color: "var(--md-sys-color-text)" }}
            />
          </div>
          <button type="button" className={HEAD_BTN} onClick={() => void load()} title="刷新（agent 侧回写后同步）">
            <RefreshCw size={11} />
          </button>
        </div>
        <button
          type="button"
          className="flex items-center justify-center gap-1 rounded-md px-2 py-1.5 text-[12px] bg-accent text-accent-fg cursor-pointer hover:opacity-90"
          onClick={openCreate}
          data-testid="wenxue-char-new"
        >
          <Plus size={12} /> 新建角色
        </button>

        {error && (
          <div className="px-2 py-1.5 rounded-md border border-amber-500/30 bg-amber-500/5 text-amber-500 text-[11px]">{error}</div>
        )}
        <div className="min-h-0 flex-1 overflow-auto flex flex-col gap-1.5">
          {loading ? (
            <div className="py-8 text-center text-fg-faint text-[12px]">读取中…</div>
          ) : filtered.length === 0 ? (
            <div className="py-8 px-2 text-center text-fg-faint text-[11px] leading-relaxed">
              {cards.length === 0
                ? "尚无角色卡（工作空间「角色」目录为空）——点「新建角色」建第一张，或让对话里的 agent 用 wenxue_character_write 生成。"
                : "无匹配角色。"}
            </div>
          ) : (
            filtered.map((c) => (
              <div
                key={c.path}
                role="button"
                tabIndex={0}
                data-testid={`wenxue-char-card-${c.name}`}
                className={`rounded-md border px-2.5 py-2 cursor-pointer transition-colors ${selected === c.path ? "border-accent" : "hover:border-accent/40"}`}
                style={{ background: "var(--md-sys-color-surface-container)", borderColor: selected === c.path ? "var(--gaea-glow)" : "var(--md-sys-color-outline-variant)" }}
                onClick={() => openCard(c)}
                onKeyDown={(e) => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); openCard(c); } }}
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="text-[13px] font-medium text-fg truncate">{c.name}</span>
                  <span className="text-[10px] text-fg-faint shrink-0">{c.words} 字</span>
                </div>
                <div className="mt-0.5 text-[11px] text-fg-dim leading-relaxed" style={{ display: "-webkit-box", WebkitLineClamp: 2, WebkitBoxOrient: "vertical", overflow: "hidden" }}>
                  {c.excerpt || "（空卡）"}
                </div>
              </div>
            ))
          )}
        </div>
        <div className="text-[10px] text-fg-faint leading-relaxed px-0.5">
          agent 回写（wenxue_character_write）后本列表自动刷新。
        </div>
      </div>

      {/* ── 右栏：详情（阅读态/编辑态）── */}
      <div className="min-w-0 flex-1 overflow-auto px-5 py-4">
        {!selectedRow || !parsed ? (
          <div className="flex h-full flex-col items-center justify-center gap-2 text-fg-faint text-[12px]">
            <Users size={28} aria-hidden className="opacity-40" />
            <span>左侧选择一张角色卡查看详情</span>
            <span className="text-[11px]">角色卡=工作空间「角色」目录的 md 文件，与 agent 工具所见一致</span>
          </div>
        ) : (
          <div>
            {/* 头部：名/字数/路径 + 动作 */}
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <div className="flex items-center gap-2">
                  <span className="text-[17px] font-semibold text-fg truncate">{parsed.name}</span>
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
                <button type="button" className={HEAD_BTN} title="让 AI 按六节结构补全此卡（写入输入框）" onClick={() => askAi(selectedRow)}>
                  <Sparkles size={10} /> 让 AI 完善
                </button>
                <button type="button" className={HEAD_BTN + " hover:!text-red-400"} title="删除角色卡" onClick={() => remove(selectedRow)}>
                  <Trash2 size={10} />
                </button>
              </div>
            </div>

            {/* 阅读态：结构化六节 / 编辑态：裸文本（CAS） */}
            {editing ? (
              <textarea
                value={draft}
                onChange={(e) => { setDraft(e.target.value); setDirty(true); }}
                spellCheck={false}
                aria-label={`编辑角色卡 ${selected}`}
                className="mt-3 w-full h-[calc(100vh-260px)] min-h-72 p-3 text-[12px] font-mono leading-relaxed bg-transparent outline-none resize-y rounded-md border"
                style={{ color: "var(--md-sys-color-text)", borderColor: "var(--md-sys-color-outline-variant)" }}
              />
            ) : parsed.structured ? (
              <div className="mt-3 flex flex-col gap-2.5" data-testid="wenxue-char-detail">
                {parsed.intro && <SectionBlock title="简介" body={parsed.intro} />}
                {parsed.sections.map((s) => (
                  <SectionBlock key={s.title} title={s.title} body={s.body} />
                ))}
                {CANON_SECTIONS.filter((t) => !parsed.sections.some((s) => s.title === t)).map((t) => (
                  <div
                    key={t}
                    className="rounded-md border border-dashed px-3 py-2 text-[11px] text-fg-faint"
                    style={{ borderColor: "var(--md-sys-color-outline-variant)" }}
                  >
                    {t} · 未填写（点「编辑」补全，或「让 AI 完善」）
                  </div>
                ))}
              </div>
            ) : (
              <div className="mt-3">
                <div className="mb-2 text-[10px] text-fg-faint">（自由格式卡——未按六节结构组织，原文如下）</div>
                <pre
                  className="whitespace-pre-wrap rounded-md border px-3 py-2.5 text-[12px] leading-relaxed font-mono"
                  style={{ color: "var(--md-sys-color-text)", borderColor: "var(--md-sys-color-outline-variant)", background: "var(--md-sys-color-surface-container)" }}
                >
                  {selectedRow.text}
                </pre>
              </div>
            )}
          </div>
        )}
      </div>

      {/* ── 新建角色：空白卡 / 从角色库导入（只读联动）── */}
      <Modal
        title="新建角色"
        open={creating}
        onCancel={() => setCreating(false)}
        footer={null}
        destroyOnClose
        width={560}
      >
        <Tabs
          activeKey={libTab}
          onChange={(k) => setLibTab(k)}
          items={[
            {
              key: "blank",
              label: "空白卡",
              children: (
                <div>
                  <input
                    autoFocus
                    value={newName}
                    onChange={(e) => setNewName(e.target.value)}
                    onKeyDown={(e) => { if (e.key === "Enter") void createBlank(); }}
                    placeholder="角色名（如：林晚）"
                    aria-label="角色名"
                    className="w-full rounded-md px-2.5 py-1.5 text-[13px] outline-none"
                    style={{ background: "var(--md-sys-color-surface-container)", border: "1px solid var(--md-sys-color-outline-variant)", color: "var(--md-sys-color-text)" }}
                  />
                  <div className="mt-2 text-[11px] text-fg-faint">
                    将在「{dirRel || DIR_CANDIDATES[0]}/」下创建 &lt;角色名&gt;.md，预置六节骨架（设定/外貌/性格/口头禅/关系/当前状态）。
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
              key: "library",
              label: <span className="flex items-center gap-1"><Download size={12} /> 从角色库导入</span>,
              children: (
                <div>
                  <div className="flex items-center gap-1.5">
                    <div className="relative flex-1 min-w-0">
                      <Search size={12} aria-hidden className="absolute left-2 top-1/2 -translate-y-1/2 text-fg-faint" />
                      <input
                        value={libQuery}
                        onChange={(e) => setLibQuery(e.target.value)}
                        onKeyDown={(e) => { if (e.key === "Enter") void searchLibrary(libQuery.trim(), 1); }}
                        placeholder="搜索角色库（名字/设定）…"
                        aria-label="搜索角色库"
                        className="w-full rounded-md py-1.5 pl-7 pr-2 text-[12px] outline-none"
                        style={{ background: "var(--md-sys-color-surface-container)", border: "1px solid var(--md-sys-color-outline-variant)", color: "var(--md-sys-color-text)" }}
                      />
                    </div>
                    <button type="button" className={HEAD_BTN + " !text-accent"} onClick={() => void searchLibrary(libQuery.trim(), 1)}>
                      搜索
                    </button>
                  </div>
                  <div className="mt-2 max-h-64 overflow-auto flex flex-col gap-1.5 min-h-24">
                    {libItems.length === 0 && !libLoading && (
                      <div className="py-6 text-center text-fg-faint text-[11px]">
                        {libTotal === 0 && libPage > 1 ? "没有更多了。" : "输入关键词搜索角色库，或直接点「搜索」看全部。"}
                      </div>
                    )}
                    {libLoading && <div className="py-6 text-center text-fg-faint text-[11px]">读取角色库中…</div>}
                    {libItems.map((c) => (
                      <div
                        key={c.id}
                        role="button"
                        tabIndex={0}
                        data-testid={`wenxue-lib-char-${c.name}`}
                        className={`rounded-md border px-2.5 py-1.5 cursor-pointer transition-colors ${libPicked?.id === c.id ? "border-accent" : "hover:border-accent/40"}`}
                        style={{ background: "var(--md-sys-color-surface-container)", borderColor: libPicked?.id === c.id ? "var(--gaea-glow)" : "var(--md-sys-color-outline-variant)" }}
                        onClick={() => setLibPicked(c)}
                        onKeyDown={(e) => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); setLibPicked(c); } }}
                      >
                        <div className="flex items-center gap-1.5">
                          <span className="text-[12px] font-medium text-fg">{c.name}</span>
                          {(c.roleType ? ROLE_TYPE_ZH[c.roleType] ?? c.roleType : c.kind) && (
                            <span className="px-1 rounded bg-bg-soft text-[9px] text-fg-dim">{c.roleType ? ROLE_TYPE_ZH[c.roleType] ?? c.roleType : c.kind}</span>
                          )}
                          {c.tags?.slice(0, 3).map((t) => (
                            <span key={t} className="px-1 rounded bg-bg-soft text-[9px] text-fg-faint">{t}</span>
                          ))}
                        </div>
                        <div className="mt-0.5 text-[10px] text-fg-dim truncate">{libSummary(c) || "（无摘要）"}</div>
                      </div>
                    ))}
                    {libItems.length < libTotal && !libLoading && (
                      <button type="button" className={HEAD_BTN + " justify-center"} onClick={() => void searchLibrary(libQuery.trim(), libPage + 1)}>
                        加载更多（{libTotal - libItems.length}）
                      </button>
                    )}
                  </div>
                  <div className="mt-2 rounded-md px-2.5 py-1.5 text-[10px] text-fg-faint leading-relaxed" style={{ background: "var(--md-sys-color-surface-container)" }}>
                    只读联动：导入=读取库卡字段映射为工作空间六节卡（设定←背景/动机，外貌←外貌/身材，性格←性格/弧线，口头禅←对白样本，当前状态←状态/备注）。
                    <span className="text-amber-500">不会写回、不登记角色库</span>；此后两边各自演进互不影响。
                  </div>
                  <button
                    type="button"
                    className="mt-2.5 w-full flex items-center justify-center gap-1 rounded-md px-2 py-1.5 text-[12px] bg-accent text-accent-fg cursor-pointer hover:opacity-90 disabled:opacity-50"
                    onClick={() => void createFromLibrary()}
                    disabled={!libPicked}
                    data-testid="wenxue-lib-import"
                  >
                    <Download size={12} /> 导入为工作空间卡{libPicked ? `（${libPicked.name}）` : ""}
                  </button>
                </div>
              ),
            },
          ]}
        />
      </Modal>
    </div>
  );
}

function SectionBlock({ title, body }: { title: string; body: string }) {
  const empty = !body.trim();
  return (
    <div
      className="rounded-md border px-3 py-2"
      style={{
        borderColor: "var(--md-sys-color-outline-variant)",
        background: empty ? "transparent" : "var(--md-sys-color-surface-container)",
        borderStyle: empty ? "dashed" : "solid",
      }}
    >
      <div className="text-[11px] font-medium text-fg-dim mb-1">{title}</div>
      {empty ? (
        <div className="text-[11px] text-fg-faint">（未填写）</div>
      ) : (
        <div className="text-[12px] text-fg leading-relaxed whitespace-pre-wrap">{body}</div>
      )}
    </div>
  );
}

const HEAD_BTN =
  "flex items-center gap-1 px-1.5 py-0.5 border-0 rounded bg-transparent text-fg-dim text-[10px] cursor-pointer hover:bg-bg-soft";
