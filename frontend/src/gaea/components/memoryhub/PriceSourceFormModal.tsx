import { useEffect } from "react";
import { Form, Input, Modal, Select, Switch } from "antd";
import { app } from "../../lib/bridge";
import type { PriceSource } from "../../lib/types";

const FREQ_OPTIONS = [
  { value: 0, label: "仅手动" },
  { value: 6, label: "每 6 小时" },
  { value: 24, label: "每天" },
  { value: 168, label: "每周" },
];

// PriceSourceFormModal 新建/编辑价格源（价格源页与价格源阅览仓库共用）。
export function PriceSourceFormModal({
  open,
  editing,
  onClose,
  onSaved,
}: {
  open: boolean;
  editing: PriceSource | null;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [form] = Form.useForm();

  useEffect(() => {
    if (!open) return;
    form.resetFields();
    if (editing) {
      form.setFieldsValue({ ...editing, cookie: editing.headers?.Cookie ?? "" });
    } else {
      form.setFieldsValue({ parser: "sc_table", frequencyHours: 24, enabled: true });
    }
  }, [open, editing, form]);

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
    onSaved();
  };

  return (
    <Modal
      title={editing ? "编辑价格源" : "新建价格源"}
      open={open}
      // WebView2 rAF 冻结时退出动画不结束会残留遮罩挡住点击；禁用动画、关闭即卸载。
      destroyOnHidden
      transitionName=""
      maskTransitionName=""
      onCancel={onClose}
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
  );
}
