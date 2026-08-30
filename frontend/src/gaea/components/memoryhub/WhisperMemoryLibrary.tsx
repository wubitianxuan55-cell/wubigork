import { useCallback, useEffect, useMemo, useState } from "react";
import { DownloadOutlined, HeartOutlined, ReadOutlined, IdcardOutlined, HomeOutlined, AimOutlined, SmileOutlined, ClockCircleOutlined, ShareAltOutlined, HistoryOutlined, CalendarOutlined } from "@ant-design/icons";
import { Modal, message } from "antd";
import { RefreshCw } from "../../icons";
import { app } from "../../lib/bridge";
import { DOMAIN_COLORS } from "../../lib/domainColors";
import type { WhisperAnchorReplayView, WhisperAnchorView, WhisperEpisodeView, WhisperEpisodeReplayView, WhisperMemoryView } from "../../lib/types";
import { EmptyState } from "../EmptyState";
import { WhisperGraphPanel, currentPersonalityId } from "../../../components/WhisperGraphPanel";

// 对齐后端 memory_taxonomy.go 6 domain（与 WhisperMemoryModal 一致）
// 3.0 Wave 4：领域分类 emoji → antd 图标（MASTER「no emoji-as-icon」）
const DOMAIN_LABELS: Record<string, string> = {
  IDENTITY: "身份", SOCIAL: "社交", DAILY_LIFE: "日常",
  PURSUITS: "追求", INNER_WORLD: "内心", TEMPORAL: "时间",
};
const DOMAIN_ICONS: Record<string, React.ReactNode> = {
  IDENTITY: <IdcardOutlined />, SOCIAL: <HeartOutlined />, DAILY_LIFE: <HomeOutlined />,
  PURSUITS: <AimOutlined />, INNER_WORLD: <SmileOutlined />, TEMPORAL: <ClockCircleOutlined />,
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

const ANCHOR_TYPE_LABELS: Record<string, string> = {
  recurring: "周期纪念日",
  milestone: "里程碑",
  relationship: "关系",
  fuzzy: "模糊",
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
  const [tab, setTab] = useState<"facts" | "episodes" | "anchors">("facts");
  const [facts, setFacts] = useState<WhisperMemoryView[]>([]);
  const [episodes, setEpisodes] = useState<WhisperEpisodeView[]>([]);
  const [anchors, setAnchors] = useState<WhisperAnchorView[]>([]);
  const [loading, setLoading] = useState(true);
  const [query, setQuery] = useState("");
  const [selected, setSelected] = useState<WhisperMemoryView | null>(null);
  const [selectedEp, setSelectedEp] = useState<WhisperEpisodeView | null>(null);
  const [replay, setReplay] = useState<WhisperEpisodeReplayView | null>(null);
  const [replayLoading, setReplayLoading] = useState(false);
  const [replayError, setReplayError] = useState("");
  const [selectedAnchor, setSelectedAnchor] = useState<WhisperAnchorView | null>(null);
  const [anchorReplay, setAnchorReplay] = useState<WhisperAnchorReplayView | null>(null);
  const [anchorReplayLoading, setAnchorReplayLoading] = useState(false);
  const [anchorReplayError, setAnchorReplayError] = useState("");
  const [graphOpen, setGraphOpen] = useState(false);

  const load = useCallback(() => {
    setLoading(true);
    Promise.all([app.WhisperMemories(), app.WhisperEpisodes(), app.WhisperAnchors()])
      .then(([fl, el, al]) => {
        setFacts(fl);
        setEpisodes(el);
        setAnchors(al);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, []);

  useEffect(() => { load(); }, [load]);

  // 情节详情打开时按 ID 重建原始对话（记忆回放，只读）。
  useEffect(() => {
    if (!selectedEp) {
      setReplay(null);
      setReplayLoading(false);
      setReplayError("");
      return;
    }
    let cancelled = false;
    setReplayLoading(true);
    setReplayError("");
    setReplay(null);
    app.WhisperEpisodeReplay(selectedEp.id)
      .then((r) => { if (!cancelled) setReplay(r); })
      .catch((err) => { if (!cancelled) setReplayError(String(err)); })
      .finally(() => { if (!cancelled) setReplayLoading(false); });
    return () => { cancelled = true; };
  }, [selectedEp]);

  // 纪念日详情打开时按锚点 ID 解析关联情节并重建原始对话（时间锚点回放）。
  useEffect(() => {
    if (!selectedAnchor) {
      setAnchorReplay(null);
      setAnchorReplayLoading(false);
      setAnchorReplayError("");
      return;
    }
    let cancelled = false;
    setAnchorReplayLoading(true);
    setAnchorReplayError("");
    setAnchorReplay(null);
    app.WhisperAnchorReplay(selectedAnchor.id)
      .then((r) => { if (!cancelled) setAnchorReplay(r); })
      .catch((err) => { if (!cancelled) setAnchorReplayError(String(err)); })
      .finally(() => { if (!cancelled) setAnchorReplayLoading(false); });
    return () => { cancelled = true; };
  }, [selectedAnchor]);

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
  const filteredAnchors = useMemo(
    () => anchors.filter((a) =>
      !q || a.summary.toLowerCase().includes(q) || a.anchorDate.includes(q) ||
      (ANCHOR_TYPE_LABELS[a.anchorType] ?? a.anchorType).toLowerCase().includes(q),
    ),
    [anchors, q],
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
        <span className="text-fg-faint text-[11px]">gaea 的人格记忆（hermes.db，只读）</span>
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
              {(episodes ?? []).length > 0 && (
                <span className="ml-1 text-fg-faint text-[10px]">{(episodes ?? []).length}</span>
              )}
            </button>
            <button
              className={`px-2.5 h-6.5 rounded-md text-[12px] transition-colors ${
                tab === "anchors" ? "bg-bg-elev text-fg shadow-sm" : "text-fg-faint hover:text-fg"
              }`}
              onClick={() => setTab("anchors")}
            >
              纪念日
              {(anchors ?? []).length > 0 && (
                <span className="ml-1 text-fg-faint text-[10px]">{(anchors ?? []).length}</span>
              )}
            </button>
          </div>
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={`搜索${tab === "facts" ? "记忆" : tab === "episodes" ? "情节" : "纪念日"}…`}
            className="w-40 px-3 h-8 rounded-lg border border-border bg-bg text-fg text-[12px] placeholder:text-fg-faint outline-none focus:border-accent transition-colors"
          />
          <button
            className="inline-flex items-center gap-1 px-2.5 h-8 rounded-lg border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors text-[12px]"
            onClick={() => setGraphOpen(true)}
            title="关系图谱：以实体为中心查看轻语记忆图谱"
          >
            <ShareAltOutlined style={{ fontSize: 12 }} />
            关系图谱
          </button>
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
            <EmptyState message="暂无情节记忆 — 与 gaea 深入对话（≥3 轮/情绪峰值）后会自动沉淀记忆片段" />
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
        ) : tab === "anchors" ? (
          /* ── 纪念日时间锚点（日期倒序，类型徽标 + 情绪） ── */
          filteredAnchors.length === 0 ? (
            <EmptyState message="暂无时间锚点 — 对话中提到生日/纪念日/里程碑并伴随情绪时，gaea 会自动沉淀「那一天」，点击即可回放原始对话" />
          ) : (
            <div className="space-y-2.5 py-1">
              {filteredAnchors.map((an) => (
                <div
                  key={an.id}
                  className="p-2.5 rounded-lg border border-border bg-bg-soft/50 hover:border-accent/50 cursor-pointer transition-colors"
                  onClick={() => setSelectedAnchor(an)}
                >
                  <div className="flex items-center gap-2">
                    <CalendarOutlined style={{ fontSize: 13, color: "var(--md-sys-color-primary)" }} />
                    <span className="text-fg text-[12.5px] font-medium">{an.anchorDate}</span>
                    <span className="px-1.5 py-0.5 rounded bg-amber-400/15 text-amber-300/90 text-[10px]">
                      {ANCHOR_TYPE_LABELS[an.anchorType] ?? an.anchorType}
                    </span>
                    {an.emotionalIntensity > 0 && (
                      <span className="ml-auto shrink-0 text-fg-faint text-[10.5px]">
                        情绪 {Math.round(an.emotionalIntensity * 100)}%
                      </span>
                    )}
                  </div>
                  <div className="mt-1 text-fg-dim text-[12px] leading-relaxed line-clamp-2">{an.summary}</div>
                </div>
              ))}
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
                  <span className="text-fg-dim" style={{ fontSize: 13, color: "var(--md-sys-color-text-secondary)" }}>{DOMAIN_ICONS[g.domain]}</span>
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
        destroyOnHidden
        transitionName=""
        maskTransitionName=""
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
        destroyOnHidden
        transitionName=""
        maskTransitionName=""
        title={
          <span>
            <ReadOutlined style={{ color: DOMAIN_COLORS.whisper }} />
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
            {/* 记忆回放：重建该情节的原始对话（审计 §C 欠账收口） */}
            <ReplayDialogue replay={replay} loading={replayLoading} error={replayError} />
          </div>
        )}
      </Modal>

      {/* 纪念日详情弹窗（时间锚点回放：锚点 → 关联情节 → 原始对话） */}
      <Modal
        open={!!selectedAnchor}
        onCancel={() => setSelectedAnchor(null)}
        footer={null}
        width={560}
        destroyOnHidden
        transitionName=""
        maskTransitionName=""
        title={
          <span>
            <CalendarOutlined style={{ color: DOMAIN_COLORS.whisper }} />
            <span className="ml-1 text-pink-400">纪念日回放</span>
          </span>
        }
      >
        {selectedAnchor && (
          <div className="space-y-2.5">
            <div className="flex items-center gap-2">
              <span className="text-fg text-[13px] font-medium">{selectedAnchor.anchorDate}</span>
              <span className="px-1.5 py-0.5 rounded bg-amber-400/15 text-amber-300/90 text-[10px]">
                {ANCHOR_TYPE_LABELS[selectedAnchor.anchorType] ?? selectedAnchor.anchorType}
              </span>
            </div>
            <div className="p-3 rounded-lg bg-bg-soft border border-border text-fg-dim text-[13px] leading-relaxed whitespace-pre-wrap">
              {selectedAnchor.summary}
            </div>
            {(anchorReplay?.linkedFactSummaries ?? []).length > 0 && (
              <div className="flex flex-wrap gap-1.5">
                {(anchorReplay?.linkedFactSummaries ?? []).map((s, i) => (
                  <span key={i} className="px-1.5 py-0.5 rounded-full bg-bg-elev text-fg-faint text-[10.5px]">{s}</span>
                ))}
              </div>
            )}
            <ReplayDialogue
              replay={anchorReplay?.episodeReplay ?? null}
              loading={anchorReplayLoading}
              error={anchorReplayError}
            />
          </div>
        )}
      </Modal>

      {/* 关系图谱面板 */}
      <WhisperGraphPanel
        open={graphOpen}
        personalityId={currentPersonalityId()}
        onClose={() => setGraphOpen(false)}
      />
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

/** ReplayDialogue 记忆回放区块：情节回放与纪念日回放共用（用户/gaea 气泡 + 轮次）。 */
function ReplayDialogue(p: { replay: WhisperEpisodeReplayView | null; loading: boolean; error: string }) {
  return (
    <div className="mt-2 pt-2.5 border-t border-border">
      <div className="flex items-center gap-1.5 text-[12px] font-medium text-fg mb-2">
        <HistoryOutlined style={{ fontSize: 12 }} />
        回放原始对话
      </div>
      {p.loading ? (
        <div className="py-4 text-center text-fg-faint text-[12px]">正在重建这段记忆…</div>
      ) : p.error ? (
        <div className="py-2 text-[12px] text-red-400">回放失败：{p.error}</div>
      ) : p.replay?.replayable ? (
        <div className="space-y-2.5 max-h-64 overflow-y-auto pr-1">
          {p.replay.dialogue.map((line, i) => (
            <div key={i} className={`flex ${line.role === "user" ? "justify-end" : "justify-start"}`}>
              <div
                className={`max-w-[85%] px-2.5 py-1.5 rounded-lg text-[12.5px] leading-relaxed whitespace-pre-wrap ${
                  line.role === "user" ? "bg-pink-500/15 text-fg" : "bg-bg-elev border border-border text-fg-dim"
                }`}
              >
                <div className={`text-[10px] mb-0.5 ${line.role === "user" ? "text-right text-pink-400/80" : "text-fg-faint"}`}>
                  {line.role === "user" ? "你" : "gaea"} · 第{line.turnIndex}轮
                </div>
                {line.text}
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="py-2 text-fg-faint text-[12px] leading-relaxed">
          {p.replay ? "原始对话已超出保留范围（仅存记忆摘要），无法逐字回放。" : "暂无可回放的原始对话。"}
        </div>
      )}
    </div>
  );
}
