import { useEffect, useState } from "react";
import { Input, Modal, Typography } from "antd";
import { useToast } from "./Toast";
import { app } from "../lib/bridge";
import type { SkillCaptureInput } from "../lib/types";

const { Text } = Typography;

// SkillCaptureModal — 把一次成功对话沉淀为可复用技能：
// 预填任务/回答，用户补技能标识与描述后保存；后端写入 .gaea/skills
// 并镜像全局技能目录，成功后热加载引擎（可立即用 /技能名 调用）。
export function SkillCaptureModal({
  open,
  task,
  solution,
  onClose,
}: {
  open: boolean;
  task: string;
  solution: string;
  onClose: () => void;
}) {
  const toast = useToast();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [taskDraft, setTaskDraft] = useState("");
  const [solutionDraft, setSolutionDraft] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (open) {
      setName("");
      setDescription("");
      setTaskDraft(task);
      setSolutionDraft(solution);
      setSaving(false);
    }
  }, [open, task, solution]);

  const save = async () => {
    const input: SkillCaptureInput = {
      name: name.trim(),
      description: description.trim(),
      task: taskDraft.trim(),
      solution: solutionDraft.trim(),
    };
    if (!input.name) {
      toast.show("请填写技能标识（如 weekly-report）", "warn");
      return;
    }
    if (!input.task || !input.solution) {
      toast.show("适用场景与操作步骤不能为空", "warn");
      return;
    }
    setSaving(true);
    try {
      const res = await app.CaptureSkill(input);
      toast.show(
        res.reloaded
          ? `技能已保存并热加载：/${res.name}（技能 ${res.skills} 个）`
          : `技能已保存：/${res.name}`,
      );
      onClose();
    } catch (e: unknown) {
      toast.show(String((e as Error)?.message ?? e), "warn");
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      title="沉淀为技能"
      open={open}
      onOk={() => void save()}
      onCancel={onClose}
      okText="保存为技能"
      cancelText="取消"
      confirmLoading={saving}
      width={640}
      destroyOnHidden
      transitionName=""
      maskTransitionName=""
    >
      <div style={{ display: "flex", flexDirection: "column", gap: 12, marginTop: 8 }}>
        <div>
          <Text strong>
            技能标识 <Text type="danger">*</Text>
          </Text>
          <Input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="如 weekly-report（字母/数字/_/-/.，字母开头）"
            style={{ marginTop: 4 }}
            onPressEnter={() => void save()}
          />
        </div>
        <div>
          <Text strong>一句话描述</Text>
          <Input
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            maxLength={120}
            placeholder="留空则取任务前 60 字"
            style={{ marginTop: 4 }}
          />
        </div>
        <div>
          <Text strong>适用场景</Text>
          <Input.TextArea
            value={taskDraft}
            onChange={(e) => setTaskDraft(e.target.value)}
            rows={2}
            style={{ marginTop: 4 }}
          />
        </div>
        <div>
          <Text strong>操作步骤（可编辑后再保存）</Text>
          <Input.TextArea
            value={solutionDraft}
            onChange={(e) => setSolutionDraft(e.target.value)}
            rows={6}
            style={{ marginTop: 4, fontFamily: "var(--ds-font-mono, monospace)" }}
          />
        </div>
        <Text type="secondary" style={{ fontSize: 12 }}>
          保存后写入 .gaea/skills 并镜像全局技能目录，之后可用 /{name.trim() || "技能名"} 调用。
        </Text>
      </div>
    </Modal>
  );
}
