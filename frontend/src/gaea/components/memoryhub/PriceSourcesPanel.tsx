import { useCallback, useEffect, useMemo, useState } from "react";
import { Form, Input, Modal, Select, Switch } from "antd";
import { CloudUpload, Coins, Pencil, Plus, RefreshCw, Trash2 } from "../../icons";
import { app } from "../../lib/bridge";
import type { PriceCandidate, PriceFetchRecord, PriceSource } from "../../lib/types";
import { useToast } from "../Toast";

const FREQ_OPTIONS = [
  { value: 0, label: "仅手动" },
  { value: 6, label: "每 6 小时" },
  { value: 24, label: "每天" },
  { value: 168, label: "每周" },
];
const DISPLAY_LIMIT = 60;

function freqLabel(h: number): string {
  return FREQ_OPTIONS.find((o) => o.value === h)?.label ?? `每 ${h} 小时`;
}

// PriceSourcesPanel 价格源订阅与抓取结果管理（记忆中枢/办公侧共用）：
// 订阅固定网站（如各地造价信息网）→ 手动/定时抓取 → 与成本库匹配 →
// 变更高亮（旧价→新价 + 差额/环比）→ 用户勾选确认发布（写回成本库 + 价格历史）
// 或忽略。遵循「无确认不写库」。
export function PriceSourcesPanel({ onChanged }: { onChanged?: () => void }) {
  const [sources, setSources] = useState<PriceSource[]>([]);
  const [fetches, setFetches] = useState<PriceFetchRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [fetchingId, setFetchingId] = useState<string | null>(null);
  const [editOpen, setEditOpen] = useState(false);
  const [editing, setEditing] = useState<PriceSource | null>(null);
  const [deleting, setDeleting] = useState<PriceSource | null>(null);
  const [checked, setChecked] = useState<Record<string, Set<string>>>({});
  const [form] = Form.useForm();
  const toast = useToast();

  const load = useCallback(() => {
    setLoading(true);
    Promise.all([app.PriceSources(), app.PriceFetches()])
      .then(([s, f]) => {
        setSources(s ?? []);
        setFetches(f ?? []);
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const timeText = useMemo(() => {
    return (s: string) => {
      if (!s) return "从未抓取";
      const d = new Date(s);
      if (Number.isNaN(d.getTime())) return s;
      return d.toLocaleString("zh-CN", { hour12: false });
    };
  }, []);

  const defaultChecked = useCallback((cands: PriceCandidate[]): Set<string> => {
    return new Set(cands.filter((c) => c.status !== "无变化").map((c) => c.title));
  }, []);

  const toggle = useCallback((fetchId: string, title: string) => {
    setChecked((prev) => {
      const next = new Set(prev[fetchId] ?? []);
      if (next.has(title)) next.delete(title);
      else next.add(title);
      return { ...prev, [fetchId]: next };
    });
  }, []);

  const openCreate = () => {
    setEditing(null);
    form.resetFields();
    form.setFieldsValue({ parser: "sc_table", frequencyHours: 24, enabled: true });
    setEditOpen(true);
  };
  const openEdit = (src: PriceSource) => {
    setEditing(src);
    form.setFieldsValue({ ...src, cookie: src.headers?.Cookie ?? "" });
    setEditOpen(true);
  };
  const saveSource = async () => {
    const v = await form.validateFields();
    const headers: Record<string, string> = {};
    if (v.cookie) headers.Cookie = v.cookie;
    const src: PriceSource = {
      id: editing?.id ?? "",
      name: v.name,
      url: v.url,
      parser: v.parser ?? "sc_table",
      frequencyHours: v.frequencyHours ?? 0,
      area: v.area ?? "",
      headers,
      enabled: v.enabled ?? true,
      lastFetchAt: editing?.lastFetchAt ?? "",
      createdAt: editing?.createdAt ?? "",
    };
    await app.PriceSourceSave(src);
    setEditOpen(false);
    load();
  };

  const fetchNow = useCallback(
    async (src: PriceSource) => {
      setFetchingId(src.id);
      try {
        const rec = await app.PriceFetch(src.id);
        setChecked((prev) => ({ ...prev, [rec.id]: defaultChecked(rec.candidates) }));
        toast.show(`抓取完成：${rec.candidates.length} 条价格，请确认后发布`, "info");
        load();
      } catch (e) {
        toast.show(`抓取失败：${String(e)}`, "warn");
      } finally {
        setFetchingId(null);
      }
    },
    [defaultChecked, load, toast],
  );

  const applyFetch = useCallback(
    async (f: PriceFetchRecord) => {
      const titles = [...(checked[f.id] ?? defaultChecked(f.candidates))];
      if (titles.length === 0) return;
      try {
        const n = await app.PriceFetchApply(f.id, titles);
        toast.show(`已发布 ${n} 条价格更新`, "info");
        load();
        onChanged?.();
      } catch (e) {
        toast.show(`发布失败：${String(e)}`, "warn");
      }
    },
    [checked, defaultChecked, load, onChanged, toast],
  );

  const ignoreFetch = useCallback(
    async (f: PriceFetchRecord) => {
      await app.PriceFetchIgnore(f.id).catch(() => {});
      load();
    },
    [load],
  );

  const pendingFirst = useMemo(() => {
    const order: Record<string, number> = { pending: 0, applied: 1, ignored: 2 };
    return [...fetches].sort((a, b) => (order[a.status] ?? 3) - (order[b.status] ?? 3) || b.fetchedAt.localeCompare(a.fetchedAt));
  }, [fetches]);

  const countBy = useCallback((cands: PriceCandidate[], status: string) => cands.filter((c) => c.status === status).length, []);

  return (
    <div className="flex flex-col h-full text-fg-dim text-xs">
      <div className="flex items-center justify-between px-3 py-2 border-b border-border-soft">
        <span className="flex items-center gap-1.5 font-semibold text-fg text-sm">
          <CloudUpload size={13} className="text-sky-400" />
          价格源
        </span>
        <div className="flex items-center gap-1.5">
          <button
            className="flex items-center justify-center w-6 h-6 border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft rounded"
            onClick={load}
            title="刷新价格源"
          >
            <RefreshCw size={12} />
          </button>
          <button
            className="flex items-center justify-center w-6 h-6 border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft rounded"
            onClick={openCreate}
            title="新建价格源"
          >
            <Plus size={12} />
          </button>
          {sources.length > 0 && (
            <span className="text-[10px] text-fg-faint border border-border-soft/60 rounded-full px-1.5 py-px">
              {sources.length}
            </span>
          )}
        </div>
      </div>

      <div className="shrink-0 px-3 pt-2 text-[10.5px] text-fg-faint">
        订阅造价信息网价格表，手动/定时抓取后确认发布（旧价自动留存历史）
      </div>

      <div className="flex-1 min-h-0 overflow-y-auto p-2 flex flex-col gap-1.5">
        {loading ? (
          <div className="py-8 text-center text-fg-faint text-[11px]">加载中…</div>
        ) : (
          <>
            {/* 订阅源 */}
            {sources.map((src) => (
              <div key={src.id} className="p-2 rounded-lg border border-border-soft/70 bg-bg-soft/30">
                <div className="flex items-center gap-1.5">
                  <Coins size={12} className="text-sky-400 shrink-0" />
                  <span className="truncate text-fg text-[12px] font-medium">{src.name}</span>
                  <span className="px-1.5 py-px rounded bg-bg-elev text-fg-faint text-[9.5px] shrink-0">{freqLabel(src.frequencyHours)}</span>
                  {!src.enabled && (
                    <span className="px-1.5 py-px rounded bg-bg-elev text-fg-faint text-[9.5px] shrink-0">停用</span>
                  )}
                  <span className="ml-auto shrink-0 text-fg-faint text-[10px]">{timeText(src.lastFetchAt)}</span>
                  <button
                    className="shrink-0 px-2 h-6 rounded-md bg-sky-400/15 text-sky-300 text-[11px] cursor-pointer hover:bg-sky-400/25 transition-colors disabled:opacity-50"
                    disabled={fetchingId === src.id}
                    onClick={() => void fetchNow(src)}
                    title="立即抓取该价格源"
                  >
                    {fetchingId === src.id ? "抓取中…" : "抓取"}
                  </button>
                  <button
                    className="shrink-0 w-6 h-6 inline-flex items-center justify-center rounded text-fg-faint hover:text-fg hover:bg-bg-elev"
                    onClick={() => openEdit(src)}
                    title="编辑价格源"
                  >
                    <Pencil size={11} />
                  </button>
                  <button
                    className="shrink-0 w-6 h-6 inline-flex items-center justify-center rounded text-fg-faint hover:text-red-400 hover:bg-bg-elev"
                    onClick={() => setDeleting(src)}
                    title="删除价格源"
                  >
                    <Trash2 size={11} />
                  </button>
                </div>
                <div className="mt-1 truncate text-fg-faint text-[9.5px] font-mono">{src.url}</div>
              </div>
            ))}

            {/* 抓取结果 */}
            {pendingFirst.map((f) => {
              const selected = checked[f.id] ?? defaultChecked(f.candidates);
              const shown = f.candidates.slice(0, DISPLAY_LIMIT);
              return (
                <div key={f.id} className={`p-2 rounded-lg border ${f.status === "pending" ? "border-amber-400/30 bg-amber-400/5" : "border-border-soft/60 bg-bg-soft/20 opacity-70"}`}>
                  <div className="flex items-center gap-1.5">
                    <span className="truncate text-fg text-[12px] font-medium">{f.sourceName}</span>
                    {f.period && <span className="px-1.5 py-px rounded bg-bg-elev text-fg-faint text-[9.5px]">期 {f.period}</span>}
                    <span className="px-1.5 py-px rounded bg-bg-elev text-fg-faint text-[9.5px]">
                      ↑{countBy(f.candidates, "更新")} 新{countBy(f.candidates, "新增")} 同{countBy(f.candidates, "无变化")}
                    </span>
                    <span className="ml-auto text-fg-faint text-[10px]">{timeText(f.fetchedAt)}</span>
                    {f.status === "pending" ? (
                      <>
                        <button
                          className="shrink-0 px-2 h-6 rounded-md bg-accent/15 text-accent text-[11px] cursor-pointer hover:bg-accent/25 transition-colors"
                          onClick={() => void applyFetch(f)}
                          disabled={selected.size === 0}
                        >
                          发布 {selected.size} 条
                        </button>
                        <button
                          className="shrink-0 px-2 h-6 rounded-md bg-bg-elev text-fg-faint text-[11px] cursor-pointer hover:text-fg transition-colors"
                          onClick={() => void ignoreFetch(f)}
                        >
                          忽略
                        </button>
                      </>
                    ) : (
                      <span className="px-1.5 py-px rounded bg-bg-elev text-fg-faint text-[9.5px]">
                        {f.status === "applied" ? "已发布" : "已忽略"}
                      </span>
                    )}
                  </div>
                  <div className="mt-1.5 space-y-1">
                    {shown.map((c) => (
                      <label key={c.title} className={`flex items-center gap-1.5 text-[11px] ${c.status === "无变化" ? "opacity-55" : ""}`}>
                        <input type="checkbox" checked={selected.has(c.title)} onChange={() => toggle(f.id, c.title)} />
                        <span className="min-w-0 flex-1 truncate">
                          {c.title}
                          {c.spec && <span className="text-fg-faint"> · {c.spec}</span>}
                          {c.unit && <span className="text-fg-faint"> /{c.unit}</span>}
                        </span>
                        {c.status === "更新" && c.existingPrice > 0 && (
                          <span className="shrink-0 text-fg-faint line-through">¥{c.existingPrice}</span>
                        )}
                        <span className={`shrink-0 font-semibold ${c.status === "更新" ? "text-amber-300" : "text-fg"}`}>
                          ¥{c.price}
                        </span>
                        {c.status === "更新" && (
                          <span className={`shrink-0 text-[10px] ${c.diff >= 0 ? "text-red-400" : "text-ok"}`}>
                            {c.diff >= 0 ? "+" : ""}{c.diff}（{c.diff >= 0 ? "+" : ""}{c.diffPct}%）
                          </span>
                        )}
                        {c.anomaly && (
                          <span
                            className="shrink-0 px-1 rounded bg-red-500/15 text-red-400 text-[9.5px]"
                            title={c.anomalyReason}
                          >
                            异常
                          </span>
                        )}
                        {c.status === "新增" && (
                          <span className="shrink-0 px-1 rounded bg-ok/15 text-ok text-[9.5px]">新增</span>
                        )}
                        {c.status === "无变化" && (
                          <span className="shrink-0 text-fg-faint text-[9.5px]">无变化</span>
                        )}
                      </label>
                    ))}
                    {f.candidates.length > DISPLAY_LIMIT && (
                      <div className="text-[10px] text-fg-faint">仅显示前 {DISPLAY_LIMIT} 条，其余请分批处理</div>
                    )}
                  </div>
                </div>
              );
            })}

            {sources.length === 0 && pendingFirst.length === 0 && (
              <div className="flex flex-col items-center justify-center flex-1 gap-2 px-6 text-center text-fg-faint/50">
                <CloudUpload size={24} className="opacity-40" />
                <span className="text-[11px] leading-relaxed">
                  暂无价格源
                  <br />
                  添加造价信息网地址后即可抓取价格
                </span>
              </div>
            )}
          </>
        )}
      </div>

      {/* 新建/编辑价格源 */}
      <Modal
        title={editing ? "编辑价格源" : "新建价格源"}
        open={editOpen}
        onCancel={() => setEditOpen(false)}
        onOk={() => void saveSource()}
        okText="保存"
        cancelText="取消"
        width={560}
      >
        <Form form={form} layout="vertical" size="small">
          <Form.Item label="名称" name="name" rules={[{ required: true, message: "请输入名称" }]}>
            <Input placeholder="如：四川造价信息网（期 758）" />
          </Form.Item>
          <Form.Item label="网址" name="url" rules={[{ required: true, message: "请输入网址" }]}>
            <Input placeholder="http://…/pricelist.aspx?period=758" />
          </Form.Item>
          <div className="grid grid-cols-2 gap-3">
            <Form.Item label="抓取频率" name="frequencyHours">
              <Select options={FREQ_OPTIONS} />
            </Form.Item>
            <Form.Item label="地区过滤" name="area">
              <Input placeholder="如：成都市区（留空取首个报价列）" />
            </Form.Item>
          </div>
          <Form.Item label="Cookie / 自定义请求头（选填）" name="cookie">
            <Input.TextArea
              rows={2}
              placeholder="部分站点（如重庆造价信息网）需要浏览器验证，登录后复制 Cookie 粘贴到这里"
            />
          </Form.Item>
          <Form.Item label="启用" name="enabled" valuePropName="checked">
            <Switch size="small" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 删除确认 */}
      <Modal
        title="删除价格源"
        open={!!deleting}
        onCancel={() => setDeleting(null)}
        onOk={async () => {
          if (deleting) await app.PriceSourceDelete(deleting.id).catch(() => {});
          setDeleting(null);
          load();
        }}
        okText="删除"
        okButtonProps={{ danger: true }}
        cancelText="取消"
      >
        <p className="text-[13px] text-fg-dim">确定删除价格源「{deleting?.name}」吗？（抓取记录与价格历史保留）</p>
      </Modal>
    </div>
  );
}
