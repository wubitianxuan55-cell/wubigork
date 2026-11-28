// Maintain — 造价库的两个小维护面（2026-11-28）：
//   · 定额地区回填：定额地区与上游清单项目挂钩，但挂钩只在项目导入时写进新建的定额；
//     导入前就在库里的地区一直空着。这里先预览（只被一个省的项目引用的才回填），确认后落库。
//   · 检索别名：行话 / 缩写 ↔ 标准说法（「砼」对「混凝土」），搜索时双向展开。别名存字典
//     （kind=alias），模型经 cost_library 的 dict 也能维护；这里只管看和改。
import { useEffect, useState } from "react";
import { Modal } from "antd";
import { Plus, Trash2 } from "../../gaea/icons";
import { app } from "../../gaea/lib/bridge";
import type { WorkcostDictItem, WorkcostQuotaRegionFix } from "../../gaea/lib/types";

function message(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}

function countBy(list: { key: string }[]): { key: string; count: number }[] {
  const counts = new Map<string, number>();
  for (const it of list) counts.set(it.key, (counts.get(it.key) ?? 0) + 1);
  return [...counts.entries()].map(([key, count]) => ({ key, count })).sort((a, b) => b.count - a.count);
}

export function RegionBackfillDialog({ open, onClose, onDone }: {
  open: boolean;
  onClose: () => void;
  /** 落库后的回执（提示 + 静默重读）。 */
  onDone: (msg: string) => void;
}) {
  const [phase, setPhase] = useState<"loading" | "ready" | "failed" | "applying" | "done">("loading");
  const [plan, setPlan] = useState<WorkcostQuotaRegionFix[]>([]);
  const [error, setError] = useState("");
  const [applied, setApplied] = useState(0);

  // 打开即出预览（干跑，不写库）。
  useEffect(() => {
    if (!open) return;
    let alive = true;
    setPhase("loading");
    setError("");
    setApplied(0);
    if (typeof app.WorkcostQuotaRegionBackfill !== "function") {
      setPhase("failed");
      setError("当前版本的后端没有定额地区回填");
      return;
    }
    void app.WorkcostQuotaRegionBackfill(true)
      .then((p) => { if (alive) { setPlan(p ?? []); setPhase("ready"); } })
      .catch((e: unknown) => { if (alive) { setError(message(e)); setPhase("failed"); } });
    return () => { alive = false; };
  }, [open]);

  const fillable = plan.filter((f) => !!f.region);
  const skipped = plan.filter((f) => !f.region);

  const apply = async () => {
    setPhase("applying");
    try {
      const res = await app.WorkcostQuotaRegionBackfill(false);
      const n = (res ?? []).filter((f) => f.applied).length;
      setApplied(n);
      setPhase("done");
      onDone(`已回填 ${n} 条定额的地区`);
    } catch (e) {
      setError(message(e));
      setPhase("failed");
    }
  };

  return (
    <Modal
      title="按上游清单项目回填定额地区"
      open={open}
      onCancel={() => { if (phase !== "applying") onClose(); }}
      footer={[
        <button key="close" type="button" className="cl-btn" disabled={phase === "applying"}
          onClick={onClose}>{phase === "done" ? "关闭" : "取消"}</button>,
        phase === "ready" && fillable.length > 0 && (
          <button key="apply" type="button" className="cl-btn cl-btn-primary" data-testid="cost-region-apply"
            onClick={() => void apply()}>写入 {fillable.length} 条</button>
        ),
      ]}
      closable={phase !== "applying"}
      maskClosable={phase !== "applying"}
      destroyOnHidden
      transitionName=""
      maskTransitionName=""
      width={560}
    >
      <div className="cl-dialog" data-testid="cost-region-dialog">
        {phase === "loading" && <p>正在按清单引用关系核对…</p>}
        {phase === "failed" && <p className="cl-dialog-error" role="alert">{error}</p>}
        {(phase === "ready" || phase === "applying") && (
          <>
            {plan.length === 0 ? (
              <p>没有还没标地区的定额，不用回填。</p>
            ) : (
              <>
                <p>
                  没标地区的定额共 {plan.length} 条。
                  {fillable.length > 0
                    ? <>其中只被<b>一个省</b>的项目引用的 <b>{fillable.length}</b> 条可以回填：
                        {countBy(fillable.map((f) => ({ key: f.region ?? "" }))).map((c) => ` ${c.key} ${c.count} 条`).join("，")}。</>
                    : "没有被单个省的项目引用的，回填不了。"}
                </p>
                {skipped.length > 0 && (
                  <p className="cl-note">
                    另有 {skipped.length} 条不动：
                    {countBy(skipped.map((f) => ({ key: f.reason ?? "" }))).map((c) => ` ${c.key}（${c.count} 条）`).join("；")}。
                    这些可以在定额详情里手标。
                  </p>
                )}
                {fillable.length > 0 && (
                  <details>
                    <summary>看可回填的定额（前 50 条）</summary>
                    <ul className="cl-batch-failures">
                      {fillable.slice(0, 50).map((f) => (
                        <li key={f.code}><span className="cl-code">{f.code}</span> {f.title} → {f.region}</li>
                      ))}
                    </ul>
                  </details>
                )}
              </>
            )}
          </>
        )}
        {phase === "done" && (
          <p>已写入 {applied} 条。列表会跟着刷新；没回填的那些地区仍是空的，可在定额详情里手标。</p>
        )}
      </div>
    </Modal>
  );
}

