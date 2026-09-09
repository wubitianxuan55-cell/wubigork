import { useEffect, useMemo, useState } from "react";
import { Form, Input, InputNumber, Modal, Select, TreeSelect } from "antd";
import { app } from "../../lib/bridge";
import type { CostCategory, CostComponent, CostComposeRecord, CostEntry, CostSummary } from "../../lib/types";
import { ChevronDown, Clock } from "../../icons";
import { ComposeEvidenceTable } from "./ComposeEvidenceTable";

const fmtPrice = new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 2 });

// 确认时间展示：本地时区 YYYY-MM-DD HH:mm（非法/空串原样回显，不编造）。
const fmtRecordTime = (iso: string) => {
  const d = new Date(iso);
  if (!iso || Number.isNaN(d.getTime())) return iso || "—";
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`;
};

// CostEntryModal 成本条目新建/编辑弹窗（记忆中枢 CostLibrary 与办公侧
// CostLibraryPanel 共用，避免两处维护两份表单逻辑）。
export function CostEntryModal({
  open,
  editing,
  onClose,
  onSaved,
}: {
  open: boolean;
  editing: CostSummary | null;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [form] = Form.useForm();
  const [categories, setCategories] = useState<CostCategory[]>([]);
  // v4.158 组价依据回看：该条目全部确认记录（null=加载中/无编辑对象；[]=无记录或加载失败）。
  const [records, setRecords] = useState<CostComposeRecord[] | null>(null);

  // 分类树 → antd TreeSelect treeData + 路径索引（多级：选任意节点即以其完整路径保存）。
  const treeData = useMemo(() => buildTreeData(categories), [categories]);
  const pathById = useMemo(() => {
    const m = new Map<number, string>();
    const walk = (nodes: CostCategory[], prefix: string) => {
      for (const n of nodes ?? []) {
        const p = prefix ? `${prefix}/${n.name}` : n.name;
        m.set(n.id, p);
        walk(n.children ?? [], p);
      }
    };
    walk(categories, "");
    return m;
  }, [categories]);

  useEffect(() => {
    if (!open) return;
    form.resetFields();
    app.CostCategories().then((tree) => setCategories(tree ?? [])).catch(() => {});
    if (editing) {
      app.CostGet(editing.name).then((e) => {
        if (e) {
          const id = [...pathById.entries()].find(([, p]) => p === e.categoryPath)?.[0];
          form.setFieldsValue({
            ...e,
            categoryId: id,
            tags: (e.tags ?? []).join(", "),
            components: e.components ?? [],
          });
        }
      });
    } else {
      form.setFieldsValue({ categoryId: undefined, status: "现行" });
    }
  }, [open, editing, form, pathById]);

  // v4.158 组价依据回看（独立于表单 effect：分类树加载不触发重复拉取）。
  // 加载失败按无记录静默收起（诚实但不打扰）；无记录时折叠区整体不渲染。
  useEffect(() => {
    if (!open || !editing) {
      setRecords(null);
      return;
    }
    let alive = true;
    setRecords(null);
    app.CostComposeRecords(editing.name)
      .then((rs) => {
        if (alive) setRecords(rs ?? []);
      })
      .catch(() => {
        if (alive) setRecords([]);
      });
    return () => {
      alive = false;
    };
  }, [open, editing]);

  const handleSubmit = async () => {
    const v = await form.validateFields();
    const categoryPath = v.categoryId ? pathById.get(v.categoryId) ?? "" : "";
    const entry: CostEntry = {
      name: editing?.name ?? v.name,
      title: v.title,
      code: (v.code ?? "").trim() || undefined,
      category: leafOf(categoryPath) || "其他",
      categoryPath,
      unit: v.unit ?? "",
      price: v.price ?? 0,
      laborFee: v.laborFee ?? 0,
      materialFee: v.materialFee ?? 0,
      machineFee: v.machineFee ?? 0,
      managementFee: v.managementFee ?? 0,
      profitFee: v.profitFee ?? 0,
      advanceFee: v.advanceFee ?? 0,
      taxRate: v.taxRate ?? 0,
      components: (v.components ?? []).filter((c: CostComponent) => c.title?.trim()),
      spec: v.spec ?? "",
      source: v.source ?? "",
      region: v.region ?? "",
      priceDate: v.priceDate ?? "",
      priceType: v.priceType ?? "",
      validUntil: v.validUntil ?? "",
      sourceRow: v.sourceRow ?? 0,
      tags: (v.tags ?? "").split(",").map((s: string) => s.trim()).filter(Boolean),
      status: v.status ?? "现行",
      body: v.body ?? "",
    };
    await app.CostSave(entry);
    onSaved();
  };

  return (
    <Modal
      title={editing ? "编辑成本" : "新建成本"}
      open={open}
      onCancel={onClose}
      onOk={handleSubmit}
      okText="保存"
      destroyOnHidden
      transitionName=""
      maskTransitionName=""
      cancelText="取消"
      width={760}
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
        <Form.Item label="定额/清单编码" name="code" extra="可选；组价检索与归因对标的精确锚点（同名不同地区/口径靠编码区分）">
          <Input placeholder="如：A1-12 / 040101001" />
        </Form.Item>
        <div className="grid grid-cols-3 gap-3">
          <Form.Item label="分类（多级）" name="categoryId">
            <TreeSelect
              treeData={treeData}
              treeDefaultExpandAll
              placeholder="选择分类（支持二级/三级）"
              showSearch
              treeNodeFilterProp="title"
              allowClear
              style={{ width: "100%" }}
            />
          </Form.Item>
          <Form.Item label="单价（元）" name="price">
            <InputNumber min={0} style={{ width: "100%" }} placeholder="3200" />
          </Form.Item>
          <Form.Item label="单位" name="unit">
            <Input placeholder="台班/吨/m³/工日" />
          </Form.Item>
        </div>
        <div className="grid grid-cols-3 gap-3">
          <Form.Item label="人工费（元）" name="laborFee" extra="人材机二级合计（组成行可留空）">
            <InputNumber min={0} style={{ width: "100%" }} placeholder="0" />
          </Form.Item>
          <Form.Item label="材料费（元）" name="materialFee">
            <InputNumber min={0} style={{ width: "100%" }} placeholder="0" />
          </Form.Item>
          <Form.Item label="机械费（元）" name="machineFee">
            <InputNumber min={0} style={{ width: "100%" }} placeholder="0" />
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
          <Form.Item label="地区" name="region">
            <Input placeholder="如：成都市区 / 上海（价格三要素）" />
          </Form.Item>
          <Form.Item label="价格时间 / 期数" name="priceDate">
            <Input placeholder="如：2026-08 / 2026年第2期" />
          </Form.Item>
        </div>
        <div className="grid grid-cols-2 gap-3">
          <Form.Item label="价格口径" name="priceType" extra="出厂价=不含运杂；到场价=含运杂与采保费；安装综合价=含安装调试，套定额时勿重复计取">
            <Select
              allowClear
              placeholder="选择价格口径"
              options={[
                { value: "出厂价", label: "出厂价" },
                { value: "到场价", label: "到场价" },
                { value: "安装综合价", label: "安装综合价" },
              ]}
            />
          </Form.Item>
          <Form.Item label="有效期至" name="validUntil" extra="留空 = 长期有效">
            <Input placeholder="YYYY-MM-DD" />
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
        <Form.Item
          label="人材机组成（二级明细）"
          extra="每一行 = 名称/含量/单价/金额；损耗系数写入备注（导入自动生成）"
        >
          <Form.List name="components">
            {(fields, { add, remove }) => (
              <div className="space-y-1.5">
                {fields.map((field, idx) => (
                  <div
                    key={field.key}
                    className="flex items-start gap-1.5 rounded-lg border border-border/70 bg-bg-soft/40 p-1.5"
                  >
                    <div className="w-24 shrink-0">
                      <Form.Item name={[field.name, "kind"]} className="!mb-0">
                        <Select
                          size="small"
                          placeholder="类别"
                          options={[
                            { value: "人工", label: "人工" },
                            { value: "材料", label: "材料" },
                            { value: "机械", label: "机械" },
                            { value: "人工+机械", label: "人工+机械" },
                          ]}
                        />
                      </Form.Item>
                    </div>
                    <Form.Item name={[field.name, "title"]} className="!mb-0 flex-1 min-w-0">
                      <Input size="small" placeholder="名称，如 挖土方(甩土)" />
                    </Form.Item>
                    <Form.Item name={[field.name, "unit"]} className="!mb-0 w-16 shrink-0">
                      <Input size="small" placeholder="单位" />
                    </Form.Item>
                    <Form.Item name={[field.name, "quantity"]} className="!mb-0 w-20 shrink-0">
                      <InputNumber size="small" min={0} style={{ width: "100%" }} placeholder="含量" />
                    </Form.Item>
                    <Form.Item name={[field.name, "price"]} className="!mb-0 w-20 shrink-0">
                      <InputNumber size="small" min={0} style={{ width: "100%" }} placeholder="单价" />
                    </Form.Item>
                    <Form.Item name={[field.name, "amount"]} className="!mb-0 w-20 shrink-0">
                      <InputNumber size="small" min={0} style={{ width: "100%" }} placeholder="金额" />
                    </Form.Item>
                    <Form.Item name={[field.name, "note"]} className="!mb-0 hidden">
                      <Input />
                    </Form.Item>
                    <button
                      className="shrink-0 w-6 h-6 inline-flex items-center justify-center rounded text-fg-faint hover:text-red-400 hover:bg-bg-elev"
                      onClick={() => remove(field.name)}
                      title={`删除第 ${idx + 1} 行`}
                    >
                      ✕
                    </button>
                  </div>
                ))}
                <button
                  className="w-full h-7 rounded-lg border border-dashed border-border text-fg-faint hover:text-accent hover:border-accent/50 transition-colors text-[11.5px]"
                  onClick={() => add({ kind: "人工", title: "", unit: "", quantity: 0, price: 0, amount: 0, note: "" })}
                >
                  ＋ 添加组成行
                </button>
              </div>
            )}
          </Form.List>
        </Form.Item>
        <div className="grid grid-cols-4 gap-3">
          <Form.Item label="管理费（元）" name="managementFee" extra="费率仅展示追溯，不参与计算">
            <InputNumber min={0} style={{ width: "100%" }} placeholder="0" />
          </Form.Item>
          <Form.Item label="利润（元）" name="profitFee">
            <InputNumber min={0} style={{ width: "100%" }} placeholder="0" />
          </Form.Item>
          <Form.Item label="垫资（元）" name="advanceFee">
            <InputNumber min={0} style={{ width: "100%" }} placeholder="0" />
          </Form.Item>
          <Form.Item label="税率（%）" name="taxRate">
            <InputNumber min={0} max={100} style={{ width: "100%" }} placeholder="9" />
          </Form.Item>
        </div>
        <Form.Item label="导入原始行号（自动记录，一般无需填写）" name="sourceRow">
          <InputNumber min={0} style={{ width: "100%" }} placeholder="0" />
        </Form.Item>
        <Form.Item label="备注 / 计算说明" name="body">
          <Input.TextArea rows={3} placeholder="含燃油与操作手等说明" />
        </Form.Item>
      </Form>
      {/* v4.158 组价依据回看（只读折叠区，不动上方编辑表单）：仅在编辑态且有确认记录时渲染 */}
      {editing && records && records.length > 0 && <ComposeEvidencePanel records={records} />}
    </Modal>
  );
}

interface TreeDataItem {
  title: string;
  value: number;
  children?: TreeDataItem[];
}

// ── 组价依据回看（v4.158 AI 组价复核闭环，只读）──────────────────
// 每次确认组价（CostComposeApply）留痕一条记录：时间 + 推荐价 + LLM/规则徽标；
// 展开单条显示价格带一行（P25/P50/P75）+ 人材机组成小表 + 证据链小表。

/** 组价依据折叠区：按确认时间倒序列出记录（后端已倒序，这里再兜底排一次）。 */
function ComposeEvidencePanel({ records }: { records: CostComposeRecord[] }) {
  const [open, setOpen] = useState(false);
  const [expandedId, setExpandedId] = useState<number | null>(null);
  const sorted = [...records].sort((a, b) => (b.createdAt ?? "").localeCompare(a.createdAt ?? ""));
  return (
    <div className="mt-1 rounded-lg border border-border-soft bg-bg-soft/30">
      <button
        type="button"
        className="flex w-full items-center gap-1.5 px-2.5 py-2 text-left text-[11.5px] font-semibold text-fg hover:text-accent transition-colors"
        aria-expanded={open}
        onClick={() => setOpen((v) => !v)}
      >
        <Clock size={12} className="text-sky-400" />
        组价依据（{sorted.length} 次）
        <ChevronDown
          size={11}
          className={`ml-auto text-fg-faint transition-transform ${open ? "rotate-180" : ""}`}
        />
      </button>
      {open && (
        <div className="space-y-1.5 px-2.5 pb-2.5">
          {sorted.map((r) => (
            <ComposeRecordItem
              key={r.id}
              record={r}
              expanded={expandedId === r.id}
              onToggle={() => setExpandedId((cur) => (cur === r.id ? null : r.id))}
            />
          ))}
        </div>
      )}
    </div>
  );
}

/** 单条确认记录行：展开显示价格带一行 + 人材机组成小表 + 证据链小表。 */
function ComposeRecordItem({
  record,
  expanded,
  onToggle,
}: {
  record: CostComposeRecord;
  expanded: boolean;
  onToggle: () => void;
}) {
  const snap = record.snapshot;
  return (
    <div className="rounded-md border border-border/70 bg-bg">
      <button
        type="button"
        className="flex w-full items-center gap-2 px-2 py-1.5 text-left text-[11px]"
        aria-expanded={expanded}
        onClick={onToggle}
      >
        <span className="shrink-0 tabular-nums text-fg-dim">{fmtRecordTime(record.createdAt)}</span>
        {snap && (
          <span className="shrink-0 font-semibold text-amber-300 tabular-nums">
            ¥{fmtPrice.format(snap.recommendedPrice)}
            {snap.unit ? `/${snap.unit}` : ""}
          </span>
        )}
        <span
          className={`px-1.5 py-px rounded text-[9.5px] ${
            record.llmUsed ? "text-violet-400 bg-violet-400/10" : "text-amber-400 bg-amber-400/10"
          }`}
        >
          {record.llmUsed ? "LLM" : "规则"}
        </span>
        <span className="ml-auto shrink-0 text-fg-faint">{expanded ? "收起" : "展开"}</span>
      </button>
      {expanded &&
        (snap ? (
          <div className="space-y-2 border-t border-border-soft px-2 py-2">
            {snap.band && (
              <div className="text-[10.5px] text-fg-dim tabular-nums">
                <span className="text-fg-faint">价格带 </span>
                P25 ¥{fmtPrice.format(snap.band.p25)} · P50 ¥{fmtPrice.format(snap.band.median)} · P75 ¥
                {fmtPrice.format(snap.band.p75)}
                {`（${snap.band.samples} 个样本）`}
              </div>
            )}
            {snap.components && snap.components.length > 0 && (
              <div>
                <div className="mb-1 text-[10.5px] font-semibold text-fg">人材机组成（{snap.components.length} 行）</div>
                <table className="w-full text-[11px]">
                  <thead className="text-fg-faint text-left">
                    <tr>
                      <th className="py-0.5 pr-2 font-normal w-20">类别</th>
                      <th className="py-0.5 pr-2 font-normal">名称</th>
                      <th className="py-0.5 pr-2 font-normal w-14">单位</th>
                      <th className="py-0.5 pr-2 font-normal w-16 text-right">含量</th>
                      <th className="py-0.5 pr-2 font-normal w-20 text-right">单价(元)</th>
                      <th className="py-0.5 font-normal w-20 text-right">金额(元)</th>
                    </tr>
                  </thead>
                  <tbody>
                    {snap.components.map((c, i) => (
                      <tr key={i} className="border-t border-border-soft/60">
                        <td className="py-1 pr-2 text-fg-dim">{c.kind || "—"}</td>
                        <td className="py-1 pr-2 text-fg">{c.title || "—"}</td>
                        <td className="py-1 pr-2 text-fg-dim">{c.unit || "—"}</td>
                        <td className="py-1 pr-2 text-right tabular-nums text-fg-dim">{c.quantity ?? 0}</td>
                        <td className="py-1 pr-2 text-right tabular-nums text-fg-dim">{fmtPrice.format(c.price ?? 0)}</td>
                        <td className="py-1 text-right tabular-nums text-fg">{fmtPrice.format(c.amount ?? 0)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
            {snap.evidence.length > 0 && (
              <div>
                <div className="mb-1 text-[10.5px] font-semibold text-fg">证据链（{snap.evidence.length} 条）</div>
                <ComposeEvidenceTable rows={snap.evidence} band={snap.band} maxCls="max-h-44" />
              </div>
            )}
          </div>
        ) : (
          <div className="border-t border-border-soft px-2 py-1.5 text-[10.5px] text-fg-faint">
            该次确认未留存快照，仅记录时间与拆解方式
          </div>
        ))}
    </div>
  );
}

function buildTreeData(nodes: CostCategory[]): TreeDataItem[] {
  return (nodes ?? []).map((n) => ({
    title: n.name,
    value: n.id,
    children: n.children?.length ? buildTreeData(n.children) : undefined,
  }));
}

function leafOf(path: string): string {
  const parts = path.split("/").filter(Boolean);
  return parts[parts.length - 1] ?? "";
}
