import { useCallback, useEffect, useMemo, useState } from "react";
import { Modal, Form, Input, InputNumber, Select } from "antd";
import { Coins, Pencil, Plus, RefreshCw, Trash2 } from "../../icons";
import { app } from "../../lib/bridge";
import type { CostEntry, CostSummary } from "../../lib/types";
import { EmptyState } from "../EmptyState";

const CATEGORIES = ["all", "机械", "材料", "人工", "运输", "检测", "其他"];
const STATUSES = ["all", "现行", "草稿", "已归档"];

/** CostLibrary 成本库：成本条目（单价/单位/规格/来源）统一管理。 */
export function CostLibrary() {
  const [entries, setEntries] = useState<CostSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [query, setQuery] = useState("");
  const [category, setCategory] = useState("all");
  const [status, setStatus] = useState("all");
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<CostSummary | null>(null);
  const [deleteName, setDeleteName] = useState<string | null>(null);
  const [form] = Form.useForm();

  const load = useCallback(() => {
    setLoading(true);
    app
      .CostSearch(query, category, status)
      .then((list) => {
        setEntries(list);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, [query, category, status]);

  useEffect(() => {
    const t = setTimeout(load, 250);
    return () => clearTimeout(t);
  }, [load]);

  const openCreate = () => {
    setEditing(null);
    form.resetFields();
    form.setFieldsValue({ category: "其他", status: "现行" });
    setModalOpen(true);
  };
  const openEdit = (s: CostSummary) => {
    setEditing(s);
    app.CostGet(s.name).then((e) => {
      form.setFieldsValue({ ...e, tags: (e.tags ?? []).join(", ") });
    });
    setModalOpen(true);
  };
  const handleSubmit = async () => {
    const v = await form.validateFields();
    const entry: CostEntry = {
      name: editing?.name ?? v.name,
      title: v.title,
      category: v.category ?? "其他",
      unit: v.unit ?? "",
      price: v.price ?? 0,
      spec: v.spec ?? "",
      source: v.source ?? "",
      tags: (v.tags ?? "").split(",").map((s: string) => s.trim()).filter(Boolean),
      status: v.status ?? "现行",
      body: v.body ?? "",
      updatedAt: "",
      createdAt: editing ? "" : "",
    };
    await app.CostSave(entry);
    setModalOpen(false);
    load();
  };
  const handleDelete = async () => {
    if (!deleteName) return;
    await app.CostDelete(deleteName);
    setDeleteName(null);
    load();
  };

  const priceText = useMemo(() => {
    const fmt = new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 2 });
    return (p: number) => "¥" + fmt.format(p);
  }, []);

  return (
    <div className="h-full flex flex-col">
      {/* 工具条 */}
      <div className="shrink-0 flex items-center gap-2 px-4 pt-3 pb-2">
        <div className="text-fg text-[13px] font-medium">成本库</div>
        <span className="text-fg-faint text-[11px]">单价/单位/来源，供方案测算复用</span>
        <div className="ml-auto flex items-center gap-2">
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="搜索名称/规格/来源…"
            className="w-48 px-3 h-8 rounded-lg border border-border bg-bg text-fg text-[12px] placeholder:text-fg-faint outline-none focus:border-accent transition-colors"
          />
          <button
            className="inline-flex items-center gap-1 px-2.5 h-8 rounded-lg border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors text-[12px]"
            onClick={load}
            title="刷新"
          >
            <RefreshCw size={13} />
          </button>
          <button
            className="inline-flex items-center gap-1 px-3 h-8 rounded-lg bg-accent text-white text-[12px] hover:opacity-90 transition-opacity"
            onClick={openCreate}
          >
            <Plus size={13} /> 新建成本
          </button>
        </div>
      </div>

      {/* 筛选 */}
      <div className="shrink-0 flex items-center gap-2 px-4 pb-2 flex-wrap">
        <div className="flex items-center gap-1">
          {CATEGORIES.map((c) => (
            <button
              key={c}
              onClick={() => setCategory(c)}
              className={`px-2.5 h-7 rounded-full text-[11.5px] transition-colors ${
                category === c ? "bg-accent text-white" : "bg-bg-elev text-fg-faint hover:text-fg border border-border"
              }`}
            >
              {c === "all" ? "全部分类" : c}
            </button>
          ))}
        </div>
        <div className="flex items-center gap-1 ml-2">
          {STATUSES.map((s) => (
            <button
              key={s}
              onClick={() => setStatus(s)}
              className={`px-2.5 h-7 rounded-full text-[11.5px] transition-colors ${
                status === s ? "bg-accent text-white" : "bg-bg-elev text-fg-faint hover:text-fg border border-border"
              }`}
            >
              {s === "all" ? "全部状态" : s}
            </button>
          ))}
        </div>
        <span className="ml-auto text-fg-faint text-[11px]">{entries.length} 条</span>
      </div>

      {/* 列表 */}
      <div className="flex-1 min-h-0 overflow-y-auto px-4 pb-4 space-y-2">
        {loading ? (
          <div className="py-10 text-center text-fg-faint text-[13px]">加载中…</div>
        ) : entries.length === 0 ? (
          <EmptyState title="暂无成本条目" desc="添加机械/材料/人工等成本数据，供方案测算复用" />
        ) : (
          entries.map((e) => (
            <div key={e.name} className="p-3 rounded-lg border border-border bg-bg-soft/60">
              <div className="flex items-center gap-2">
                <Coins size={14} className="text-amber-400 shrink-0" />
                <span className="text-fg text-[13px] font-medium truncate">{e.title}</span>
                {e.spec && <span className="text-fg-faint text-[11px] shrink-0">{e.spec}</span>}
                <span className="px-1.5 py-0.5 rounded bg-bg-elev text-fg-faint text-[10.5px] shrink-0">{e.category}</span>
                {e.status !== "现行" && (
                  <span className="px-1.5 py-0.5 rounded bg-bg-elev text-fg-faint text-[10.5px] shrink-0">{e.status}</span>
                )}
                <span className="ml-auto shrink-0 text-fg text-[13px] font-semibold text-amber-300">
                  {priceText(e.price)}
                  {e.unit && <span className="text-fg-faint text-[11px] font-normal"> /{e.unit}</span>}
                </span>
                <div className="flex items-center gap-0.5 shrink-0">
                  <button className="w-6 h-6 inline-flex items-center justify-center rounded text-fg-faint hover:text-fg hover:bg-bg-elev" onClick={() => openEdit(e)} title="编辑">
                    <Pencil size={12} />
                  </button>
                  <button className="w-6 h-6 inline-flex items-center justify-center rounded text-fg-faint hover:text-red-400 hover:bg-bg-elev" onClick={() => setDeleteName(e.name)} title="删除">
                    <Trash2 size={12} />
                  </button>
                </div>
              </div>
              {e.source && <div className="mt-1 text-fg-faint text-[11px]">来源：{e.source}</div>}
            </div>
          ))
        )}
      </div>

      {/* 新建/编辑 Modal */}
      <Modal
        title={editing ? "编辑成本" : "新建成本"}
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={handleSubmit}
        okText="保存"
        cancelText="取消"
        width={560}
      >
        <Form form={form} layout="vertical" size="small">
          {!editing && (
            <Form.Item label="名称（kebab-case，如 hp300）" name="name" rules={[{ required: true, message: "请输入名称" }]}>
              <Input placeholder="hp300" />
            </Form.Item>
          )}
          <Form.Item label="标题" name="title" rules={[{ required: true, message: "请输入标题" }]}>
            <Input placeholder="如：HP300 高频液压振动锤" />
          </Form.Item>
          <div className="grid grid-cols-3 gap-3">
            <Form.Item label="分类" name="category">
              <Select options={CATEGORIES.filter((c) => c !== "all").map((c) => ({ value: c, label: c }))} />
            </Form.Item>
            <Form.Item label="单价（元）" name="price">
              <InputNumber min={0} style={{ width: "100%" }} placeholder="3200" />
            </Form.Item>
            <Form.Item label="单位" name="unit">
              <Input placeholder="台班/吨/m³/工日" />
            </Form.Item>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <Form.Item label="规格型号" name="spec">
              <Input placeholder="300kW" />
            </Form.Item>
            <Form.Item label="来源" name="source">
              <Input placeholder="定额/市场询价/历史项目" />
            </Form.Item>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <Form.Item label="状态" name="status">
              <Select options={["现行", "草稿", "已归档"].map((s) => ({ value: s, label: s }))} />
            </Form.Item>
            <Form.Item label="标签（逗号分隔）" name="tags">
              <Input placeholder="振动锤, 桩基" />
            </Form.Item>
          </div>
          <Form.Item label="备注 / 计算说明" name="body">
            <Input.TextArea rows={3} placeholder="含燃油与操作手等说明" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 删除确认 */}
      <Modal
        title="删除成本"
        open={!!deleteName}
        onCancel={() => setDeleteName(null)}
        onOk={handleDelete}
        okText="删除"
        okButtonProps={{ danger: true }}
        cancelText="取消"
      >
        <p className="text-[13px] text-fg-dim">确定删除成本条目「{deleteName}」吗？</p>
      </Modal>
    </div>
  );
}
