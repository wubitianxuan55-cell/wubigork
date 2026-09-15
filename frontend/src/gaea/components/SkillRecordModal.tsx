import { useEffect, useState } from "react";
import { Button, Input, Modal, Typography } from "antd";
import { useToast } from "./Toast";
import { app } from "../lib/bridge";
import type { SkillDraft } from "../lib/types";

const { Text } = Typography;

// SkillRecordModal — 会话录制技能（阶段七 7.2-1 演示式录制，对标 Record a
// Skill 形态）：把当前会话的多轮回放（含工具调用）经 LLM 蒸馏成结构化草稿
// （名称/描述/适用场景/步骤/注意事项），用户审阅修订后保存——与单轮「沉淀
// 为技能」走同一落盘通道（同名覆盖 + 全局镜像 + 热加载）。LLM 只产草稿，
// 落盘必经用户确认。
export function SkillRecordModal({ open, onClose }: { open: boolean; onClose: () => void }) {
  const toast = useToast();
  const [drafting, setDrafting] = useState(false);
  const [draft, setDraft] = useState<SkillDraft | null>(null);
  const [replay, setReplay] = useState("");
  const [preview, setPreview] = useState("");
  const [saving, setSaving] = useState(false);

  const reset = () => {
    setDrafting(false);
    setDraft(null);
    setReplay("");
    setPreview("");
    setSaving(false);
  };

  const distill = async () => {
    setDrafting(true);
    try {
      const res = await app.SkillDraftFromSession();
      setDraft(res.draft);
      setReplay(res.replay);
      setPreview(res.preview);
    } catch (e: unknown) {
      toast.show(String((e as Error)?.message ?? e), "warn");
    } finally {
      setDrafting(false);
    }
  };

  useEffect(() => {
    if (open) {
      reset();
      void distill();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  const save = async () => {
    if (!draft) return;
    const payload: SkillDraft = {
      ...draft,
      steps: (draft.steps ?? []).map((s) => s.trim()).filter(Boolean),
      cautions: (draft.cautions ?? []).map((s) => s.trim()).filter(Boolean),
    };
    if (!payload.name?.trim()) {
      toast.show("请填写技能标识（如 weekly-report）", "warn");
      return;
    }
    if (payload.steps.length === 0) {
      toast.show("至少保留一条操作步骤", "warn");
      return;
    }
    setSaving(true);
    try {
      const res = await app.SkillDraftSave(payload);
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

  const patch = (p: Partial<SkillDraft>) => setDraft((d) => (d ? { ...d, ...p } : d));

  return (
    <Modal
      title="从会话录制技能"
      open={open}
      onOk={() => void save()}
      onCancel={onClose}
      okText="保存为技能"
      cancelText="取消"
      confirmLoading={saving}
      width={720}
      okButtonProps={{ disabled: drafting || !draft }}
    >
      {drafting && <Text type="secondary">正在蒸馏本会话…</Text>}
      {!drafting && !draft && (
        <div>
          <Text type="secondary">尚未生成草稿。</Text>
          <Button size="small" className="ml-2" onClick={() => void distill()}>
            重新蒸馏
          </Button>
        </div>
      )}
      {draft && (
        <div className="flex flex-col gap-3">
          <div className="flex gap-2">
            <div className="flex-1">
              <Text type="secondary" className="block mb-1">
                技能标识（字母开头，字母/数字/-/_）
              </Text>
              <Input
                value={draft.name}
                onChange={(e) => patch({ name: e.target.value })}
                placeholder="weekly-report"
                data-testid="record-name"
              />
            </div>
            <div className="flex-1">
              <Text type="secondary" className="block mb-1">
                一句话用途
              </Text>
              <Input
                value={draft.description}
                onChange={(e) => patch({ description: e.target.value })}
                placeholder="生成三段式周报"
                data-testid="record-desc"
              />
            </div>
          </div>
          <div>
            <Text type="secondary" className="block mb-1">
              适用场景
            </Text>
            <Input.TextArea
              value={draft.scenario}
              onChange={(e) => patch({ scenario: e.target.value })}
              rows={2}
              data-testid="record-scenario"
            />
          </div>
          <div>
            <Text type="secondary" className="block mb-1">
              操作步骤（每行一条，可增删改——保存以你编辑的内容为准）
            </Text>
            <Input.TextArea
              value={(draft.steps ?? []).join("\n")}
              onChange={(e) => patch({ steps: e.target.value.split("\n") })}
              rows={6}
              data-testid="record-steps"
            />
          </div>
          <div>
            <Text type="secondary" className="block mb-1">
              注意事项（每行一条，可留空）
            </Text>
            <Input.TextArea
              value={(draft.cautions ?? []).join("\n")}
              onChange={(e) => patch({ cautions: e.target.value.split("\n") })}
              rows={2}
              data-testid="record-cautions"
            />
          </div>
          <details>
            <summary className="cursor-pointer text-fg-dim text-xs">SKILL.md 预览（蒸馏时生成）</summary>
            <pre className="mt-2 max-h-56 overflow-auto text-xs bg-bg-soft rounded-md p-2" data-testid="record-preview">
              {preview}
            </pre>
          </details>
          <details>
            <summary className="cursor-pointer text-fg-dim text-xs">会话回放（蒸馏依据）</summary>
            <pre className="mt-2 max-h-40 overflow-auto text-xs bg-bg-soft rounded-md p-2">{replay}</pre>
          </details>
        </div>
      )}
    </Modal>
  );
}