export function AliasDialog({ open, aliases, onClose, onChanged }: {
  open: boolean;
  /** 字典里的检索别名（name=别名、parent=标准说法）。 */
  aliases: WorkcostDictItem[];
  onClose: () => void;
  /** 增删之后让页面重读字典。 */
  onChanged: () => Promise<void> | void;
}) {
  const [alias, setAlias] = useState("");
  const [canon, setCanon] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const add = async () => {
    if (!alias.trim() || !canon.trim()) {
      setError("别名和它对应的标准说法都要填");
      return;
    }
    setBusy(true);
    setError("");
    try {
      await app.WorkcostDictSet({ id: 0, kind: "alias", name: alias.trim(), parent: canon.trim() }, false);
      setAlias("");
      setCanon("");
      await onChanged();
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  };
  const remove = async (item: WorkcostDictItem) => {
    setBusy(true);
    setError("");
    try {
      await app.WorkcostDictSet(item, true);
      await onChanged();
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal
      title="检索别名（行话 ↔ 标准说法）"
      open={open}
      onCancel={() => { if (!busy) onClose(); }}
      footer={<button type="button" className="cl-btn" disabled={busy} onClick={onClose}>关闭</button>}
      closable={!busy}
      maskClosable={!busy}
      destroyOnHidden
      transitionName=""
      maskTransitionName=""
      width={520}
    >
      <div className="cl-dialog" data-testid="cost-alias-dialog">
        <p className="cl-note">
          搜索框里出现别名时自动带上标准说法，反过来也一样（「砼」对「混凝土」两头都能搜到）。
          别名存在造价库字典里，造价模式的模型也能加。
        </p>
        {aliases.length === 0 ? (
          <p className="cl-dim">还没有别名。</p>
        ) : (
          <ul className="cl-alias-list" data-testid="cost-alias-list">
            {aliases.map((a) => (
              <li key={a.id}>
                <span><b>{a.name}</b> ↔ {a.parent}</span>
                <button type="button" className="cl-btn cl-btn-danger-ghost" disabled={busy} aria-label={`删除别名 ${a.name}`}
                  onClick={() => void remove(a)}>
                  <Trash2 size={11} aria-hidden />删除
                </button>
              </li>
            ))}
          </ul>
        )}
        <div className="cl-edit-row">
          <input className="cl-edit-input" aria-label="别名" placeholder="别名，如 砼" value={alias} disabled={busy}
            onChange={(e) => setAlias(e.target.value)} />
          <span className="cl-dim">对应</span>
          <input className="cl-edit-input" aria-label="标准说法" placeholder="标准说法，如 混凝土" value={canon} disabled={busy}
            onChange={(e) => setCanon(e.target.value)} onKeyDown={(e) => { if (e.key === "Enter") void add(); }} />
          <button type="button" className="cl-btn" disabled={busy} data-testid="cost-alias-add" onClick={() => void add()}>
            <Plus size={12} aria-hidden />添加
          </button>
        </div>
        {error && <p className="cl-dialog-error" role="alert">{error}</p>}
      </div>
    </Modal>
  );
}
