import { Button, Card, message, Select, Space, Switch, Tag, Typography } from 'antd'
import { LinkOutlined } from '@ant-design/icons'
import * as App from '../../../wailsjs/go/app/App'
import { C } from '../../utils/theme'
import { engineLabel, FEATURES } from './utils'
import { useModelCenter } from './context'

export function BindSection() {
  const { engines, featureCfg, featureDraft, featureEnabled, modelRoutes, chatVoiceCfg, chatVoiceDraft, chatVoiceSaving, chatVoiceOptions, chatVoiceValue, voiceCfg, setVoiceCfg, portraitCfg, portraitDraft, portraitModelOptions, portraitSaving, llmModels, ttsModels, setFeatureDraft, setChatVoiceDraft, setPortraitDraft, handleSaveFeature, handleToggleFeatureEnabled, handleSaveChatVoice, handleClearChatVoice, handleSavePortrait } = useModelCenter()
  return (
            <div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 14, flexWrap: 'wrap' }}>
                <LinkOutlined style={{ color: C('color-text-secondary') }} />
                <Typography.Text strong style={{ color: C('color-text'), fontSize: 15 }}>功能模型绑定</Typography.Text>
                <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11 }}>
                  各功能板块独立模型，设置后持久化（重启不丢）
                </Typography.Text>
              </div>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', gap: 14 }}>
                {FEATURES.map(f => {
                  const cur = featureCfg[f.key]
                  const draft = featureDraft[f.key] || { engine: '', model: '' }
                  const engineModels = draft.engine ? llmModels.filter(m => m.engineId === draft.engine) : []
                  const bound = !!cur?.engine && !!cur?.model
                  return (
                    <Card key={f.key} size="small" style={{ background: 'var(--bg-glass)', border: bound ? '1px solid rgba(34,197,94,0.35)' : '1px solid var(--border-subtle)', borderRadius: 12 }}>
                      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 10, gap: 8, flexWrap: 'wrap' }}>
                        <Space size={6}>
                          <span style={{ fontSize: 16 }}>{f.icon}</span>
                          <Typography.Text strong style={{ color: C('color-text'), fontSize: 13 }}>{f.label}</Typography.Text>
                          {f.key === 'chat' && (
                            <>
                              <Tag color="purple" style={{ fontSize: 9, margin: 0 }}>TTS {chatVoiceCfg.model || voiceCfg.tts.model || '自动'}</Tag>
                              <Tag color="blue" style={{ fontSize: 9, margin: 0 }}>STT {voiceCfg.stt.model || '自动'}</Tag>
                            </>
                          )}
                          {f.key === 'office' && <Tag color="cyan" style={{ fontSize: 9, margin: 0 }}>通用 + 方案 + 知识库</Tag>}
                          {f.key === 'characterlib' && <Tag color="geekblue" style={{ fontSize: 9, margin: 0 }}>生成 / 补全</Tag>}
                          {f.key === 'routine' && <Tag color="green" style={{ fontSize: 9, margin: 0 }}>通用文本兜底 · 本地/免费优先</Tag>}
                        </Space>
                        <Tag color={bound ? 'green' : 'default'} style={{ fontSize: 10, margin: 0 }}>{bound ? '已绑定' : '未绑定'}</Tag>
                      </div>
                      <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginBottom: 10, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {bound ? `当前：${cur!.engine} / ${cur!.model}` : '尚未绑定，选择引擎和模型后点绑定'}
                      </Typography.Text>
                      {f.key === 'routine' && (
                        <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginBottom: 6, lineHeight: 1.6 }}>
                          纯文本摘要/归一化/抽取/改写等无专业工具覆盖的活，由云端 agent 通过 routine_llm 兜底调用（识图/OCR/检索/转换走各自专业工具）；未绑定默认本地 herdsman
                        </Typography.Text>
                      )}
                      {modelRoutes[f.key] && (
                        <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block' }}>
                          当前生效：{modelRoutes[f.key].engine || '-'} / {modelRoutes[f.key].model || '-'}（{modelRoutes[f.key].source || '-'}）
                        </Typography.Text>
                      )}
                      {f.key === 'office' && modelRoutes['gaea'] && (
                        <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginTop: 2 }}>
                          通用办公 / 知识库路由：{modelRoutes['gaea'].engine || '-'} / {modelRoutes['gaea'].model || '-'}（{modelRoutes['gaea'].source || '-'}）
                        </Typography.Text>
                      )}
                      <div style={{ display: 'flex', gap: 8 }}>
                        <Select size="small" placeholder="引擎" value={draft.engine || undefined}
                          onChange={(v: string) => setFeatureDraft(p => ({ ...p, [f.key]: { engine: v, model: '' } }))}
                          style={{ flex: 1, minWidth: 0 }}
                          options={engines.filter(e => e.enabled).map(e => ({ value: e.id, label: engineLabel(e) }))} />
                        <Select size="small" placeholder="模型" value={draft.model || undefined}
                          onChange={(v: string) => setFeatureDraft(p => ({ ...p, [f.key]: { engine: p[f.key]?.engine || '', model: v } }))}
                          style={{ flex: 1, minWidth: 0 }}
                          options={engineModels.map(m => ({ value: m.modelId, label: m.modelName }))} />
                      </div>
                      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginTop: 10, gap: 8 }}>
                        <span style={{ fontSize: 11, color: C('color-text-secondary') }}>功能启用（停用后回退全局模型）</span>
                        <Switch size="small" checked={featureEnabled[f.key] !== false} onChange={(v: boolean) => handleToggleFeatureEnabled(f.key, v)} />
                      </div>
                      <Button size="small" type={bound ? 'primary' : 'default'} block onClick={() => handleSaveFeature(f.key)} style={{ marginTop: 8, fontSize: 11 }}>
                        {bound ? '更新绑定' : '绑定'}
                      </Button>
                    </Card>
                  )
                })}
                {/* 功能绑定：聊天语音（优先于全局 TTS，模型列表随引擎自动刷新） */}
                <Card size="small" style={{ background: 'var(--bg-glass)', border: chatVoiceCfg.model ? '1px solid rgba(168,85,247,0.35)' : '1px dashed var(--border-subtle)', borderRadius: 12 }}>
                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 10, gap: 8, flexWrap: 'wrap' }}>
                    <Space size={6}>
                      <span style={{ fontSize: 16 }}>🗣️</span>
                      <Typography.Text strong style={{ color: C('color-text'), fontSize: 13 }}>聊天语音</Typography.Text>
                      <Tag color="purple" style={{ fontSize: 9, margin: 0 }}>TTS</Tag>
                      {chatVoiceCfg.model && <Tag color="geekblue" style={{ fontSize: 9, margin: 0 }}>功能绑定</Tag>}
                    </Space>
                    <Tag color={chatVoiceCfg.model ? 'green' : 'default'} style={{ fontSize: 10, margin: 0 }}>
                      {chatVoiceCfg.model ? '已绑定' : '未绑定'}
                    </Tag>
                  </div>
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginBottom: 8, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {chatVoiceCfg.model ? `当前：${chatVoiceCfg.engine} / ${chatVoiceCfg.model}` : '未绑定：语音对话使用全局 TTS（语音模型页）'}
                  </Typography.Text>
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 10, display: 'block', marginBottom: 8 }}>
                    优先于全局 TTS；列表随引擎模型自动刷新，后续新增语音模型无需改代码
                  </Typography.Text>
                  <div style={{ display: 'flex', gap: 8 }}>
                    <Select size="small" placeholder="引擎" value={chatVoiceDraft.engine || undefined}
                      onChange={(v: string) => setChatVoiceDraft({ engine: v, model: '' })}
                      style={{ flex: 1, minWidth: 0 }}
                      options={engines.filter(e => e.enabled && ttsModels.some(m => m.engineId === e.id)).map(e => ({ value: e.id, label: engineLabel(e) }))} />
                    <Select size="small" placeholder="语音模型" value={chatVoiceDraft.model || undefined}
                      onChange={(v: string) => setChatVoiceDraft(p => ({ ...p, model: v }))}
                      style={{ flex: 1, minWidth: 0 }}
                      options={ttsModels.filter(m => m.engineId === chatVoiceDraft.engine).map(m => ({ value: m.modelId, label: m.modelName }))} />
                  </div>
                  {chatVoiceDraft.engine && chatVoiceDraft.model && chatVoiceOptions.length > 0 && (
                    <div style={{ display: 'flex', gap: 8, marginTop: 8, alignItems: 'center' }}>
                      <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, whiteSpace: 'nowrap' }}>音色</Typography.Text>
                      <Select size="small" value={chatVoiceValue} placeholder="选择音色"
                        onChange={async (v: string) => {
                          try {
                            await (App as any).VoiceApplySettings?.({ ttsVoice: v })
                            message.success('音色已更新：' + v)
                          } catch (err: any) {
                            message.error(err?.message || '音色更新失败')
                          }
                          setVoiceCfg(p => ({ ...p, tts: { ...p.tts, voice: v } }))
                        }}
                        style={{ flex: 1, minWidth: 0 }}
                        options={chatVoiceOptions} />
                    </div>
                  )}
                  <div style={{ display: 'flex', gap: 8, marginTop: 8 }}>
                    <Button size="small" type={chatVoiceCfg.model ? 'primary' : 'default'} block loading={chatVoiceSaving} onClick={handleSaveChatVoice} style={{ fontSize: 11 }}>
                      {chatVoiceCfg.model ? '更新绑定' : '绑定聊天语音'}
                    </Button>
                    {chatVoiceCfg.model && (
                      <Button size="small" danger onClick={handleClearChatVoice} loading={chatVoiceSaving} style={{ fontSize: 11 }}>清除</Button>
                    )}
                  </div>
                </Card>
                {/* 绘梦：自身界面选择 */}
                <Card size="small" style={{ background: 'rgba(255,255,255,0.02)', border: '1px dashed var(--border-subtle)', borderRadius: 12 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <span style={{ fontSize: 16 }}>🎨</span>
                    <Typography.Text strong style={{ color: C('color-text'), fontSize: 13 }}>绘梦</Typography.Text>
                  </div>
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginTop: 8, lineHeight: 1.6 }}>
                    图片模型在绘梦界面内选择（后端 / 模型 / ComfyUI 启停），无需在此重复设置
                  </Typography.Text>
                </Card>

                {/* 角色库剧照：独立图片后端/模型（空 = 跟随绘梦） */}
                <Card size="small" style={{ background: portraitCfg.backend ? 'var(--bg-glass)' : 'rgba(255,255,255,0.02)', border: portraitCfg.backend ? '1px solid rgba(96,165,250,0.35)' : '1px dashed var(--border-subtle)', borderRadius: 12 }}>
                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 10, gap: 8, flexWrap: 'wrap' }}>
                    <Space size={6}>
                      <span style={{ fontSize: 16 }}>🖼️</span>
                      <Typography.Text strong style={{ color: C('color-text'), fontSize: 13 }}>角色库剧照</Typography.Text>
                      <Tag color="blue" style={{ fontSize: 9, margin: 0 }}>图片</Tag>
                    </Space>
                    <Tag color={portraitCfg.backend ? 'blue' : 'default'} style={{ fontSize: 10, margin: 0 }}>
                      {portraitCfg.backend ? `${portraitCfg.backend} / ${portraitCfg.model || '跟随绘梦'}` : '跟随绘梦'}
                    </Tag>
                  </div>
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginBottom: 10, lineHeight: 1.6 }}>
                    角色卡「生成剧照」使用独立后端；留空则跟随绘梦页当前选择
                  </Typography.Text>
                  <div style={{ display: 'flex', gap: 8 }}>
                    <Select
                      size="small"
                      placeholder="后端"
                      value={portraitDraft.backend || undefined}
                      onChange={(v: string) => setPortraitDraft({ backend: v, model: '' })}
                      style={{ flex: 1, minWidth: 0 }}
                      options={[
                        { value: '', label: '跟随绘梦' },
                        { value: 'xai', label: 'xAI 云端' },
                        { value: 'comfyui', label: 'ComfyUI 本地' },
                        { value: 'herdsman', label: 'Herdsman 本地' },
                        { value: 'ollama', label: 'Ollama 本地' },
                      ]}
                    />
                    <Select
                      size="small"
                      placeholder="模型"
                      value={portraitDraft.model || undefined}
                      onChange={(v: string) => setPortraitDraft(p => ({ ...p, model: v }))}
                      style={{ flex: 1, minWidth: 0 }}
                      options={portraitModelOptions}
                    />
                  </div>
                  <Button size="small" type={portraitCfg.backend ? 'primary' : 'default'} block
                    loading={portraitSaving} onClick={handleSavePortrait}
                    style={{ marginTop: 10, fontSize: 11 }}>
                    {portraitCfg.backend ? '更新剧照绑定' : '绑定剧照后端'}
                  </Button>
                </Card>
              </div>
            </div>
  )
}
