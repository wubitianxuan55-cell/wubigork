import { useCallback, useEffect, useState } from "react";
import { Modal, Button, Input, Form, Tag } from "antd";
import { AlertCircle, Brain, Plus, Pencil, Trash2, RefreshCw } from "../../icons";
import { app } from "../../lib/bridge";
import type { ProfileFactView } from "../../lib/types";
import { EmptyState } from "../EmptyState";

const TYPE_COLORS: Record<string, string> = {
  user: "blue",
  feedback: "purple",
  project: "green",
  reference: "orange",
};

/** ProfileLibrary 主脑全局画像库：跨板块共享的用户画像事实（CRUD + 冲突提示）。 */
export function ProfileLibrary() {
  const [facts, setFacts] = useState<ProfileFactView[]>([]);
  const [conflicts, setConflicts] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<ProfileFactView | null>(null);
  const [deleteName, setDeleteName] = useState<string | null>(null);
  const [form] = Form.useForm();

  const load = useCallback(() => {
    setLoading(true);
    Promise.all([app.ProfileList(), app.ProfileConflicts()])
      .then(([list, c]) => {
        setFacts(list);
        setConflicts(c);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, []);

  useEffect(() => { load(); }, [load]);

  const openCreate = () => {
    setEditing(null);
    form.resetFields();
    form.setFieldsValue({ type: "user", kind: "semantic" });
    setModalOpen(true);
  };

  const openEdit = (f: ProfileFactView) => {
    setEditing(f);
    form.setFieldsValue(f);
    setModalOpen(true);
  };

  const handleSubmit = async () => {
    const v = await form.validateFields();
    const fact: ProfileFactView = {
      name: editing?.name ?? v.name,
      title: v.title ?? "",
      description: v.description,
      type: v.type ?? "user",
      kind: v.kind ?? "semantic",
      tags: (v.tags ?? "").split(",").map((s: string) => s.trim()).filter(Boolean),
      body: v.body,
    };
    await app.ProfileSave(fact);
    setModalOpen(false);
    load();
  };

  const handleDelete = async () => {
    if (!deleteName) return;
    await app.ProfileDelete(deleteName);
    setDeleteName(null);
    load();
  };

  // 冲突裁决：prefer=profile 删 facts（以画像为准）；prefer=facts 删画像
  const handleResolve = async (name: string, prefer: "profile" | "facts") => {
    await app.ProfileResolveConflict(name, prefer);
    load();
  };

  return (
    <div className="h-full flex flex-col">
      {/* 冲突提示横幅 */}
      {conflicts.length > 0 && (
        <div className="shrink-0 mx-4 mt-3 px-3 py-2 rounded-lg border border-yellow-300/40 bg-yellow-300/10 text-[12px] text-fg-dim flex items-start gap-2">
          <AlertCircle size={14} className="text-yellow-300 mt-0.5 shrink-0" />
          <div>
            <div className="font-medium text-yellow-200">画像与遗留 facts 冲突（{conflicts.length}）</div>
            {conflicts.map((c, i) => {
              const name = c.split(":")[0].trim();
              return (
                <div key={i} className="flex items-start gap-2 text-fg-faint">
                  <span className="flex-1 min-w-0">{c}</span>
                  <span className="flex items-center gap-1 shrink-0">
                    <button
                      className="px-1.5 py-0.5 rounded bg-amber-400/20 text-amber-200 text-[10.5px] hover:bg-amber-400/30"
                      onClick={() => handleResolve(name, "profile")}
                      title="删除办公 facts 中的同名事实，以画像为准"
                    >
                      以画像为准
                    </button>
                    <button
                      className="px-1.5 py-0.5 rounded bg-bg-elev text-fg-faint text-[10.5px] hover:text-fg"
                      onClick={() => handleResolve(name, "facts")}
                      title="删除画像，以办公 facts 为准"
                    >
                      以 facts 为准
                    </button>
                  </span>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* 工具条 */}
      <div className="shrink-0 flex items-center gap-2 px-4 pt-3 pb-2">
        <div className="text-fg text-[13px] font-medium">用户画像</div>
        <span className="text-fg-faint text-[11px]">跨板块共享的主脑全局画像</span>
        <div className="ml-auto flex items-center gap-1.5">
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
            <Plus size={13} /> 新建画像
          </button>
        </div>
      </div>

      {/* 列表 */}
      <div className="flex-1 min-h-0 overflow-y-auto px-4 pb-4 space-y-2">
        {loading ? (
          <div className="py-10 text-center text-fg-faint text-[13px]">加载中…</div>
        ) : facts.length === 0 ? (
          <EmptyState message="暂无用户画像 — 办公 agent 记住的用户偏好会出现在这里（remember type=user）" />
        ) : (
          facts.map((f) => (
            <div key={f.name} className="p-3 rounded-lg border border-border bg-bg-soft/60">
              <div className="flex items-center gap-2">
                <Brain size={14} className="text-accent shrink-0" />
                <span className="text-fg text-[13px] font-medium truncate">{f.title || f.name}</span>
                <Tag color={TYPE_COLORS[f.type] ?? "default"} style={{ fontSize: 11, lineHeight: "18px", marginInlineEnd: 0 }}>
                  {f.type}
                </Tag>
                {f.kind !== "semantic" && (
                  <Tag style={{ fontSize: 11, lineHeight: "18px", marginInlineEnd: 0 }}>{f.kind}</Tag>
                )}
                <div className="ml-auto flex items-center gap-0.5">
                  <button className="w-6 h-6 inline-flex items-center justify-center rounded text-fg-faint hover:text-fg hover:bg-bg-elev" onClick={() => openEdit(f)} title="编辑">
                    <Pencil size={12} />
                  </button>
                  <button className="w-6 h-6 inline-flex items-center justify-center rounded text-fg-faint hover:text-red-400 hover:bg-bg-elev" onClick={() => setDeleteName(f.name)} title="删除">
                    <Trash2 size={12} />
                  </button>
                </div>
              </div>
              <div className="mt-1.5 text-fg-dim text-[12.5px] leading-relaxed whitespace-pre-wrap">{f.description}</div>
              {f.tags.length > 0 && (
                <div className="mt-1.5 flex flex-wrap gap-1">
                  {f.tags.map((t) => (
                    <span key={t} className="px-1.5 py-0.5 rounded bg-bg-elev text-fg-faint text-[10.5px]">{t}</span>
                  ))}
                </div>
              )}
            </div>
          ))
        )}
      </div>

      {/* 新建/编辑 Modal */}
      <Modal
        title={editing ? "编辑画像" : "新建画像"}
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
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
            <Form.Item label="名称（kebab-case，如 prefers-tabs）" name="name" rules={[{ required: true, message: "请输入名称" }]}>
              <Input placeholder="prefers-tabs" />
            </Form.Item>
          )}
          <Form.Item label="标题" name="title">
            <Input placeholder="用户偏好（可选）" />
          </Form.Item>
          <Form.Item label="描述（一行摘要）" name="description" rules={[{ required: true, message: "请输入描述" }]}>
            <Input placeholder="如：用户喜欢先给大纲再展开" />
          </Form.Item>
          <Form.Item label="正文" name="body" rules={[{ required: true, message: "请输入正文" }]}>
            <Input.TextArea rows={4} placeholder="事实细节（Markdown）" />
          </Form.Item>
          <div className="grid grid-cols-2 gap-3">
            <Form.Item label="类型" name="type">
              <Input />
            </Form.Item>
            <Form.Item label="标签（逗号分隔）" name="tags">
              <Input placeholder="办公, 写作风格" />
            </Form.Item>
          </div>
        </Form>
      </Modal>

      {/* 删除确认 */}
      <Modal
        title="删除画像"
        open={!!deleteName}
        onCancel={() => setDeleteName(null)}
        onOk={handleDelete}
        okText="删除"
        destroyOnHidden
        transitionName=""
        maskTransitionName=""
        okButtonProps={{ danger: true }}
        cancelText="取消"
      >
        <p className="text-[13px] text-fg-dim">确定删除画像「{deleteName}」吗？此操作不可撤销。</p>
      </Modal>
    </div>
  );
}
