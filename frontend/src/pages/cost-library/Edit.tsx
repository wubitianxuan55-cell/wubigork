// Edit — 造价库库页编辑面（2026-11-27）：直接修正录入数据，不用绕回造价模式。
// 四处编辑：清单行（字段 + 定额挂接）、定额（字段 + 含量行）、工料机（主数据 +
// 按期调价）、项目封面（口径 + 费率）。全部走 store 的既有保存入口
// （Workcost*Save / SetPrice / Attach / Detach），语义索引打脏随写自动生效。
//
// 2026-11-28 索引与编辑优化：
//   - 规范值（专业 / 地区 / 分类）的下拉读字典（DictSelect），字典里规划了、还没有行在用
//     的值也列出来；没有合适的就地新增（走字典写入，规矩在存储层把关）。期数可以手填新月份。
//   - 未保存改动回报给页面（换行 / 换级前先问一句）；Ctrl+S 保存、Esc 取消。
//   - 主数据保存里改了现行价，顺手登记一条调价历史——不留「悄悄改价」的口子。
//   - 新建与复制另存（mode = create / duplicate）：清单行要选归属项目；定额另存要给新编码。
//   - 挂接定额、含量行加工料机用 SearchPicker 搜出来再选，不再只列前 500 条工料机。
import { useCallback, useEffect, useRef, useState } from "react";
import { Plus, Trash2, X } from "../../gaea/icons";
import { app } from "../../gaea/lib/bridge";
import type {
  WorkcostBillItem, WorkcostBillProject, WorkcostBillQuotaLink, WorkcostQuota,
  WorkcostQuotaItem, WorkcostResource,
} from "../../gaea/lib/types";
import { RESOURCE_KINDS, fmtMoney, fmtQty } from "./logic";
import { DictSelect } from "./DictSelect";
import { SearchPicker, type PickerOption } from "./SearchPicker";

function message(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}

/** 规范值字典接入：各维度的字典值与就地新增。 */
export interface EditDicts {
  specialty?: string[];
  region?: string[];
  /** 某类别下的分类路径（字典里该类别之下的全部）。 */
  categoryOf?: (kind: string) => string[];
  /** 就地新增一个字典值，成功返回库里最终的写法（地区会被归一成省级全称）。 */
  add?: (kind: "specialty" | "region" | "category", name: string, parent?: string) => Promise<string>;
}

/** edit=改这一条；create=新建（item 是空模板）；duplicate=以 item 为底另存一条。 */
export type EditMode = "edit" | "create" | "duplicate";

/** 并集去重：字典值在前，库里已有取值兜底（字典读不到时就只有后者）。 */
function unionOptions(a: string[] = [], b: string[] = []): string[] {
  return [...new Set([...a, ...b])];
}

/** 复制定额时建议的新编码：原码加 -2，已是「原码-N」的就再加一。 */
function suggestQuotaCode(code: string): string {
  const m = code.match(/^(.*)-(\d+)$/);
  return m ? `${m[1]}-${Number(m[2]) + 1}` : `${code}-2`;
}

/**
 * 编辑面的公共生命周期：把「有没有未保存改动」回报给页面（换行 / 换级前页面会先问一句），
 * Ctrl+S 保存、Esc 取消。
 */
function useEditLifecycle(opts: { dirty: boolean; onDirty?: (dirty: boolean) => void; save: () => void; cancel: () => void }): void {
  const onDirty = opts.onDirty;
  useEffect(() => { onDirty?.(opts.dirty); }, [opts.dirty, onDirty]);
  useEffect(() => () => onDirty?.(false), [onDirty]);
  const latest = useRef(opts);
  latest.current = opts;
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && !e.shiftKey && !e.altKey && e.key.toLowerCase() === "s") {
        e.preventDefault();
        latest.current.save();
      } else if (e.key === "Escape") {
        latest.current.cancel();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);
}

