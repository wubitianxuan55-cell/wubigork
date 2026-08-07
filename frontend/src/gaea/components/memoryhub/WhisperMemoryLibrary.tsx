import { useCallback, useEffect, useMemo, useState } from "react";
import { DownloadOutlined, HeartOutlined, ReadOutlined } from "@ant-design/icons";
import { Modal, message } from "antd";
import { RefreshCw } from "../../icons";
import { app } from "../../lib/bridge";
import type { WhisperEpisodeView, WhisperMemoryView } from "../../lib/types";
import { EmptyState } from "../EmptyState";

// 对齐后端 memory_taxonomy.go 6 domain（与 WhisperMemoryModal 一致）
const DOMAIN_LABELS: Record<string, string> = {
  IDENTITY: "🪪 身份", SOCIAL: "💕 社交", DAILY_LIFE: "🏠 日常",
  PURSUITS: "🎯 追求", INNER_WORLD: "🧘 内心", TEMPORAL: "⏰ 时间",
};
const DOMAIN_ORDER = ["IDENTITY", "SOCIAL", "DAILY_LIFE", "PURSUITS", "INNER_WORLD", "TEMPORAL"];

// 主导情绪 → emoji（轻量情绪色标，差异化而非对标）
const EMOTION_EMOJI: Record<string, string> = {
  开心: "😊", 喜悦: "😊", 快乐: "😄", 平静: "🙂", 温柔: "🥰",
  爱: "😍", 亲密: "💗", 感动: "🥹", 难过: "😢", 低落: "😞",
  悲伤: "😢", 生气: "😠", 愤怒: "😡", 害怕: "😨", 焦虑: "😰",
  紧张: "😖", 惊喜: "😮", 惊讶: "😲", 兴奋: "🤩", 期待: "🤗",
  疲惫: "😪", 孤独: "🫥", 感激: "🙏",
};

function emotionEmoji(label: string): string {
  if (!label) return "✨";
  return EMOTION_EMOJI[label] ?? EMOTION_EMOJI[label.trim()] ?? "✨";
}

function formatTime(iso: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getMonth() + 1}月${d.getDate()}日 ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

