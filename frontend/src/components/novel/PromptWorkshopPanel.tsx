// PromptWorkshopPanel.tsx — 提示词工坊面板（t6 首刀：模板可编辑覆盖层）。
// 20 个 prompts/*.json 此前对用户只读——本面板消费 PromptTemplate* 五绑定，
// 呈现「列表分组 → 详情编辑（内置基线对照）→ 变量渲染预览」三区；
// 打开才拉一次（零轮询，RewriteHistoryPanel 同款），保存/恢复后定向刷新。
// 来源语义：override=自定义（orange）/ builtin=内置（default）；覆盖行存在
// 但未启用加灰 Tag「已停用」（此时生效的仍是内置）。
// t6-C2：顶栏「导出/导入模板包」——导出走 saveExportBlob 双门落盘；导入
// 双门选文件后三态决策在 Go 侧，结果弹窗逐行展示（规格
// 进度计划/gaea-prompt-bundle-t6c2-20260916.md §5）。
import React, { useCallback, useEffect, useMemo, useState } from 'react'
import { Alert, Button, Collapse, Empty, Input, Modal, Popconfirm, Spin, Switch, Tag, Typography, message } from 'antd'
import {
  listTemplates, getTemplate, saveTemplate, resetTemplate, previewTemplate, exportBundle, importBundle,
  type BundleImportResult, type PromptTemplateDetail, type PromptTemplateIssue, type PromptTemplateMeta, type PromptTemplateView,
} from './api/prompt'
import { saveExportBlob } from '../../gaea/lib/saveFile'
import { inShellEnv, pickFileAsFile } from '../../gaea/lib/pickFile'

/** category → 分组标题（小说域口径；未知分类回落「其它」）。 */
const CATEGORY_LABELS: Record<string, string> = {
  worldview: '世界观',
  character: '角色',
  outline: '大纲',
  chapter: '章节创作',
  rewrite: '重写',
  analysis: '情节分析',
  summary: '摘要',
  'book-import': '拆书导入',
  skill: '技能结晶',
}

/** 编辑表单草稿（预览未保存内容时也用这份草稿）。 */
interface DraftForm {
  system: string
  task: string
  outputDesc: string
  category: string
  description: string
  isActive: boolean
}

const EMPTY_FORM: DraftForm = { system: '', task: '', outputDesc: '', category: '', description: '', isActive: true }

const softTextStyle: React.CSSProperties = { fontSize: 12, color: 'var(--v3-fg-soft, #6b7280)' }
const labelTextStyle: React.CSSProperties = { fontSize: 12, color: 'var(--v3-fg-soft, #6b7280)', marginBottom: 2 }

/** 导入 action → 结果弹窗标签（Go promptstore.Action*；skipped_* 用警示色）。 */
const ACTION_VIEWS: Record<string, { label: string; color?: string }> = {
  kept_system_default: { label: '维持内置' },
  converted_to_custom: { label: '转为自定义', color: 'orange' },
  created_or_updated: { label: '写入自定义', color: 'green' },
  skipped_invalid: { label: '跳过·校验未过', color: 'red' },
  skipped_unknown: { label: '跳过·未知键', color: 'red' },
  skipped_duplicate: { label: '跳过·重复键', color: 'orange' },
}

