// Edit — 造价库库页编辑面（2026-11-27）：直接修正录入数据，不用绕回造价模式。
// 四处编辑：清单行（字段 + 定额挂接）、定额（字段 + 含量行）、工料机（主数据 +
// 按期调价）、项目封面（口径 + 费率）。全部走 store 的既有保存入口
// （Workcost*Save / SetPrice / Attach / Detach），语义索引打脏随写自动生效。
// 规矩：规范值（专业 / 地区 / 期数 / 分类）只给库里已有的取值——新取值的归类
// 仍归造价模式的模型，这里只负责把录错的值改对。
import { useEffect, useMemo, useState } from "react";
import { Plus, Trash2, X } from "../../gaea/icons";
import { app } from "../../gaea/lib/bridge";
import type {
  WorkcostBillItem, WorkcostBillProject, WorkcostBillQuotaLink, WorkcostQuota,
  WorkcostQuotaItem, WorkcostResource,
} from "../../gaea/lib/types";
import { RESOURCE_KINDS, fmtMoney, fmtQty } from "./logic";

function message(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}

/** 编辑表单的一格：标签 + 控件 + 可选说明。 */
function Field({ label, hint, children }: { label: string; hint?: string; children: React.ReactNode }) {
  return (
    <label className="cl-edit-field">
      <span className="cl-edit-label">{label}</span>
      {children}
      {hint && <span className="cl-edit-hint">{hint}</span>}
    </label>
  );
}

/** 数字输入（空串合法——保存时按 0 处理或按必填校验）。 */
function NumInput({ value, onChange, step, testid }: {
  value: number;
  onChange: (n: number) => void;
  step?: string;
  testid?: string;
}) {
  return (
    <input className="cl-edit-input" type="number" step={step ?? "any"} min={0} value={Number.isFinite(value) ? String(value) : ""}
      data-testid={testid} onChange={(e) => onChange(e.target.value === "" ? 0 : Number(e.target.value))} />
  );
}

function EditActions({ busy, error, saveLabel, onSave, onCancel }: {
  busy: boolean;
  error: string;
  saveLabel: string;
  onSave: () => void;
  onCancel: () => void;
}) {
  return (
    <div className="cl-edit-actions">
      <button type="button" className="cl-btn cl-btn-primary" disabled={busy} data-testid="cost-edit-save" onClick={onSave}>
        {saveLabel}
      </button>
      <button type="button" className="cl-btn" disabled={busy} onClick={onCancel}><X size={12} aria-hidden />取消</button>
      {error && <span className="cl-warn-text" role="alert" data-testid="cost-edit-error">{error}</span>}
    </div>
  );
}

// ── 清单行编辑（字段 + 定额挂接）────────────────────────────────────