/** WhisperMemoryLibrary 右脑聊天记忆库（只读）：hermes.db 记忆事实 + 情节记忆浏览。 */
export function WhisperMemoryLibrary() {
  const [tab, setTab] = useState<"facts" | "episodes">("facts");
  const [facts, setFacts] = useState<WhisperMemoryView[]>([]);
  const [episodes, setEpisodes] = useState<WhisperEpisodeView[]>([]);
  const [loading, setLoading] = useState(true);
  const [query, setQuery] = useState("");
  const [selected, setSelected] = useState<WhisperMemoryView | null>(null);
  const [selectedEp, setSelectedEp] = useState<WhisperEpisodeView | null>(null);

  const load = useCallback(() => {
    setLoading(true);
    Promise.all([app.WhisperMemories(), app.WhisperEpisodes()])
      .then(([fl, el]) => {
        setFacts(fl);
        setEpisodes(el);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, []);

  useEffect(() => { load(); }, [load]);

  const handleExport = useCallback(async () => {
    const dir = await app.PickDirectory();
    if (!dir) return;
    try {
      const n = await app.WhisperExportArchive(dir);
      message.success(`已导出 ${n} 个 Markdown 档案到 ${dir}`);
    } catch (err) {
      message.error(`导出失败: ${String(err)}`);
    }
  }, []);

  const q = query.trim().toLowerCase();
  const filtered = useMemo(
    () => facts.filter((f) =>
      !q ||
      f.subject.toLowerCase().includes(q) ||
      f.summary.toLowerCase().includes(q) ||
      f.subcategory.toLowerCase().includes(q),
    ),
    [facts, q],
  );
  const filteredEpisodes = useMemo(
    () => episodes.filter((e) =>
      !q || e.summary.toLowerCase().includes(q) || e.dominantEmotion.toLowerCase().includes(q),
    ),
    [episodes, q],
  );

  // 按 6 domain 分组（未识别的归入"其他"）
  const groups = useMemo(() => {
    const byDomain = new Map<string, WhisperMemoryView[]>();
    for (const f of filtered) {
      const key = DOMAIN_ORDER.includes(f.domain) ? f.domain : "其他";
      if (!byDomain.has(key)) byDomain.set(key, []);
      byDomain.get(key)!.push(f);
    }
    const order = [...DOMAIN_ORDER];
    if (byDomain.has("其他")) order.push("其他");
    return order.filter((d) => byDomain.has(d)).map((d) => ({ domain: d, items: byDomain.get(d)! }));
  }, [filtered]);

  return (
    <div className="h-full flex flex-col">
      {/* 工具条 */}
      <div className="shrink-0 flex items-center gap-2 px-4 pt-3 pb-2">
        <div className="text-fg text-[13px] font-medium">聊天记忆</div>
        <span className="text-fg-faint text-[11px]">Hermes 的人格记忆（hermes.db，只读）</span>
        <div className="ml-auto flex items-center gap-2">
          {/* Tab 切换：事实 / 情节 */}
          <div className="flex rounded-lg border border-border bg-bg p-0.5">
            <button
              className={`px-2.5 h-6.5 rounded-md text-[12px] transition-colors ${
                tab === "facts" ? "bg-bg-elev text-fg shadow-sm" : "text-fg-faint hover:text-fg"
              }`}
              onClick={() => setTab("facts")}
            >
              事实
            </button>
            <button
              className={`px-2.5 h-6.5 rounded-md text-[12px] transition-colors ${
                tab === "episodes" ? "bg-bg-elev text-fg shadow-sm" : "text-fg-faint hover:text-fg"
              }`}
              onClick={() => setTab("episodes")}
            >
              情节
              {episodes.length > 0 && (
                <span className="ml-1 text-fg-faint text-[10px]">{episodes.length}</span>
              )}
            </button>
          </div>
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={`搜索${tab === "facts" ? "记忆" : "情节"}…`}
            className="w-40 px-3 h-8 rounded-lg border border-border bg-bg text-fg text-[12px] placeholder:text-fg-faint outline-none focus:border-accent transition-colors"
          />
          <button
            className="inline-flex items-center gap-1 px-2.5 h-8 rounded-lg border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors text-[12px]"
            onClick={handleExport}
            title="导出归档（Markdown 按领域/子类分目录）"
          >
            <DownloadOutlined style={{ fontSize: 12 }} />
            导出归档
          </button>
          <button
            className="inline-flex items-center gap-1 px-2.5 h-8 rounded-lg border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors text-[12px]"
            onClick={load}
            title="刷新"
          >
            <RefreshCw size={13} />
          </button>
        </div>
      </div>

      {/* 内容区 */}
      <div className="flex-1 min-h-0 overflow-y-auto px-4 pb-4">
        {loading ? (
          <div className="py-10 text-center text-fg-faint text-[13px]">加载中…</div>
        ) : tab === "episodes" ? (
          /* ── 情节时间线（时间倒序，情绪角标 + 强度条 + 关键词） ── */
          filteredEpisodes.length === 0 ? (
            <EmptyState message="暂无情节记忆 — 与 Hermes 深入对话（≥3 轮/情绪峰值）后会自动沉淀记忆片段" />
          ) : (
            <div className="relative space-y-2.5 py-1">
              {/* 时间线竖线 */}
              <div className="absolute left-[9px] top-3 bottom-3 w-px bg-border" />
              {filteredEpisodes.map((ep) => {
                const pct = Math.max(8, Math.min(100, Math.round(ep.emotionalIntensity * 100)));
                return (
                  <div
                    key={ep.id}
                    className="relative pl-8 cursor-pointer group"
                    onClick={() => setSelectedEp(ep)}
                  >
                    {/* 时间线节点 */}
                    <div className="absolute left-0 top-3 w-[18px] h-[18px] rounded-full border-2 border-accent/60 bg-bg flex items-center justify-center">
                      <span className="text-[9px] leading-none">{emotionEmoji(ep.dominantEmotion)}</span>
                    </div>
                    <div className="p-2.5 rounded-lg border border-border bg-bg-soft/50 group-hover:border-accent/50 transition-colors">
                      <div className="flex items-center gap-2">
                        <span className="text-fg text-[12.5px] leading-snug line-clamp-2 flex-1">{ep.summary}</span>
                      </div>
                      <div className="mt-1.5 flex items-center gap-2">
                        {/* 情绪 + 强度条 */}
                        <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-bg-elev text-fg-dim text-[10.5px]">
                          {emotionEmoji(ep.dominantEmotion)} {ep.dominantEmotion || "平静"}
                        </span>
                        <div className="w-16 h-1 rounded-full bg-bg-elev overflow-hidden" title={`情绪强度 ${pct}%`}>
                          <div className="h-full rounded-full bg-gradient-to-r from-amber-400/70 to-pink-400/80" style={{ width: `${pct}%` }} />
                        </div>
                        {/* 关键词 chips */}
                        {(ep.keywords ?? []).slice(0, 3).map((k) => (
                          <span key={k} className="px-1.5 py-0.5 rounded-full bg-amber-400/15 text-amber-300/90 text-[10px]">
                            {k}
                          </span>
                        ))}
                        <span className="ml-auto shrink-0 text-fg-faint text-[10.5px]">
                          {formatTime(ep.createdAt)}
                          {ep.startTurn > 0 && ` · 第${ep.startTurn}-${ep.endTurn}轮`}
                        </span>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          )
        ) : filtered.length === 0 ? (
          <EmptyState message="暂无聊天记忆 — 与角色对话产生的人格记忆会出现在这里（只读浏览）" />
        ) : (
          /* ── 事实分组列表 ── */
          <div className="space-y-4">
            {groups.map((g) => (
              <div key={g.domain}>
                <div className="flex items-center gap-2 mb-1.5">
                  <HeartOutlined style={{ fontSize: 13, color: "#f472b6" }} />
                  <span className="text-fg-dim text-[12px] font-medium">{DOMAIN_LABELS[g.domain] ?? g.domain}</span>
                  <span className="text-fg-faint text-[10.5px]">{g.items.length}</span>
                </div>
                <div className="space-y-1.5">
                  {g.items.map((f) => (
                    <div
                      key={f.id}
                      className="p-2.5 rounded-lg border border-border bg-bg-soft/50 hover:border-accent/50 cursor-pointer transition-colors"
                      onClick={() => setSelected(f)}
                    >
                      <div className="flex items-center gap-2">
                        <span className="text-fg text-[12.5px] font-medium truncate">{f.subject}</span>
                        {f.tier === "core" && (
                          <span className="px-1.5 py-0.5 rounded bg-amber-400/20 text-amber-300 text-[10px]">核心</span>
                        )}
                        {f.status === "retired" && (
                          <span className="px-1.5 py-0.5 rounded bg-bg-elev text-fg-faint text-[10px]">已退役</span>
                        )}
                        <span className="ml-auto shrink-0 text-fg-faint text-[10.5px]">
                          W{f.weight.toFixed(1)} · C{f.confidence.toFixed(1)}
                        </span>
                      </div>
                      <div className="mt-1 text-fg-dim text-[12px] leading-relaxed line-clamp-2">{f.summary}</div>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* 事实详情弹窗 */}
      <Modal
        open={!!selected}
        onCancel={() => setSelected(null)}
        footer={null}
        width={520}
        title={
          <span>
            <span className="text-pink-400">聊天记忆</span>
            {" · "}{selected?.subject}
          </span>
        }
      >
        {selected && (
          <div className="space-y-2.5">
            <div className="p-3 rounded-lg bg-bg-soft border border-border text-fg-dim text-[13px] leading-relaxed whitespace-pre-wrap">
              {selected.summary}
            </div>
            <div className="grid grid-cols-2 gap-x-4 gap-y-1.5 text-[12px]">
              <RowItem label="领域" value={selected.domain} />
              <RowItem label="子类" value={selected.subcategory} />
              <RowItem label="权重" value={selected.weight.toFixed(2)} />
              <RowItem label="置信度" value={selected.confidence.toFixed(2)} />
              <RowItem label="层级" value={selected.tier === "core" ? "核心" : selected.tier || "普通"} />
              <RowItem label="状态" value={selected.status === "retired" ? "已退役" : "活跃"} />
              <RowItem label="更新" value={selected.updatedAt ? new Date(selected.updatedAt).toLocaleString() : ""} />
              <RowItem label="ID" value={selected.id} mono />
            </div>
          </div>
        )}
      </Modal>

      {/* 情节详情弹窗 */}
      <Modal
        open={!!selectedEp}
        onCancel={() => setSelectedEp(null)}
        footer={null}
        width={520}
        title={
          <span>
            <ReadOutlined style={{ color: "#f472b6" }} />
            <span className="ml-1 text-pink-400">情节记忆</span>
          </span>
        }
      >
        {selectedEp && (
          <div className="space-y-2.5">
            <div className="p-3 rounded-lg bg-bg-soft border border-border text-fg-dim text-[13px] leading-relaxed whitespace-pre-wrap">
              {selectedEp.summary}
            </div>
            <div className="grid grid-cols-2 gap-x-4 gap-y-1.5 text-[12px]">
              <RowItem label="情绪" value={`${emotionEmoji(selectedEp.dominantEmotion)} ${selectedEp.dominantEmotion || "平静"}`} />
              <RowItem label="强度" value={`${Math.round(selectedEp.emotionalIntensity * 100)}%`} />
              <RowItem label="时间" value={selectedEp.createdAt ? new Date(selectedEp.createdAt).toLocaleString() : ""} />
              <RowItem label="轮次" value={selectedEp.startTurn > 0 ? `第 ${selectedEp.startTurn}-${selectedEp.endTurn} 轮` : ""} />
              <RowItem label="关键词" value={(selectedEp.keywords ?? []).join("、")} />
              <RowItem label="会话" value={selectedEp.sourceSessionId} mono />
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}

function RowItem(p: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-fg-faint w-14 shrink-0">{p.label}</span>
      <span className={`text-fg-dim truncate ${p.mono ? "font-mono text-[11px]" : ""}`}>{p.value}</span>
    </div>
  );
}
