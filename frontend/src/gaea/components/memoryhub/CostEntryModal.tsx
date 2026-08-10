import { useEffect } from "react";
import { Form, Input, InputNumber, Modal, Select } from "antd";
import { app } from "../../lib/bridge";
import type { CostEntry, CostSummary } from "../../lib/types";

const CATEGORIES = ["机械", "材料", "人工", "运输", "检测", "其他"];

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

  useEffect(() => {
    if (!open) return;
    form.resetFields();
    if (editing) {
      app.CostGet(editing.name).then((e) => {
        if (e) form.setFieldsValue({ ...e, tags: (e.tags ?? []).join(", ") });
      });
    } else {
      form.setFieldsValue({ category: "其他", status: "现行" });
    }
  }, [open, editing, form]);

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
      createdAt: "",
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
            <Select options={CATEGORIES.map((c) => ({ value: c, label: c }))} />
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