export function BillEdit({ item, specialtyOptions, onSaved, onCancel }: {
  item: WorkcostBillItem;
  /** 专业可选值（库里已有的规范值；不含「未标注」哨兵）。 */
  specialtyOptions: string[];
  onSaved: (msg: string) => void;
  onCancel: () => void;
}) {
  const [title, setTitle] = useState(item.title);
  const [unit, setUnit] = useState(item.unit);
  const [quantity, setQuantity] = useState(item.quantity);
  const [division, setDivision] = useState(item.division);
  const [feature, setFeature] = useState(item.feature);
  const [priceOverride, setPriceOverride] = useState(item.priceOverride);
  // 专业空=保持原值（SaveBillItem 的口径），所以下拉里保留「保持未标注」。
  const [specialty, setSpecialty] = useState(item.specialty ?? "");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  // 挂接定额：进编辑态即取（1:N 引用表为准）。quantity=0 表示跟随清单工程量。
  const [links, setLinks] = useState<WorkcostBillQuotaLink[] | null>(null);
  const [linkDraft, setLinkDraft] = useState<Record<number, number>>({});
  const [attachCode, setAttachCode] = useState("");
  const [attachQty, setAttachQty] = useState(0);
  const [linkBusy, setLinkBusy] = useState(false);
  const [linkError, setLinkError] = useState("");

  useEffect(() => {
    let alive = true;
    setLinks(null);
    void app.WorkcostBillQuotaLinks(item.id)
      .then((ls) => { if (alive) setLinks(ls ?? []); })
      .catch((e: unknown) => { if (alive) { setLinks([]); setLinkError(message(e)); } });
    return () => { alive = false; };
  }, [item.id]);

  const save = async () => {
    if (!title.trim()) {
      setError("清单项需要名称");
      return;
    }
    setBusy(true);
    setError("");
    try {
      await app.WorkcostBillItemSave({
        ...item, title: title.trim(), unit: unit.trim(), quantity, division: division.trim(),
        feature: feature.trim(), priceOverride, specialty: specialty.trim(),
      });
      onSaved(`已保存清单项「${title.trim()}」`);
      onCancel(); // 成功即退出编辑态（回到只读详情；页面静默重读跟在后面）
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  };

  // 挂接是即时动作（幂等 upsert / 解挂可重挂），不与字段保存绑在一起。
  const runLink = async (task: () => Promise<unknown>) => {
    setLinkBusy(true);
    setLinkError("");
    try {
      await task();
      setLinks((await app.WorkcostBillQuotaLinks(item.id)) ?? []);
    } catch (e) {
      setLinkError(message(e));
    } finally {
      setLinkBusy(false);
    }
  };

  return (
    <div className="cl-edit" data-testid="cost-bill-edit">
      <div className="cl-edit-grid">
        <Field label="名称"><input className="cl-edit-input" value={title} data-testid="cost-edit-bill-title" onChange={(e) => setTitle(e.target.value)} /></Field>
        <Field label="单位"><input className="cl-edit-input" value={unit} onChange={(e) => setUnit(e.target.value)} /></Field>
        <Field label="工程量"><NumInput value={quantity} onChange={setQuantity} testid="cost-edit-bill-quantity" /></Field>
        <Field label="分部"><input className="cl-edit-input" value={division} onChange={(e) => setDivision(e.target.value)} /></Field>
        <Field label="专业" hint="只给库里已有的叫法；空=保持未标注">
          <select className="cl-select" value={specialty} data-testid="cost-edit-bill-specialty" onChange={(e) => setSpecialty(e.target.value)}>
            <option value="">{specialty ? "（保持）" : "未标注"}</option>
            {specialtyOptions.map((s) => <option key={s} value={s}>{s}</option>)}
          </select>
        </Field>
        <Field label="手填单价" hint="0=不手填，按挂接定额现算"><NumInput value={priceOverride} onChange={setPriceOverride} /></Field>
      </div>
      <Field label="项目特征">
        <textarea className="cl-edit-input cl-edit-area" rows={2} value={feature} onChange={(e) => setFeature(e.target.value)} />
      </Field>
      <EditActions busy={busy} error={error} saveLabel="保存清单项" onSave={() => void save()} onCancel={onCancel} />

      <section className="cl-block">
        <div className="cl-block-head"><span className="cl-block-label">挂接定额</span><span className="cl-dim">{links ? `${links.length} 条` : "读取中…"}</span></div>
        {links && links.length === 0 && <p className="cl-note">没有挂接定额（手填项）。在下方按定额编码挂接。</p>}
        {links && links.length > 0 && (
          <table className="cl-table cl-table-inner" data-testid="cost-edit-links">
            <thead><tr><th>定额</th><th className="cl-num">工程量</th><th /></tr></thead>
            <tbody>
              {links.map((l) => (
                <tr key={l.id}>
                  <td className="cl-cell-title">
                    <span className="cl-code">{l.quotaCode}</span>
                    {l.quantity > 0 ? <span className="cl-dim"> 量 {fmtQty(l.quantity)}</span> : <span className="cl-dim">（跟随清单工程量）</span>}
                  </td>
                  <td className="cl-num">
                    <NumInput value={linkDraft[l.id] ?? l.quantity} onChange={(n) => setLinkDraft((d) => ({ ...d, [l.id]: n }))} testid={`cost-edit-link-qty-${l.quotaCode}`} />
                  </td>
                  <td>
                    <button type="button" className="cl-btn cl-btn-ghost" disabled={linkBusy}
                      onClick={() => void runLink(() => app.WorkcostBillQuotaAttach(item.id, l.quotaCode, linkDraft[l.id] ?? l.quantity))}>
                      更新
                    </button>
                    <button type="button" className="cl-btn cl-btn-danger-ghost" disabled={linkBusy} title="解挂这条定额（可重新挂接）"
                      onClick={() => void runLink(() => app.WorkcostBillQuotaDetach(item.id, l.id))}>
                      <Trash2 size={11} aria-hidden />解挂
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
        <div className="cl-edit-row">
          <input className="cl-edit-input" placeholder="定额编码，如 WP01" value={attachCode} aria-label="挂接定额编码"
            onChange={(e) => setAttachCode(e.target.value)} />
          <NumInput value={attachQty} onChange={setAttachQty} />
          <button type="button" className="cl-btn" disabled={linkBusy || !attachCode.trim()} data-testid="cost-edit-link-attach"
            onClick={() => void runLink(async () => {
              await app.WorkcostBillQuotaAttach(item.id, attachCode.trim(), attachQty);
              setAttachCode("");
              setAttachQty(0);
            })}>
            <Plus size={12} aria-hidden />挂接（0=跟随清单量）
          </button>
          {linkError && <span className="cl-warn-text" role="alert">{linkError}</span>}
        </div>
      </section>
    </div>
  );
}

// ── 定额编辑（字段 + 含量行）────────────────────────────────────────

export function QuotaEdit({ quota, specialtyOptions, chapterOptionsOf, resources, regionOptions, onSaved, onCancel }: {
  /** 带含量行的完整定额（QuotaDetail 的 full）。 */
  quota: WorkcostQuota;
  specialtyOptions: string[];
  /** 某专业下可选的分部（表单里换专业时随之换）。 */
  chapterOptionsOf: (specialty: string) => string[];
  /** 工料机全量（添加含量行时带出单位与行类）。 */
  resources: WorkcostResource[];
  /** 库里已有的地区取值（规范值只给已有叫法）。 */
  regionOptions: string[];
  onSaved: (msg: string) => void;
  onCancel: () => void;
}) {
  const [title, setTitle] = useState(quota.title);
  const [specialty, setSpecialty] = useState(quota.specialty);
  const [chapter, setChapter] = useState(quota.chapter);
  const [region, setRegion] = useState(quota.region);
  const [unit, setUnit] = useState(quota.unit);
  const [note, setNote] = useState(quota.note);
  const [items, setItems] = useState<WorkcostQuotaItem[]>(() => (quota.items ?? []).map((it) => ({ ...it })));
  const [addCode, setAddCode] = useState("");
  const [addQty, setAddQty] = useState(0);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const addMatch = useMemo(() => resources.find((r) => r.code === addCode.trim()), [resources, addCode]);

  const save = async () => {
    if (!title.trim()) {
      setError("定额需要名称");
      return;
    }
    setBusy(true);
    setError("");
    try {
      await app.WorkcostQuotaSave({
        ...quota, title: title.trim(), specialty: specialty.trim(), chapter: chapter.trim(),
        region: region.trim(), unit: unit.trim(), note: note.trim(), items,
      });
      onSaved(`已保存定额「${title.trim()}」`);
      onCancel(); // 成功即退出编辑态（整组替换后回只读详情，页面静默重读跟在后面）
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  };

  const addItem = () => {
    const code = addCode.trim();
    if (!code) return;
    if (items.some((it) => it.resourceCode === code)) {
      setError(`含量行里已有 ${code}，直接改那行的含量`);
      return;
    }
    setItems((list) => [...list, {
      quotaCode: quota.code, resourceCode: code,
      kind: (addMatch?.kind ?? "材料") as WorkcostQuotaItem["kind"],
      title: addMatch?.title ?? code, spec: addMatch?.spec ?? "", unit: addMatch?.unit ?? "",
      quantity: addQty, resourcePrice: 0, lossRate: 0,
    }]);
    setAddCode("");
    setAddQty(0);
  };

  return (
    <div className="cl-edit" data-testid="cost-quota-edit">
      <div className="cl-edit-grid">
        <Field label="名称"><input className="cl-edit-input" value={title} data-testid="cost-edit-quota-title" onChange={(e) => setTitle(e.target.value)} /></Field>
        <Field label="计量单位"><input className="cl-edit-input" value={unit} onChange={(e) => setUnit(e.target.value)} /></Field>
        <Field label="专业" hint="只给库里已有的叫法">
          <select className="cl-select" value={specialty} data-testid="cost-edit-quota-specialty"
            onChange={(e) => {
              setSpecialty(e.target.value);
              // 换专业后原分部多半不归属了：不在新专业的分部面里就落回未归。
              if (!chapterOptionsOf(e.target.value).includes(chapter)) setChapter("");
            }}>
            <option value="">未标注</option>
            {specialtyOptions.map((s) => <option key={s} value={s}>{s}</option>)}
          </select>
        </Field>
        <Field label="分部" hint="随专业走；没有合适的就保持未归">
          <select className="cl-select" value={chapter} onChange={(e) => setChapter(e.target.value)}>
            <option value="">未归分部</option>
            {chapterOptionsOf(specialty).map((c) => <option key={c} value={c}>{c}</option>)}
          </select>
        </Field>
        <Field label="地区" hint="省级行政区（只到省）；随上游清单项目导入带上，没有合适的就留空">
          <select className="cl-select" value={region} onChange={(e) => setRegion(e.target.value)}>
            <option value="">未标地区</option>
            {regionOptions.map((r) => <option key={r} value={r}>{r}</option>)}
          </select>
        </Field>
      </div>
      <Field label="备注">
        <textarea className="cl-edit-input cl-edit-area" rows={2} value={note} onChange={(e) => setNote(e.target.value)} />
      </Field>

      <section className="cl-block">
        <div className="cl-block-head"><span className="cl-block-label">含量行（每单位消耗）</span><span className="cl-dim">{items.length} 行</span></div>
        {items.length > 0 && (
          <table className="cl-table cl-table-inner" data-testid="cost-edit-items">
            <thead><tr><th>行类</th><th>资源</th><th>单位</th><th className="cl-num">含量</th><th className="cl-num">损耗率%</th><th /></tr></thead>
            <tbody>
              {items.map((it, i) => (
                <tr key={`${it.resourceCode}:${i}`}>
                  <td>{it.kind}</td>
                  <td className="cl-cell-title"><span className="cl-code">{it.resourceCode}</span> {it.title}</td>
                  <td>{it.unit || "—"}</td>
                  <td className="cl-num"><NumInput value={it.quantity} onChange={(n) => setItems((list) => list.map((x, j) => (j === i ? { ...x, quantity: n } : x)))} testid={`cost-edit-item-qty-${it.resourceCode}`} /></td>
                  <td className="cl-num"><NumInput value={(it.lossRate ?? 0) * 100} onChange={(n) => setItems((list) => list.map((x, j) => (j === i ? { ...x, lossRate: n / 100 } : x)))} /></td>
                  <td>
                    <button type="button" className="cl-btn cl-btn-danger-ghost" title="删除这条含量行"
                      onClick={() => setItems((list) => list.filter((_, j) => j !== i))}>
                      <Trash2 size={11} aria-hidden />删行
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
        <div className="cl-edit-row">
          <input className="cl-edit-input" placeholder="工料机编码，如 C20" value={addCode} aria-label="添加含量行的工料机编码" list="cl-edit-res-codes"
            onChange={(e) => setAddCode(e.target.value)} />
          <datalist id="cl-edit-res-codes">
            {resources.slice(0, 500).map((r) => <option key={r.code} value={r.code}>{r.title}</option>)}
          </datalist>
          <NumInput value={addQty} onChange={setAddQty} />
          <button type="button" className="cl-btn" disabled={!addCode.trim()} data-testid="cost-edit-item-add" onClick={addItem}>
            <Plus size={12} aria-hidden />加行
          </button>
        </div>
        <p className="cl-edit-hint">整组替换落库：没列在这儿的旧行会删掉；新行缺的行类 / 单位 / 单价按库里的工料机补齐。</p>
      </section>
      <EditActions busy={busy} error={error} saveLabel="保存定额" onSave={() => void save()} onCancel={onCancel} />
    </div>
  );
}

// ── 工料机编辑（主数据 + 按期调价）──────────────────────────────────

export function ResourceEdit({ resource, regionOptions, periodOptions, categoryOptions, onSaved, onCancel }: {
  resource: WorkcostResource;
  /** 库里已有的地区 / 期数取值。 */
  regionOptions: string[];
  periodOptions: string[];
  /** 该类别下已有的分类路径。 */
  categoryOptions: string[];
  onSaved: (msg: string) => void;
  onCancel: () => void;
}) {
  const [kind, setKind] = useState(resource.kind);
  const [title, setTitle] = useState(resource.title);
  const [spec, setSpec] = useState(resource.spec);
  const [unit, setUnit] = useState(resource.unit);
  const [categoryPath, setCategoryPath] = useState(resource.categoryPath);
  const [region, setRegion] = useState(resource.region);
  const [priceDate, setPriceDate] = useState(resource.priceDate);
  const [lossRate, setLossRate] = useState(resource.lossRate * 100);
  const [note, setNote] = useState(resource.note);
  const [basePrice, setBasePrice] = useState(resource.basePrice);
  const [currentPrice, setCurrentPrice] = useState(resource.currentPrice);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  // 按期调价是独立动作（写调价历史并推进现行价），与主数据保存分开。
  const [pPrice, setPPrice] = useState(0);
  const [pPeriod, setPPeriod] = useState("");
  const [pRegion, setPRegion] = useState(resource.region);
  const [pType, setPType] = useState("信息价");
  const [priceBusy, setPriceBusy] = useState(false);

  const save = async () => {
    if (!title.trim()) {
      setError("工料机需要名称");
      return;
    }
    setBusy(true);
    setError("");
    try {
      const saved = await app.WorkcostResourceSave({
        ...resource, kind: kind as WorkcostResource["kind"], title: title.trim(), spec: spec.trim(),
        unit: unit.trim(), categoryPath: categoryPath.trim(), region: region.trim(), priceDate: priceDate.trim(),
        lossRate: lossRate / 100, note: note.trim(), basePrice, currentPrice,
      });
      onSaved(`已保存工料机「${saved.title}」`);
      onCancel(); // 成功即退出编辑态（回只读详情；页面静默重读跟在后面）
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  };

  const setPrice = async () => {
    if (pPrice <= 0) {
      setError("调价需要正的价格");
      return;
    }
    setPriceBusy(true);
    setError("");
    try {
      const saved = await app.WorkcostResourceSetPrice(resource.id, pPrice, pPeriod.trim(), pRegion.trim(), pType.trim(), "造价库界面", "");
      onSaved(`已调价：${saved.title} 现行价 ${fmtMoney(saved.currentPrice)}`);
      // 调价是登记动作：留在编辑态，主数据的未保存改动不动。
    } catch (e) {
      setError(message(e));
    } finally {
      setPriceBusy(false);
    }
  };

  return (
    <div className="cl-edit" data-testid="cost-resource-edit">
      <div className="cl-edit-grid">
        <Field label="类别">
          <select className="cl-select" value={kind} data-testid="cost-edit-res-kind" onChange={(e) => setKind(e.target.value)}>
            {RESOURCE_KINDS.map((k) => <option key={k} value={k}>{k}</option>)}
          </select>
        </Field>
        <Field label="名称"><input className="cl-edit-input" value={title} data-testid="cost-edit-res-title" onChange={(e) => setTitle(e.target.value)} /></Field>
        <Field label="规格"><input className="cl-edit-input" value={spec} onChange={(e) => setSpec(e.target.value)} /></Field>
        <Field label="单位"><input className="cl-edit-input" value={unit} onChange={(e) => setUnit(e.target.value)} /></Field>
        <Field label="分类" hint="该类别下已有的分类；未细分就保持空">
          <select className="cl-select" value={categoryPath} onChange={(e) => setCategoryPath(e.target.value)}>
            <option value="">未细分</option>
            {categoryOptions.map((c) => <option key={c} value={c}>{c}</option>)}
          </select>
        </Field>
        <Field label="地区" hint="省级行政区（只到省）；工料机按省分行，改动会跟着身份走">
          <select className="cl-select" value={region} onChange={(e) => setRegion(e.target.value)}>
            <option value="">未标注</option>
            {regionOptions.map((r) => <option key={r} value={r}>{r}</option>)}
          </select>
        </Field>
        <Field label="价格期数">
          <select className="cl-select" value={priceDate} onChange={(e) => setPriceDate(e.target.value)}>
            <option value="">无期</option>
            {periodOptions.map((p) => <option key={p} value={p}>{p}</option>)}
          </select>
        </Field>
        <Field label="损耗率%"><NumInput value={lossRate} onChange={setLossRate} step="0.1" /></Field>
        <Field label="基准价"><NumInput value={basePrice} onChange={setBasePrice} /></Field>
        <Field label="现行价"><NumInput value={currentPrice} onChange={setCurrentPrice} testid="cost-edit-res-price" /></Field>
      </div>
      <Field label="备注">
        <textarea className="cl-edit-input cl-edit-area" rows={2} value={note} onChange={(e) => setNote(e.target.value)} />
      </Field>
      <EditActions busy={busy} error={error} saveLabel="保存工料机" onSave={() => void save()} onCancel={onCancel} />

      <section className="cl-block">
        <div className="cl-block-head"><span className="cl-block-label">按期调价</span><span className="cl-dim">写调价历史并推进现行价</span></div>
        <div className="cl-edit-row">
          <NumInput value={pPrice} onChange={setPPrice} testid="cost-edit-res-setprice-price" />
          <select className="cl-select" value={pPeriod} aria-label="调价期数" onChange={(e) => setPPeriod(e.target.value)}>
            <option value="">期数（归一 YYYY-MM，可手填）</option>
            {periodOptions.map((p) => <option key={p} value={p}>{p}</option>)}
          </select>
          <select className="cl-select" value={pRegion} aria-label="调价地区" onChange={(e) => setPRegion(e.target.value)}>
            <option value="">地区</option>
            {regionOptions.map((r) => <option key={r} value={r}>{r}</option>)}
          </select>
          <input className="cl-edit-input" value={pType} aria-label="价格口径" onChange={(e) => setPType(e.target.value)} />
          <button type="button" className="cl-btn" disabled={priceBusy || pPrice <= 0} data-testid="cost-edit-res-setprice" onClick={() => void setPrice()}>登记调价</button>
        </div>
      </section>
    </div>
  );
}

// ── 项目封面编辑（口径 + 费率）──────────────────────────────────────

export function ProjectEdit({ project, specialtyOptions, regionOptions, periodOptions, onSaved, onCancel }: {
  project: WorkcostBillProject;
  specialtyOptions: string[];
  regionOptions: string[];
  periodOptions: string[];
  onSaved: (msg: string) => void;
  onCancel: () => void;
}) {
  const [specialty, setSpecialty] = useState(project.specialty);
  const [region, setRegion] = useState(project.region);
  const [pricePeriod, setPricePeriod] = useState(project.pricePeriod);
  const [managementRate, setManagementRate] = useState(project.managementRate * 100);
  const [regulatoryRate, setRegulatoryRate] = useState(project.regulatoryRate * 100);
  const [profitRate, setProfitRate] = useState(project.profitRate * 100);
  const [taxRate, setTaxRate] = useState(project.taxRate * 100);
  const [profitIncludesRegulatory, setProfitIncludesRegulatory] = useState(project.profitIncludesRegulatory);
  const [controlPrice, setControlPrice] = useState(project.controlPrice);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const saveProfile = async () => {
    setBusy(true);
    setError("");
    try {
      await app.WorkcostBillProjectProfileSave(project.id, specialty.trim(), region.trim(), pricePeriod.trim());
      onSaved("已保存项目口径");
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  };

  const saveRates = async () => {
    setBusy(true);
    setError("");
    try {
      await app.WorkcostBillProjectRatesSave(project.id, {
        managementRate: managementRate / 100, regulatoryRate: regulatoryRate / 100,
        profitRate: profitRate / 100, taxRate: taxRate / 100,
      }, profitIncludesRegulatory, controlPrice);
      onSaved("已保存项目费率");
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="cl-edit" data-testid="cost-project-edit">
      <section className="cl-block">
        <div className="cl-block-head"><span className="cl-block-label">口径（规范值，空=保持原值）</span></div>
        <div className="cl-edit-grid">
          <Field label="专业">
            <select className="cl-select" value={specialty} data-testid="cost-edit-proj-specialty" onChange={(e) => setSpecialty(e.target.value)}>
              <option value="">{specialty ? "（保持）" : "未标注"}</option>
              {specialtyOptions.map((s) => <option key={s} value={s}>{s}</option>)}
            </select>
          </Field>
          <Field label="地区" hint="省级行政区（只到省）">
            <select className="cl-select" value={region} onChange={(e) => setRegion(e.target.value)}>
              <option value="">{region ? "（保持）" : "未标注"}</option>
              {regionOptions.map((r) => <option key={r} value={r}>{r}</option>)}
            </select>
          </Field>
          <Field label="计价期">
            <select className="cl-select" value={pricePeriod} onChange={(e) => setPricePeriod(e.target.value)}>
              <option value="">{pricePeriod ? "（保持）" : "未标注"}</option>
              {periodOptions.map((p) => <option key={p} value={p}>{p}</option>)}
            </select>
          </Field>
        </div>
        <div className="cl-edit-actions">
          <button type="button" className="cl-btn cl-btn-primary" disabled={busy} data-testid="cost-edit-proj-profile-save" onClick={() => void saveProfile()}>保存口径</button>
        </div>
      </section>
      <section className="cl-block">
        <div className="cl-block-head"><span className="cl-block-label">取费（0=不计取）</span></div>
        <div className="cl-edit-grid">
          <Field label="企业管理费%"><NumInput value={managementRate} onChange={setManagementRate} step="0.1" /></Field>
          <Field label="规费%"><NumInput value={regulatoryRate} onChange={setRegulatoryRate} step="0.1" /></Field>
          <Field label="利润%"><NumInput value={profitRate} onChange={setProfitRate} step="0.1" /></Field>
          <Field label="增值税%"><NumInput value={taxRate} onChange={setTaxRate} step="0.1" /></Field>
          <Field label="控制价"><NumInput value={controlPrice} onChange={setControlPrice} /></Field>
        </div>
        <label className="cl-edit-field cl-edit-check">
          <input type="checkbox" checked={profitIncludesRegulatory} onChange={(e) => setProfitIncludesRegulatory(e.target.checked)} />
          <span>利润以「直接费+规费」为基数</span>
        </label>
        <div className="cl-edit-actions">
          <button type="button" className="cl-btn cl-btn-primary" disabled={busy} data-testid="cost-edit-proj-rates-save" onClick={() => void saveRates()}>保存费率</button>
          <button type="button" className="cl-btn" disabled={busy} onClick={onCancel}><X size={12} aria-hidden />取消</button>
          {error && <span className="cl-warn-text" role="alert" data-testid="cost-edit-error">{error}</span>}
        </div>
      </section>
    </div>
  );
}