/** 取消前守一道：有未保存改动先问一句。 */
function guardCancel(dirty: boolean): boolean {
  return !dirty || window.confirm("这个编辑框里有未保存的改动，确定放弃吗？");
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

/** 期数输入：可手填新月份（统一存 YYYY-MM，库里归一；认不出的写法后端会拒），已有期数给出候选。 */
function PeriodInput({ label, value, onChange, options, testid }: {
  label: string;
  value: string;
  onChange: (next: string) => void;
  options: string[];
  testid?: string;
}) {
  const listId = `cl-periods-${testid ?? label}`;
  return (
    <>
      <input className="cl-edit-input" aria-label={label} data-testid={testid} list={listId} value={value}
        placeholder="YYYY-MM，如 2026-12" onChange={(e) => onChange(e.target.value)} />
      <datalist id={listId}>
        {options.map((p) => <option key={p} value={p} />)}
      </datalist>
    </>
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

export function BillEdit({ item, specialtyOptions, onSaved, onCancel, dicts, onDirty, mode = "edit", projects, onCreated }: {
  item: WorkcostBillItem;
  /** 专业可选值（库里已有的取值；字典接入后只是兜底）。 */
  specialtyOptions: string[];
  onSaved: (msg: string) => void;
  onCancel: () => void;
  dicts?: EditDicts;
  onDirty?: (dirty: boolean) => void;
  mode?: EditMode;
  /** 新建 / 复制时可选的归属项目（清单行必须归属一个项目）。 */
  projects?: { id: number; name: string }[];
  /** 新建成功：把落库后的那一行交给页面（定位过去）。 */
  onCreated?: (saved: WorkcostBillItem) => void;
}) {
  const creating = mode !== "edit";
  const [projectId, setProjectId] = useState(item.projectId);
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
    if (creating) {
      setLinks([]);
      return;
    }
    void app.WorkcostBillQuotaLinks(item.id)
      .then((ls) => { if (alive) setLinks(ls ?? []); })
      .catch((e: unknown) => { if (alive) { setLinks([]); setLinkError(message(e)); } });
    return () => { alive = false; };
  }, [item.id, creating]);

  const snapshot = JSON.stringify([projectId, title, unit, quantity, division, feature, priceOverride, specialty]);
  const [initial] = useState(snapshot);
  const dirty = creating || snapshot !== initial;

  const save = async () => {
    if (!title.trim()) {
      setError("清单项需要名称");
      return;
    }
    if (creating && projectId <= 0) {
      setError("清单项需要归属一个项目");
      return;
    }
    setBusy(true);
    setError("");
    try {
      const saved = await app.WorkcostBillItemSave({
        // 新建：不带上模板里的 id / 挂接（编码留空，后端自动给 M 序号）。
        ...(creating ? { ...item, id: 0, quotaCode: "", quotaCodes: [], linkQuantities: undefined, listPrice: 0, gcode: "" } : item),
        projectId: creating ? projectId : item.projectId,
        title: title.trim(), unit: unit.trim(), quantity, division: division.trim(),
        feature: feature.trim(), priceOverride, specialty: specialty.trim(),
      });
      if (creating) {
        onCreated?.(saved);
        onSaved(`已新建清单项「${saved.title || title.trim()}」`);
      } else {
        onSaved(`已保存清单项「${title.trim()}」`);
      }
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

  const requestCancel = () => {
    if (guardCancel(dirty)) onCancel();
  };
  useEditLifecycle({ dirty, onDirty, save: () => void save(), cancel: requestCancel });

  // 挑定额走混合检索（名称 / 意思相近都能搜到）；函数身份要稳定，SearchPicker 的防抖才不会每次渲染都重来。
  const itemSpecialty = item.specialty ?? "";
  const searchQuotas = useCallback(async (kw: string): Promise<PickerOption[]> =>
    ((await app.WorkcostQuotaList(itemSpecialty, kw)) ?? []).map((q) => ({ code: q.code, title: q.title })), [itemSpecialty]);

  return (
    <div className="cl-edit" data-testid="cost-bill-edit">
      <div className="cl-edit-grid">
        {creating && (
          <Field label="归属项目" hint="清单行是某个项目的使用属性">
            <select className="cl-select" aria-label="归属项目" data-testid="cost-edit-bill-project" value={projectId}
              onChange={(e) => setProjectId(Number(e.target.value))}>
              <option value={0}>选一个项目</option>
              {(projects ?? []).map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}
            </select>
          </Field>
        )}
        <Field label="名称"><input className="cl-edit-input" value={title} data-testid="cost-edit-bill-title" onChange={(e) => setTitle(e.target.value)} /></Field>
        <Field label="单位"><input className="cl-edit-input" value={unit} onChange={(e) => setUnit(e.target.value)} /></Field>
        <Field label="工程量"><NumInput value={quantity} onChange={setQuantity} testid="cost-edit-bill-quantity" /></Field>
        <Field label="分部"><input className="cl-edit-input" value={division} onChange={(e) => setDivision(e.target.value)} /></Field>
        <Field label="专业" hint="字典里的专业；空=保持未标注">
          <DictSelect label="清单专业" testid="cost-edit-bill-specialty" value={specialty}
            emptyLabel={specialty ? "（保持）" : "未标注"}
            options={unionOptions(dicts?.specialty, specialtyOptions)}
            add={dicts?.add ? (name) => dicts.add!("specialty", name) : undefined}
            onChange={setSpecialty} />
        </Field>
        <Field label="手填单价" hint="0=不手填，按挂接定额现算"><NumInput value={priceOverride} onChange={setPriceOverride} /></Field>
      </div>
      <Field label="项目特征">
        <textarea className="cl-edit-area cl-edit-input" rows={2} value={feature} onChange={(e) => setFeature(e.target.value)} />
      </Field>
      <EditActions busy={busy} error={error} saveLabel={creating ? "新建清单项" : "保存清单项"} onSave={() => void save()} onCancel={requestCancel} />

      {!creating && (
        <section className="cl-block">
          <div className="cl-block-head"><span className="cl-block-label">挂接定额</span><span className="cl-dim">{links ? `${links.length} 条` : "读取中…"}</span></div>
          {links && links.length === 0 && <p className="cl-note">没有挂接定额（手填项）。在下方搜定额挂接，或直接填编码。</p>}
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
            <SearchPicker label="挂接定额编码" placeholder="搜定额名称，或直接填编码" text={attachCode} onText={setAttachCode}
              search={searchQuotas} onPick={setAttachCode} />
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
      )}
    </div>
  );
}

// ── 定额编辑（字段 + 含量行）────────────────────────────────────────

export function QuotaEdit({ quota, specialtyOptions, chapterOptionsOf, resources, regionOptions, onSaved, onCancel, dicts, onDirty, mode = "edit", onCreated }: {
  /** 带含量行的完整定额（QuotaDetail 的 full）。 */
  quota: WorkcostQuota;
  specialtyOptions: string[];
  /** 某专业下可选的分部（表单里换专业时随之换）。 */
  chapterOptionsOf: (specialty: string) => string[];
  /** 工料机全量（添加含量行时带出单位与行类）。 */
  resources: WorkcostResource[];
  /** 库里已有的地区取值（字典接入后只是兜底）。 */
  regionOptions: string[];
  onSaved: (msg: string) => void;
  onCancel: () => void;
  dicts?: EditDicts;
  onDirty?: (dirty: boolean) => void;
  mode?: EditMode;
  onCreated?: (saved: WorkcostQuota) => void;
}) {
  const creating = mode !== "edit";
  const [code, setCode] = useState(mode === "duplicate" ? suggestQuotaCode(quota.code) : quota.code);
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

  const snapshot = JSON.stringify([code, title, specialty, chapter, region, unit, note, items]);
  const [initial] = useState(snapshot);
  const dirty = creating || snapshot !== initial;

  const save = async () => {
    if (!title.trim()) {
      setError("定额需要名称");
      return;
    }
    const finalCode = creating ? code.trim() : quota.code;
    if (creating && !finalCode) {
      setError("新定额需要编码");
      return;
    }
    setBusy(true);
    setError("");
    try {
      const saved = await app.WorkcostQuotaSave({
        ...quota,
        // 另存 / 新建：新编码、新身份；溯源字段不带过去（那是指向源定额之外的东西）。
        ...(creating ? { id: 0, origCode: "", derivedFrom: "", gcode: "" } : {}),
        code: finalCode,
        title: title.trim(), specialty: specialty.trim(), chapter: chapter.trim(),
        region: region.trim(), unit: unit.trim(), note: note.trim(),
        source: creating ? (mode === "duplicate" ? `造价库界面复制自 ${quota.code}` : "造价库界面新建") : quota.source,
        items: creating ? items.map((it) => ({ ...it, id: 0, quotaCode: finalCode })) : items,
      });
      if (creating) {
        onCreated?.(saved);
        onSaved(mode === "duplicate" ? `已另存为定额「${saved.title || title.trim()}」（${saved.code || finalCode}）` : `已新建定额「${saved.title || title.trim()}」`);
      } else {
        onSaved(`已保存定额「${title.trim()}」`);
      }
      onCancel(); // 成功即退出编辑态（整组替换后回只读详情，页面静默重读跟在后面）
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  };

  const addItem = (pick: string) => {
    const resourceCode = pick.trim();
    if (!resourceCode) return;
    if (items.some((it) => it.resourceCode === resourceCode)) {
      setError(`含量行里已有 ${resourceCode}，直接改那行的含量`);
      return;
    }
    const match = resources.find((r) => r.code === resourceCode);
    setItems((list) => [...list, {
      quotaCode: creating ? code.trim() : quota.code, resourceCode,
      kind: (match?.kind ?? "材料") as WorkcostQuotaItem["kind"],
      title: match?.title ?? resourceCode, spec: match?.spec ?? "", unit: match?.unit ?? "",
      quantity: addQty, resourcePrice: 0, lossRate: 0,
    }]);
    setAddCode("");
    setAddQty(0);
    setError("");
  };

  const requestCancel = () => {
    if (guardCancel(dirty)) onCancel();
  };
  useEditLifecycle({ dirty, onDirty, save: () => void save(), cancel: requestCancel });

  const searchResources = useCallback(async (kw: string): Promise<PickerOption[]> =>
    ((await app.WorkcostResourceList("", kw)) ?? []).map((r) => ({ code: r.code, title: r.title, spec: r.spec })), []);

  return (
    <div className="cl-edit" data-testid="cost-quota-edit">
      <div className="cl-edit-grid">
        {creating && (
          <Field label="编码" hint={mode === "duplicate" ? `另存为新编码（复制自 ${quota.code}）；已存在会撞唯一约束，换一个` : "库内唯一"}>
            <input className="cl-edit-input" value={code} data-testid="cost-edit-quota-code" onChange={(e) => setCode(e.target.value)} />
          </Field>
        )}
        <Field label="名称"><input className="cl-edit-input" value={title} data-testid="cost-edit-quota-title" onChange={(e) => setTitle(e.target.value)} /></Field>
        <Field label="计量单位"><input className="cl-edit-input" value={unit} onChange={(e) => setUnit(e.target.value)} /></Field>
        <Field label="专业" hint="字典里的专业（规划了但还没行在用的也在）">
          <DictSelect label="定额专业" testid="cost-edit-quota-specialty" value={specialty} emptyLabel="未标注"
            options={unionOptions(dicts?.specialty, specialtyOptions)}
            add={dicts?.add ? (name) => dicts.add!("specialty", name) : undefined}
            onChange={(next) => {
              setSpecialty(next);
              // 换专业后原分部多半不归属了：不在新专业的分部面里就落回未归。
              if (!chapterOptionsOf(next).includes(chapter)) setChapter("");
            }} />
        </Field>
        <Field label="分部" hint="随专业走；没有合适的就保持未归">
          <select className="cl-select" aria-label="定额分部" value={chapter} onChange={(e) => setChapter(e.target.value)}>
            <option value="">未归分部</option>
            {chapterOptionsOf(specialty).map((c) => <option key={c} value={c}>{c}</option>)}
            {chapter && !chapterOptionsOf(specialty).includes(chapter) && <option value={chapter}>{chapter}</option>}
          </select>
        </Field>
        <Field label="地区" hint="省级行政区（只到省）；随上游清单项目导入带上，没有合适的就留空">
          <DictSelect label="定额地区" value={region} emptyLabel="未标地区"
            options={unionOptions(dicts?.region, regionOptions)}
            add={dicts?.add ? (name) => dicts.add!("region", name) : undefined}
            onChange={setRegion} />
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
          <SearchPicker label="添加含量行的工料机编码" placeholder="搜工料机名称 / 规格，或直接填编码" text={addCode} onText={setAddCode}
            search={searchResources} onPick={setAddCode} />
          <NumInput value={addQty} onChange={setAddQty} />
          <button type="button" className="cl-btn" disabled={!addCode.trim()} data-testid="cost-edit-item-add" onClick={() => addItem(addCode)}>
            <Plus size={12} aria-hidden />加行
          </button>
        </div>
        <p className="cl-edit-hint">整组替换落库：没列在这儿的旧行会删掉；新行缺的行类 / 单位 / 单价按库里的工料机补齐。</p>
      </section>
      <EditActions busy={busy} error={error} saveLabel={creating ? (mode === "duplicate" ? "另存为新定额" : "新建定额") : "保存定额"} onSave={() => void save()} onCancel={requestCancel} />
    </div>
  );
}

// ── 工料机编辑（主数据 + 按期调价）──────────────────────────────────

export function ResourceEdit({ resource, regionOptions, periodOptions, categoryOptions, onSaved, onCancel, dicts, onDirty, mode = "edit", onCreated }: {
  resource: WorkcostResource;
  /** 库里已有的地区 / 期数取值（字典接入后地区只是兜底）。 */
  regionOptions: string[];
  periodOptions: string[];
  /** 该类别下已有的分类路径。 */
  categoryOptions: string[];
  onSaved: (msg: string) => void;
  onCancel: () => void;
  dicts?: EditDicts;
  onDirty?: (dirty: boolean) => void;
  mode?: EditMode;
  onCreated?: (saved: WorkcostResource) => void;
}) {
  const creating = mode !== "edit";
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

  const snapshot = JSON.stringify([kind, title, spec, unit, categoryPath, region, priceDate, lossRate, note, basePrice, currentPrice]);
  const [initial] = useState(snapshot);
  const dirty = creating || snapshot !== initial;

  const save = async () => {
    if (!title.trim()) {
      setError("工料机需要名称");
      return;
    }
    setBusy(true);
    setError("");
    try {
      const saved = await app.WorkcostResourceSave({
        ...resource,
        // 新建 / 复制：新身份；编码留空由后端分配，自有码不带过去。
        ...(creating ? { id: 0, code: "", gcode: "" } : {}),
        kind: kind as WorkcostResource["kind"], title: title.trim(), spec: spec.trim(),
        unit: unit.trim(), categoryPath: categoryPath.trim(), region: region.trim(), priceDate: priceDate.trim(),
        lossRate: lossRate / 100, note: note.trim(), basePrice, currentPrice,
        source: creating ? (mode === "duplicate" ? `造价库界面复制自 ${resource.code}` : "造价库界面新建") : resource.source,
      });
      // 改价留痕：主数据保存里直接改了现行价，也登记一条调价历史（登记失败不推翻已保存的事实，
      // 但要说出来）。
      let historyNote = "";
      if (!creating && currentPrice > 0 && currentPrice !== resource.currentPrice) {
        try {
          await app.WorkcostResourceSetPrice(saved.id, currentPrice, saved.priceDate || priceDate.trim(), saved.region || region.trim(),
            resource.priceType || "界面改价", "造价库界面", `编辑面改现行价：${resource.currentPrice} → ${currentPrice}`);
        } catch (e) {
          historyNote = `（调价历史登记失败：${message(e)}）`;
        }
      }
      if (creating) {
        onCreated?.(saved);
        onSaved(`已新建工料机「${saved.title}」${saved.code ? `（${saved.code}）` : ""}`);
      } else {
        onSaved(`已保存工料机「${saved.title}」${historyNote}`);
      }
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

  const requestCancel = () => {
    if (guardCancel(dirty)) onCancel();
  };
  useEditLifecycle({ dirty, onDirty, save: () => void save(), cancel: requestCancel });

  const categoryOptionsOfKind = unionOptions(dicts?.categoryOf?.(kind), categoryOptions);
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
        <Field label="分类" hint="字典里该类别下的分类；没有合适的就地新增，未细分就保持空">
          <DictSelect label="工料机分类" value={categoryPath} emptyLabel="未细分"
            options={categoryOptionsOfKind}
            add={dicts?.add ? (name) => dicts.add!("category", name.trim(), name.includes("/") ? name.slice(0, name.lastIndexOf("/")) : "") : undefined}
            onChange={setCategoryPath} />
        </Field>
        <Field label="地区" hint="省级行政区（只到省）；工料机按地区分行，改动会跟着身份走">
          <DictSelect label="工料机地区" value={region} emptyLabel="未标注"
            options={unionOptions(dicts?.region, regionOptions)}
            add={dicts?.add ? (name) => dicts.add!("region", name) : undefined}
            onChange={setRegion} />
        </Field>
        <Field label="价格期数" hint="YYYY-MM；新月份直接填，库里统一归一成年月">
          <PeriodInput label="价格期数" value={priceDate} onChange={setPriceDate} options={periodOptions} testid="cost-edit-res-period" />
        </Field>
        <Field label="损耗率%"><NumInput value={lossRate} onChange={setLossRate} step="0.1" /></Field>
        <Field label="基准价"><NumInput value={basePrice} onChange={setBasePrice} /></Field>
        <Field label="现行价" hint="直接改现行价会同时登记一条调价历史">
          <NumInput value={currentPrice} onChange={setCurrentPrice} testid="cost-edit-res-price" />
        </Field>
      </div>
      <Field label="备注">
        <textarea className="cl-edit-input cl-edit-area" rows={2} value={note} onChange={(e) => setNote(e.target.value)} />
      </Field>
      <EditActions busy={busy} error={error} saveLabel={creating ? (mode === "duplicate" ? "另存为新工料机" : "新建工料机") : "保存工料机"} onSave={() => void save()} onCancel={requestCancel} />

      {!creating && (
        <section className="cl-block">
          <div className="cl-block-head"><span className="cl-block-label">按期调价</span><span className="cl-dim">写调价历史并推进现行价</span></div>
          <div className="cl-edit-row">
            <NumInput value={pPrice} onChange={setPPrice} testid="cost-edit-res-setprice-price" />
            <PeriodInput label="调价期数" value={pPeriod} onChange={setPPeriod} options={periodOptions} testid="cost-edit-res-setprice-period" />
            <select className="cl-select" value={pRegion} aria-label="调价地区" onChange={(e) => setPRegion(e.target.value)}>
              <option value="">地区</option>
              {unionOptions(dicts?.region, regionOptions).map((r) => <option key={r} value={r}>{r}</option>)}
            </select>
            <input className="cl-edit-input" value={pType} aria-label="价格口径" onChange={(e) => setPType(e.target.value)} />
            <button type="button" className="cl-btn" disabled={priceBusy || pPrice <= 0} data-testid="cost-edit-res-setprice" onClick={() => void setPrice()}>登记调价</button>
          </div>
        </section>
      )}
    </div>
  );
}

// ── 项目封面编辑（口径 + 费率）──────────────────────────────────────

export function ProjectEdit({ project, specialtyOptions, regionOptions, periodOptions, onSaved, onCancel, dicts, onDirty }: {
  project: WorkcostBillProject;
  specialtyOptions: string[];
  regionOptions: string[];
  periodOptions: string[];
  onSaved: (msg: string) => void;
  onCancel: () => void;
  dicts?: EditDicts;
  onDirty?: (dirty: boolean) => void;
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

  const snapshot = JSON.stringify([specialty, region, pricePeriod, managementRate, regulatoryRate, profitRate, taxRate, profitIncludesRegulatory, controlPrice]);
  const [initial] = useState(snapshot);
  const dirty = snapshot !== initial;

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

  const requestCancel = () => {
    if (guardCancel(dirty)) onCancel();
  };
  // Ctrl+S：口径与费率各存各的，两档一起存。
  useEditLifecycle({ dirty, onDirty, save: () => { void saveProfile().then(saveRates); }, cancel: requestCancel });

  const specialtyOptionsAll = unionOptions(dicts?.specialty, specialtyOptions);
  const regionOptionsAll = unionOptions(dicts?.region, regionOptions);
  return (
    <div className="cl-edit" data-testid="cost-project-edit">
      <section className="cl-block">
        <div className="cl-block-head"><span className="cl-block-label">口径（规范值，空=保持原值）</span></div>
        <div className="cl-edit-grid">
          <Field label="专业">
            <DictSelect label="项目专业" testid="cost-edit-proj-specialty" value={specialty} emptyLabel={specialty ? "（保持）" : "未标注"}
              options={specialtyOptionsAll}
              add={dicts?.add ? (name) => dicts.add!("specialty", name) : undefined}
              onChange={setSpecialty} />
          </Field>
          <Field label="地区" hint="省级行政区（只到省）">
            <DictSelect label="项目地区" value={region} emptyLabel={region ? "（保持）" : "未标注"}
              options={regionOptionsAll}
              add={dicts?.add ? (name) => dicts.add!("region", name) : undefined}
              onChange={setRegion} />
          </Field>
          <Field label="计价期" hint="YYYY-MM；新月份直接填">
            <PeriodInput label="计价期" value={pricePeriod} onChange={setPricePeriod} options={periodOptions} testid="cost-edit-proj-period" />
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
          <button type="button" className="cl-btn" disabled={busy} onClick={requestCancel}><X size={12} aria-hidden />取消</button>
          {error && <span className="cl-warn-text" role="alert" data-testid="cost-edit-error">{error}</span>}
        </div>
      </section>
    </div>
  );
}
