import { useEffect, useMemo, useState } from "react";
import { Form, Input, InputNumber, Modal, Select, TreeSelect } from "antd";
import { app } from "../../lib/bridge";
import type { CostCategory, CostComponent, CostEntry, CostSummary } from "../../lib/types";

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

  const handleSubmit = async () => {
    const v = await form.validateFields();
    const categoryPath = v.categoryId ? pathById.get(v.categoryId) ?? "" : "";
    const entry: CostEntry = {
      name: editing?.name ?? v.name,
      title: v.title,
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
    </Modal>
  );
}

interface TreeDataItem {
  title: string;
  value: number;
  children?: TreeDataItem[];
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