export default function PromptWorkshopPanel({ open, onClose }: {
  open: boolean
  onClose: () => void
}) {
  const [metas, setMetas] = useState<PromptTemplateMeta[]>([])
  const [loading, setLoading] = useState(false)
  const [selectedKey, setSelectedKey] = useState<string | null>(null)
  const [detail, setDetail] = useState<PromptTemplateDetail | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const [form, setForm] = useState<DraftForm>(EMPTY_FORM)
  const [issues, setIssues] = useState<PromptTemplateIssue[]>([])
  const [vars, setVars] = useState<Record<string, string>>({})
  const [preview, setPreview] = useState<{ systemPrompt: string; warnings: string[] } | null>(null)
  const [saving, setSaving] = useState(false)
  const [resetting, setResetting] = useState(false)
  const [previewing, setPreviewing] = useState(false)
  const [exporting, setExporting] = useState(false)
  const [importing, setImporting] = useState(false)
  const [importResult, setImportResult] = useState<BundleImportResult | null>(null)

  const refreshList = useCallback(async () => {
    setLoading(true)
    try {
      setMetas(await listTemplates())
    } catch (e) {
      message.error(`模板清单加载失败：${e instanceof Error ? e.message : String(e)}`)
      setMetas([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (open) {
      setSelectedKey(null)
      setDetail(null)
      setIssues([])
      setPreview(null)
      void refreshList()
    }
  }, [open, refreshList])

  /** 选中模板并拉详情填充表单（keepIssues：保存后刷新详情时保留校验结果展示）。 */
  const select = async (key: string, keepIssues = false) => {
    setSelectedKey(key)
    setDetail(null)
    setPreview(null)
    setVars({})
    if (!keepIssues) setIssues([])
    setDetailLoading(true)
    try {
      const d = await getTemplate(key)
      setDetail(d)
      setForm({
        system: d.template?.system ?? '',
        task: d.template?.task ?? '',
        outputDesc: d.template?.output?.description ?? '',
        category: d.meta?.category || d.template?.category || '',
        description: d.meta?.description || d.template?.description || '',
        isActive: d.meta?.source === 'override' ? d.meta.overrideActive : true,
      })
    } catch (e) {
      message.error(`模板详情加载失败：${e instanceof Error ? e.message : String(e)}`)
    } finally {
      setDetailLoading(false)
    }
  }

  /** 当前草稿组装为完整模板正文（保留 parameters/constraints 等未编辑字段）。 */
  const draftContent = (): PromptTemplateView | null => {
    if (!detail) return null
    return {
      ...detail.template,
      system: form.system,
      task: form.task,
      output: { ...(detail.template?.output ?? {}), description: form.outputDesc },
      category: form.category,
      description: form.description,
    }
  }

  const save = async () => {
    const content = draftContent()
    if (!selectedKey || !content) return
    setSaving(true)
    setIssues([])
    setPreview(null)
    try {
      const res = await saveTemplate(selectedKey, { isActive: form.isActive, category: form.category, description: form.description, content })
      setIssues(res.issues ?? [])
      if (res.saved) {
        message.success('已保存覆盖（生成链路即时生效）')
        await refreshList()
        await select(selectedKey, true)
      } else {
        message.error('保存被校验拦截，请按下方问题修改后重试')
      }
    } catch (e) {
      message.error(e instanceof Error ? e.message : String(e))
    } finally {
      setSaving(false)
    }
  }

  const reset = async () => {
    if (!selectedKey) return
    setResetting(true)
    try {
      await resetTemplate(selectedKey)
      message.success('已恢复内置模板')
      await refreshList()
      await select(selectedKey)
    } catch (e) {
      message.error(e instanceof Error ? e.message : String(e))
    } finally {
      setResetting(false)
    }
  }

  const runPreview = async () => {
    const content = draftContent()
    if (!content) return
    setPreviewing(true)
    try {
      setPreview(await previewTemplate({ content }, vars))
    } catch (e) {
      message.error(`渲染预览失败：${e instanceof Error ? e.message : String(e)}`)
    } finally {
      setPreviewing(false)
    }
  }

  // ── t6-C2 模板包：导出=绑定回 JSON 字符串 → Blob → saveExportBlob 双门
  //（壳内 GaeaSaveFileAs 系统对话框 / 浏览器 a[download]；用户取消静默）。
  const doExport = async () => {
    setExporting(true)
    try {
      const json = await exportBundle()
      const saved = await saveExportBlob(new Blob([json], { type: 'application/json' }), 'gaea-prompt-bundle.json')
      if (saved) message.success('模板包已导出')
    } catch (e) {
      message.error(`模板包导出失败：${e instanceof Error ? e.message : String(e)}`)
    } finally {
      setExporting(false)
    }
  }

  // 导入选文件双门（SkillModal 刀C-3 同款）：壳内 pickFileAsFile（GaeaPickFiles
  // 系统对话框 + .json 后置校验），浏览器动态 input 回退；取消=null 静默。
  const pickBundleFile = async (): Promise<File | null> => {
    if (inShellEnv()) return pickFileAsFile(['json'])
    return new Promise(resolve => {
      const input = document.createElement('input')
      input.type = 'file'
      input.accept = '.json'
      input.onchange = (e: Event) => resolve((e.target as HTMLInputElement).files?.[0] ?? null)
      input.click()
    })
  }

  const doImport = async () => {
    setImporting(true)
    try {
      const file = await pickBundleFile()
      if (!file) return
      const res = await importBundle(await file.text())
      setImportResult(res)
      if (res.applied) {
        message.success('模板包已导入（生成链路即时生效）')
        await refreshList()
        if (selectedKey) await select(selectedKey)
      }
    } catch (e) {
      message.error(`模板包导入失败：${e instanceof Error ? e.message : String(e)}`)
    } finally {
      setImporting(false)
    }
  }

  /** 按 category 分组（保持后端字典序的组内顺序；未知分类归「其它」）。 */
  const groups = useMemo(() => {
    const map = new Map<string, PromptTemplateMeta[]>()
    for (const m of metas) {
      const cat = m.category || ''
      if (!map.has(cat)) map.set(cat, [])
      map.get(cat)!.push(m)
    }
    return Array.from(map.entries()).map(([category, items]) => ({
      category, label: CATEGORY_LABELS[category] ?? '其它', items,
    }))
  }, [metas])

  const fmtTime = (ms: number) => {
    try { return ms > 0 ? new Date(ms).toLocaleString() : '' } catch { return '' }
  }

  const renderRow = (m: PromptTemplateMeta) => {
    const selected = m.key === selectedKey
    return (
      <div key={m.key} data-testid="prompt-workshop-row"
        onClick={() => void select(m.key)}
        style={{ display: 'flex', alignItems: 'center', gap: 6, padding: '5px 8px', borderRadius: 6, cursor: 'pointer', background: selected ? 'rgba(0,0,0,0.05)' : undefined }}>
        <span style={{ fontSize: 13 }}>{m.key}</span>
        {m.source === 'override'
          ? <Tag color="orange" style={{ marginRight: 0 }}>自定义</Tag>
          : <Tag style={{ marginRight: 0 }}>内置</Tag>}
        {m.hasOverride && !m.overrideActive && <Tag style={{ marginRight: 0 }}>已停用</Tag>}
        <span style={{ flex: 1 }} />
        <span style={{ ...softTextStyle, fontVariantNumeric: 'tabular-nums' }}>v{m.version}</span>
      </div>
    )
  }

  return (
    <Modal open={open} title="提示词工坊" onCancel={onClose} footer={null} width={920} destroyOnHidden>
      {/* t6-C2 工具栏：空态也可见（导入正是空环境的恢复路径） */}
      <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8, marginBottom: 10 }}>
        <Button size="small" loading={exporting} data-testid="prompt-workshop-bundle-export" onClick={() => void doExport()}>导出模板包</Button>
        <Button size="small" loading={importing} data-testid="prompt-workshop-bundle-import" onClick={() => void doImport()}>导入模板包</Button>
      </div>
      {loading ? (
        <div style={{ padding: '32px 0', textAlign: 'center' }}><Spin /></div>
      ) : metas.length === 0 ? (
        <Empty description="还没有可编辑的提示词模板（生成链路的提示词模板会在这里列出）" style={{ padding: '24px 0' }} />
      ) : (
        <div style={{ display: 'flex', gap: 16, alignItems: 'stretch' }}>
          {/* 区一：分组列表 */}
          <div data-testid="prompt-workshop-list" style={{ width: 280, flexShrink: 0, maxHeight: '64vh', overflowY: 'auto' }}>
            {groups.map(g => (
              <div key={g.category} style={{ marginBottom: 6 }}>
                <div data-testid="prompt-workshop-group" style={{ ...softTextStyle, fontWeight: 600, padding: '4px 8px' }}>{g.label}</div>
                {g.items.map(renderRow)}
              </div>
            ))}
          </div>
          {/* 区二/三：详情编辑 + 预览 */}
          <div style={{ flex: 1, minWidth: 0, maxHeight: '64vh', overflowY: 'auto' }}>
            {!selectedKey ? (
              <Empty description="从左侧选择一个模板，查看内置基线并编辑覆盖" style={{ padding: '48px 0' }} />
            ) : detailLoading ? (
              <div style={{ padding: '32px 0', textAlign: 'center' }}><Spin /></div>
            ) : detail ? (
              <div data-testid="prompt-workshop-detail">
                <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
                  <strong>{detail.meta.key}</strong>
                  {detail.meta.source === 'override'
                    ? <Tag color="orange" style={{ marginRight: 0 }}>自定义</Tag>
                    : <Tag style={{ marginRight: 0 }}>内置</Tag>}
                  {detail.meta.hasOverride && !detail.meta.overrideActive && <Tag style={{ marginRight: 0 }}>已停用</Tag>}
                  <span style={softTextStyle}>版本 {detail.meta.version}</span>
                  {fmtTime(detail.meta.updatedAt) && <span style={softTextStyle}>· 更新于 {fmtTime(detail.meta.updatedAt)}</span>}
                </div>
                {detail.meta.description && (
                  <Typography.Paragraph type="secondary" style={{ marginBottom: 8, marginTop: 4, fontSize: 13 }}>
                    {detail.meta.description}
                  </Typography.Paragraph>
                )}
                <Collapse size="small" style={{ marginBottom: 12 }} items={[{
                  key: 'base',
                  label: '内置基线（只读对照，恢复内置将回到这里）',
                  children: (
                    <div data-testid="prompt-workshop-base" style={{ whiteSpace: 'pre-wrap', fontSize: 12.5, lineHeight: 1.7, color: 'var(--v3-fg-soft, #6b7280)' }}>
                      {`System：${detail.base?.system ?? '（空）'}\n\nTask：${detail.base?.task ?? '（空）'}`}
                    </div>
                  ),
                }]} />
                <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                  <div>
                    <div style={labelTextStyle}>System（系统提示）</div>
                    <Input.TextArea data-testid="prompt-workshop-system" rows={5} value={form.system}
                      onChange={e => setForm(f => ({ ...f, system: e.target.value }))} />
                  </div>
                  <div>
                    <div style={labelTextStyle}>Task（任务提示）</div>
                    <Input.TextArea data-testid="prompt-workshop-task" rows={5} value={form.task}
                      onChange={e => setForm(f => ({ ...f, task: e.target.value }))} />
                  </div>
                  <div>
                    <div style={labelTextStyle}>输出说明</div>
                    <Input.TextArea data-testid="prompt-workshop-output" rows={2} value={form.outputDesc}
                      onChange={e => setForm(f => ({ ...f, outputDesc: e.target.value }))} />
                  </div>
                  <div style={{ display: 'flex', gap: 8 }}>
                    <Input data-testid="prompt-workshop-category" placeholder="分类（如 chapter）" value={form.category}
                      onChange={e => setForm(f => ({ ...f, category: e.target.value }))} />
                    <Input data-testid="prompt-workshop-description" placeholder="一句话说明" value={form.description}
                      onChange={e => setForm(f => ({ ...f, description: e.target.value }))} />
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <Switch checked={form.isActive} onChange={v => setForm(f => ({ ...f, isActive: v }))} />
                    <span style={softTextStyle}>启用覆盖（关闭后保存但不启用，生成链路继续走内置）</span>
                  </div>
                  {issues.length > 0 && (
                    <div data-testid="prompt-workshop-issues" style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                      {issues.map((iss, i) => (
                        <Alert key={`${iss.code}-${i}`} showIcon
                          type={iss.severity === 'error' ? 'error' : 'warning'}
                          message={`${iss.code}（${iss.severity === 'error' ? '阻断' : '提示'}）`}
                          description={iss.message} />
                      ))}
                    </div>
                  )}
                  <div style={{ display: 'flex', gap: 8, marginTop: 4 }}>
                    <Button size="small" type="primary" loading={saving} data-testid="prompt-workshop-save" onClick={() => void save()}>保存</Button>
                    <Popconfirm title="恢复内置模板" description="将删除该模板的自定义覆盖，保存过的修改会丢失。"
                      okText="确认恢复" cancelText="取消" onConfirm={() => void reset()}>
                      <Button size="small" loading={resetting} data-testid="prompt-workshop-reset">恢复内置</Button>
                    </Popconfirm>
                  </div>
                </div>
                {/* 区三：预览 */}
                <div style={{ marginTop: 16, paddingTop: 12, borderTop: '1px dashed rgba(0,0,0,0.12)' }}>
                  <div style={{ fontSize: 13, fontWeight: 600, marginBottom: 8 }}>变量渲染预览（按模板声明的占位符生成，预览未保存草稿）</div>
                  {(detail.template?.parameters ?? []).length === 0 ? (
                    <Typography.Text type="secondary" style={{ fontSize: 12 }}>该模板没有声明占位符变量</Typography.Text>
                  ) : (
                    <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 8 }}>
                      {(detail.template?.parameters ?? []).map(p => (
                        <Input key={p} size="small" style={{ width: 170 }} placeholder={`变量 ${p}`}
                          data-testid={`prompt-workshop-var-${p}`}
                          value={vars[p] ?? ''}
                          onChange={e => setVars(v => ({ ...v, [p]: e.target.value }))} />
                      ))}
                    </div>
                  )}
                  <Button size="small" loading={previewing} data-testid="prompt-workshop-preview-btn" onClick={() => void runPreview()}>渲染预览</Button>
                  {preview && (
                    <div style={{ marginTop: 8 }}>
                      {(preview.warnings ?? []).length > 0 && (
                        <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', marginBottom: 8, alignItems: 'center' }}>
                          <span style={softTextStyle}>未解析变量：</span>
                          {preview.warnings.map(w => <Tag key={w} color="orange" style={{ marginRight: 0 }}>{w}</Tag>)}
                        </div>
                      )}
                      <pre data-testid="prompt-workshop-preview-out"
                        style={{ margin: 0, background: 'rgba(0,0,0,0.03)', borderRadius: 6, padding: '10px 12px', fontSize: 12.5, lineHeight: 1.7, whiteSpace: 'pre-wrap', maxHeight: 220, overflowY: 'auto' }}>
                        {preview.systemPrompt}
                      </pre>
                    </div>
                  )}
                </div>
              </div>
            ) : null}
          </div>
        </div>
      )}
      {/* t6-C2 导入结果弹窗：统计一行 + 逐行 action 标签（skipped_* 带原因） */}
      <Modal open={importResult !== null} title="模板包导入结果" width={560} destroyOnHidden
        onCancel={() => setImportResult(null)}
        footer={<Button size="small" onClick={() => setImportResult(null)}>知道了</Button>}>
        {importResult && (() => {
          const st = importResult.statistics
          const skipped = st.skippedInvalid + st.skippedUnknown + st.skippedDuplicate
          return (
            <div data-testid="prompt-bundle-result">
              <Typography.Paragraph style={{ fontSize: 13, marginBottom: 10 }}>
                共 {st.total} 行：写入自定义 {st.createdOrUpdate} · 转为自定义 {st.convertedToCustom} · 维持内置 {st.keptSystemDefault}
                {skipped > 0 && <span style={{ color: 'var(--color-error, var(--md-sys-color-error, #cf1322))' }}> · 跳过 {skipped}</span>}
                {!importResult.applied && '（无变更，未落盘）'}
              </Typography.Paragraph>
              <div style={{ maxHeight: 300, overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: 4 }}>
                {importResult.outcomes.map((o, i) => {
                  const view = ACTION_VIEWS[o.action]
                  return (
                    <div key={`${o.key}-${i}`} style={{ display: 'flex', gap: 6, alignItems: 'flex-start' }}>
                      <Tag color={view?.color} style={{ marginRight: 0, flexShrink: 0 }}>{view?.label ?? o.action}</Tag>
                      <span style={{ fontSize: 12.5 }}>{o.key}</span>
                      {o.reason && <span style={softTextStyle}>{o.reason}</span>}
                    </div>
                  )
                })}
              </div>
            </div>
          )
        })()}
      </Modal>
    </Modal>
  )
}
