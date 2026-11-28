// BatchEdit — 勾选后批量改维度（2026-11-28）：专业 / 分部（清单、定额）、地区、分类、期数。
// 真实库里 330 条定额未归分部、340 条定额地区全空——靠模型逐条归类太慢，这里勾选 → 选维度 →
// 给值 → 逐行走既有的保存入口（Workcost*Save，规矩校验在存储层把关），逐行回报成败：
// 一行失败不影响其余（比如改地区后与已有行撞了身份）。不做事务：几百行里坏几行，留着
// 失败清单比整体回滚更有用。
import { useMemo, useState } from "react";
import { Modal } from "antd";
import { app } from "../../gaea/lib/bridge";
import type { WorkcostQuota, WorkcostResource } from "../../gaea/lib/types";
import { DictSelect } from "./DictSelect";
import type { EditDicts } from "./Edit";
import { RESOURCE_KINDS, type BillRow, type Tier } from "./logic";

/** 可批量改的维度。 */
export type BatchField = "specialty" | "division" | "chapter" | "region" | "category" | "period";

interface FieldMeta {
  value: BatchField;
  label: string;
  /** 字典种类；不给 = 自由文本（期数是 YYYY-MM 文本）。 */
  dictKind?: "specialty" | "region" | "category";
  hint: string;
}

const FIELDS: Record<Tier, FieldMeta[]> = {
  bill: [
    { value: "specialty", label: "专业", dictKind: "specialty", hint: "清单行的专业（字典里的值）" },
    { value: "division", label: "分部", hint: "自由文本；给已有写法可从候选里选，归并「A临建 / A施工准备与临建」这类不同写法就用它" },
  ],
  quota: [
    { value: "specialty", label: "专业", dictKind: "specialty", hint: "定额专业固定：换了专业，原分部不在新专业里的会落回「未归」" },
    { value: "chapter", label: "分部", hint: "专业内划分；给已有写法可从候选里选" },
    { value: "region", label: "地区", dictKind: "region", hint: "省级行政区（只到省）" },
  ],
  resource: [
    { value: "region", label: "地区", dictKind: "region", hint: "工料机按地区分行：改后与已有行身份（类别+名称+规格+单位+地区）相同的那条会失败" },
    { value: "category", label: "分类", dictKind: "category", hint: "分类路径（如 材料/土建材料/钢材），字典里没有的可以就地新增" },
    { value: "period", label: "价格期数", hint: "YYYY-MM，如 2026-12" },
  ],
};

function message(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}

interface Failure {
  label: string;
  reason: string;
}

