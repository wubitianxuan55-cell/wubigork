import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { message, Modal } from "antd";
import { Pencil, Plus, RefreshCw, Sparkles, Trash2, Users } from "../icons";
import { app } from "../lib/bridge";
import { useComposerInsertStore } from "../lib/store";

// WenxueCharacterHub — 图文板块「角色」标签页（2026-11-15，移植自小说板块
// 角色页形态、数据层重接）。与小说板块 characters.json/角色库不同源：这里
// 的数据面=小说工作空间的角色卡 md 文件（角色/|characters/|人物/ 目录，
// 一角色一卡），与 agent 的 wenxue_character_* 专属工具共享同一数据源——
// UI 编辑即 agent 所见，agent 回写（章节推进更新「当前状态」）UI 刷新即见。
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

const DIR_CANDIDATES = ["角色", "characters", "人物"];

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
  const [saving, setSaving] = useState(false);
  const [creating, setCreating] = useState(false);
  const [newName, setNewName] = useState("");
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

  const openCard = (row: CharacterCardRow) => {
    setSelected(row.path);
    setDraft(row.text);
    setBaseline(row.mtimeMs);
    setDirty(false);
  };

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

  const create = async () => {
    const name = newName.trim();
    if (!name) return;
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
    } catch (err) {
      message.error(err instanceof Error ? err.message : "创建失败");
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
          if (selected === row.path) { setSelected(""); setDraft(""); setDirty(false); }
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

  return (
    <div className="h-full overflow-auto px-6 py-4" data-testid="wenxue-character-hub">
      {/* 工具集绑定说明条：UI 与 agent 同源的角色卡数据面 */}
      <div
        className="mb-3 flex items-start gap-2 px-3 py-2 rounded-md border text-[11px] leading-relaxed"
        style={{ borderColor: "var(--md-sys-color-outline-variant)", background: "var(--md-sys-color-surface-container)" }}
      >
        <Users size={13} aria-hidden className="mt-0.5 text-accent shrink-0" />
        <span className="text-fg-dim">
          角色卡保存在小说工作空间的「角色」目录（一角色一卡），与对话里 agent 的
          <code className="mx-1 px-1 rounded bg-bg-soft">wenxue_character_list/read/write</code>
          工具共享同一数据源——这里改，agent 即见；agent 回写（章节推进更新「当前状态」），点刷新即见。
        </span>
      </div>

      <div className="flex items-center gap-2 mb-3">
        <input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="搜索角色…"
          aria-label="搜索角色"
          className="flex-1 min-w-0 rounded-md px-2.5 py-1.5 text-[12px] outline-none"
          style={{ background: "var(--md-sys-color-surface-container)", border: "1px solid var(--md-sys-color-outline-variant)", color: "var(--md-sys-color-text)" }}
        />
        <button
          type="button"
          className={HEAD_BTN}
          onClick={() => void load()}
          title="刷新（agent 侧回写后同步）"
        >
          <RefreshCw size={11} /> 刷新
        </button>
        <button
          type="button"
          className={HEAD_BTN + " text-accent"}
          onClick={() => { setNewName(""); setCreating(true); }}
        >
          <Plus size={12} /> 新建角色
        </button>
      </div>

      {error && (
        <div className="mb-3 px-3 py-2 rounded-md border border-amber-500/30 bg-amber-500/5 text-amber-500 text-[11px]">{error}</div>
      )}
      {loading ? (
        <div className="py-10 text-center text-fg-faint text-[12px]">读取角色卡中…</div>
      ) : cards.length === 0 ? (
        <div className="py-10 text-center text-fg-faint text-[12px]">
          尚无角色卡（工作空间「角色」目录为空）——点「新建角色」建第一张，
          或让对话里的 agent 用 wenxue_character_write 生成。
        </div>
      ) : (
        <div className="grid gap-2" style={{ gridTemplateColumns: "repeat(auto-fill, minmax(240px, 1fr))" }}>
          {filtered.map((c) => (
            <div
              key={c.path}
              role="button"
              tabIndex={0}
              data-testid={`wenxue-char-card-${c.name}`}
              className={`rounded-md border p-2.5 cursor-pointer transition-colors ${selected === c.path ? "border-accent" : "hover:border-accent/40"}`}
              style={{ background: "var(--md-sys-color-surface-container)", borderColor: selected === c.path ? "var(--gaea-glow)" : "var(--md-sys-color-outline-variant)" }}
              onClick={() => openCard(c)}
              onKeyDown={(e) => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); openCard(c); } }}
            >
              <div className="flex items-center justify-between gap-2">
                <span className="text-[13px] font-medium text-fg truncate">{c.name}</span>
                <span className="text-[10px] text-fg-faint shrink-0">{c.words} 字</span>
              </div>
              <div className="mt-1 text-[11px] text-fg-dim leading-relaxed" style={{ display: "-webkit-box", WebkitLineClamp: 2, WebkitBoxOrient: "vertical", overflow: "hidden" }}>
                {c.excerpt || "（空卡）"}
              </div>
              <div className="mt-1.5 flex items-center gap-1">
                <button
                  type="button"
                  className={HEAD_BTN}
                  title="让 AI 按六节结构补全此卡（写入输入框）"
                  onClick={(e) => { e.stopPropagation(); askAi(c); }}
                >
                  <Sparkles size={10} /> 让 AI 完善
                </button>
                <button
                  type="button"
                  className={HEAD_BTN + " hover:!text-red-400"}
                  title="删除角色卡"
                  onClick={(e) => { e.stopPropagation(); remove(c); }}
                >
                  <Trash2 size={10} />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {selected && (
        <div className="mt-4 rounded-md border" style={{ borderColor: "var(--md-sys-color-outline-variant)" }}>
          <div className="flex items-center justify-between gap-2 px-3 py-2 border-b" style={{ borderColor: "var(--md-sys-color-outline-variant)" }}>
            <span className="text-[12px] text-fg-dim font-medium flex items-center gap-1.5">
              <Pencil size={11} aria-hidden /> 编辑 {selected}
              {dirty && <span className="w-1.5 h-1.5 rounded-full bg-amber-400" title="未保存" />}
            </span>
            <button
              type="button"
              className="flex items-center gap-1 px-2 py-1 rounded bg-accent text-accent-fg text-[11px] cursor-pointer hover:opacity-90 disabled:opacity-50"
              onClick={() => void save()}
              disabled={!dirty || saving}
            >
              {saving ? "保存中…" : "保存（CAS）"}
            </button>
          </div>
          <textarea
            value={draft}
            onChange={(e) => { setDraft(e.target.value); setDirty(true); }}
            spellCheck={false}
            aria-label={`编辑角色卡 ${selected}`}
            className="w-full h-72 p-3 text-[12px] font-mono leading-relaxed bg-transparent outline-none resize-y"
            style={{ color: "var(--md-sys-color-text)" }}
          />
        </div>
      )}

      <Modal
        title="新建角色卡"
        open={creating}
        onOk={() => void create()}
        onCancel={() => setCreating(false)}
        okText="创建"
        okButtonProps={{ disabled: !newName.trim() }}
        destroyOnClose
      >
        <input
          autoFocus
          value={newName}
          onChange={(e) => setNewName(e.target.value)}
          onKeyDown={(e) => { if (e.key === "Enter") void create(); }}
          placeholder="角色名（如：林晚）"
          aria-label="角色名"
          className="w-full rounded-md px-2.5 py-1.5 text-[13px] outline-none"
          style={{ background: "var(--md-sys-color-surface-container)", border: "1px solid var(--md-sys-color-outline-variant)", color: "var(--md-sys-color-text)" }}
        />
        <div className="mt-2 text-[11px] text-fg-faint">
          将在「{dirRel || DIR_CANDIDATES[0]}/」下创建 &lt;角色名&gt;.md，预置六节骨架（设定/外貌/性格/口头禅/关系/当前状态）。
        </div>
      </Modal>
    </div>
  );
}

const HEAD_BTN =
  "flex items-center gap-1 px-1.5 py-0.5 border-0 rounded bg-transparent text-fg-dim text-[10px] cursor-pointer hover:bg-bg-soft";
