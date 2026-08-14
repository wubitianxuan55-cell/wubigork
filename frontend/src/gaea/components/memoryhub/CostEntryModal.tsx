import { useEffect, useMemo, useState } from "react";
import { Form, Input, InputNumber, Modal, Select, TreeSelect } from "antd";
import { app } from "../../lib/bridge";
import type { CostCategory, CostEntry, CostSummary } from "../../lib/types";

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
          form.setFieldsValue({ ...e, categoryId: id, tags: (e.tags ?? []).join(", ") });
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
      spec: v.spec ?? "",
      source: v.source ?? "",
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