export function BatchEditDialog({ tier, rows, open, dicts, chaptersOf, divisions, onClose, onDone }: {
  tier: Tier;
  /** 勾选的行（当前级的行模型）。 */
  rows: (BillRow | WorkcostQuota | WorkcostResource)[];
  open: boolean;
  dicts: EditDicts;
  /** 某专业下已有的分部（定额换专业时判断原分部还归不归）。 */
  chaptersOf: (specialty: string) => string[];
  /** 清单行已有的分部写法（候选）。 */
  divisions: string[];
  onClose: () => void;
  /** 跑完的回执（提示 + 静默重读）。 */
  onDone: (summary: string) => void;
}) {
  const [field, setField] = useState<BatchField>(FIELDS[tier][0]!.value);
  const [value, setValue] = useState("");
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<{ ok: number; failed: Failure[] } | null>(null);
  const meta = useMemo(() => FIELDS[tier].find((f) => f.value === field) ?? FIELDS[tier][0]!, [tier, field]);
  const tierLabel = tier === "bill" ? "清单项" : tier === "quota" ? "定额" : "工料机";

  const reset = () => {
    setValue("");
    setResult(null);
    setField(FIELDS[tier][0]!.value);
  };
  const close = () => {
    reset();
    onClose();
  };

  const run = async () => {
    if (!value.trim()) return;
    setBusy(true);
    const failed: Failure[] = [];
    let ok = 0;
    for (const row of rows) {
      const r = row as BillRow & WorkcostQuota & WorkcostResource;
      const label = `${r.code ?? ""} ${r.title ?? ""}`.trim();
      try {
        if (tier === "bill") {
          await app.WorkcostBillItemSave({ ...(row as BillRow), [field]: value.trim() });
        } else if (tier === "quota") {
          const full = await app.WorkcostQuotaGet((row as WorkcostQuota).code);
          if (!full) throw new Error("定额不在库里了（可能刚被删）");
          const next: WorkcostQuota = { ...full, [field]: value.trim() };
          if (field === "specialty" && !chaptersOf(value.trim()).includes(full.chapter ?? "")) next.chapter = "";
          await app.WorkcostQuotaSave(next);
        } else {
          await app.WorkcostResourceSave({ ...(row as WorkcostResource), [field === "period" ? "priceDate" : field]: value.trim() });
        }
        ok++;
      } catch (e) {
        failed.push({ label, reason: message(e) });
      }
    }
    setBusy(false);
    setResult({ ok, failed });
    if (failed.length === 0) {
      onDone(`已批量修改 ${ok} 条${tierLabel}`);
      onClose();
      reset();
    } else {
      onDone(`批量修改${tierLabel}：成功 ${ok} 条，失败 ${failed.length} 条`);
    }
  };

  const valueControl = () => {
    if (meta.dictKind === "specialty") {
      return <DictSelect label="改成的专业" testid="cost-batch-value" value={value} options={dicts.specialty ?? []}
        emptyLabel="选一个专业" add={dicts.add ? (name) => dicts.add!("specialty", name) : undefined} onChange={setValue} />;
    }
    if (meta.dictKind === "region") {
      return <DictSelect label="改成的地区" testid="cost-batch-value" value={value} options={dicts.region ?? []}
        emptyLabel="选一个省" add={dicts.add ? (name) => dicts.add!("region", name) : undefined} onChange={setValue} />;
    }
    if (meta.dictKind === "category") {
      // 分类是按类别分的：四类之下的分类路径都给出来。
      const options = RESOURCE_KINDS.flatMap((k) => dicts.categoryOf?.(k) ?? []);
      return <DictSelect label="改成的分类" testid="cost-batch-value" value={value} options={options}
        emptyLabel="选一个分类" add={dicts.add ? (name) => dicts.add!("category", name, name.includes("/") ? name.slice(0, name.lastIndexOf("/")) : "") : undefined}
        onChange={setValue} />;
    }
    const listId = `cl-batch-${field}`;
    const candidates = field === "chapter" ? chaptersOf("") : divisions;
    return (
      <>
        <input className="cl-edit-input" aria-label={`改成的${meta.label}`} data-testid="cost-batch-value" list={listId} value={value}
          placeholder={field === "period" ? "YYYY-MM，如 2026-12" : `填${meta.label}`} onChange={(e) => setValue(e.target.value)} />
        <datalist id={listId}>
          {candidates.map((c) => <option key={c} value={c} />)}
        </datalist>
      </>
    );
  };

  return (
    <Modal
      title={result ? `批量修改结果：成功 ${result.ok} 条，失败 ${result.failed.length} 条` : `批量修改 ${rows.length} 条${tierLabel}`}
      open={open}
      onCancel={() => { if (!busy) close(); }}
      onOk={() => void run()}
      okText={busy ? "修改中…" : `把这 ${rows.length} 条改掉`}
      cancelText="关闭"
      confirmLoading={busy}
      okButtonProps={{ danger: true, disabled: result !== null || !value.trim() }}
      closable={!busy}
      maskClosable={!busy}
      destroyOnHidden
      transitionName=""
      maskTransitionName=""
      width={520}
    >
      <div className="cl-dialog" data-testid="cost-batch-dialog">
        {result ? (
          <>
            <p>成功的 {result.ok} 条已落库；下面 {result.failed.length} 条没改，原因逐条列出（改好再来一遍即可）：</p>
            <ul className="cl-batch-failures" data-testid="cost-batch-failures">
              {result.failed.slice(0, 12).map((f) => <li key={f.label}><span className="cl-code">{f.label}</span>：{f.reason}</li>)}
              {result.failed.length > 12 && <li className="cl-dim">另有 {result.failed.length - 12} 条未列出</li>}
            </ul>
          </>
        ) : (
          <>
            <div className="cl-edit-grid">
              <label className="cl-edit-field">
                <span className="cl-edit-label">改哪一项</span>
                <select className="cl-select" aria-label="批量改哪一项" data-testid="cost-batch-field" value={field}
                  onChange={(e) => { setField(e.target.value as BatchField); setValue(""); }}>
                  {FIELDS[tier].map((f) => <option key={f.value} value={f.value}>{f.label}</option>)}
                </select>
              </label>
              <label className="cl-edit-field">
                <span className="cl-edit-label">改成</span>
                {valueControl()}
              </label>
            </div>
            <p className="cl-note">{meta.hint}。逐行保存、逐行回报：一行失败不影响其余。</p>
          </>
        )}
      </div>
    </Modal>
  );
}
