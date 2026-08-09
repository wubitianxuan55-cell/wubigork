import { Button, Card, Collapse, Segmented, Select, Space, Tag, Typography } from 'antd'
import { SettingOutlined } from '@ant-design/icons'
import SettingField from '../../components/SettingField'
import { C } from '../../utils/theme'
import { engineColor, engineLabel } from './utils'
import { useModelCenter } from './context'

const COMFY_IMAGES = [
  { modelId: 'krea2', modelName: 'Krea2 Turbo', engineId: 'comfyui', engineName: 'ComfyUI', status: 'running' },
  { modelId: 'z-image-turbo', modelName: 'Z-Image-Turbo', engineId: 'comfyui', engineName: 'ComfyUI', status: 'running' },
]

export function ImageSection() {
  const { imageBackend, setImageBackend, comfyStatus, comfyBusy, handleToggleComfy, comfyUIURL, comfyUIPath, comfyUIPythonPath, imageModel, setImageModel, imageModels, imageBackendSaving, handleSaveImageBackend, imageSaveDir, setImageSaveDir } = useModelCenter()
  return (
            <>
              {/* 图片后端 + ComfyUI + 模型 */}
              <Card style={{ marginBottom: 20, background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12 }}>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
                  {/* 后端选择 */}
                  <div>
                    <Typography.Text strong style={{ color: C('color-text'), fontSize: 14, display: 'block', marginBottom: 8 }}>图片生成后端</Typography.Text>
                    <Segmented
                      value={imageBackend}
                      onChange={(v) => setImageBackend(v as string)}
                      options={[
                        { value: 'xai', label: '☁️ xAI 云端' },
                        { value: 'comfyui', label: '🏠 ComfyUI 本地' },
                        { value: 'herdsman', label: '🚀 Herdsman' },
                        { value: 'ollama', label: '🖥 Ollama' },
                      ]}
                    />
                  </div>

                  {/* ComfyUI 配置（本机默认已写死，仅展示 + 启停） */}
                  {imageBackend === 'comfyui' && (
                    <div style={{ padding: '12px 14px', borderRadius: 10, background: 'rgba(255,255,255,0.03)', border: '1px solid var(--border-subtle)' }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
                        <span style={{
                          width: 8, height: 8, borderRadius: '50%',
                          background: comfyStatus.running ? '#22c55e' : '#64748b',
                          boxShadow: comfyStatus.running ? '0 0 8px #22c55e' : 'none',
                        }} />
                        <Typography.Text style={{ color: C('color-text'), fontSize: 13, fontWeight: 600 }}>
                          ComfyUI {comfyStatus.running ? `运行中 (端口 ${comfyStatus.port || 8188})` : '未启动'}
                        </Typography.Text>
                        <Button size="small" type={comfyStatus.running ? 'default' : 'primary'} loading={comfyBusy}
                          onClick={handleToggleComfy} style={{ fontSize: 11 }}>
                          {comfyStatus.running ? '⏹ 停止' : '▶ 启动'}
                        </Button>
                      </div>
                      <div style={{ fontSize: 11, color: C('color-text-secondary'), marginTop: 8, lineHeight: 1.8 }}>
                        <div>URL：<span style={{ color: C('color-text') }}>{comfyUIURL}</span></div>
                        <div>启动位置：<span style={{ color: C('color-text') }}>{comfyUIPath || 'C:\\AI\\ComfyUI\\ComfyUI（默认）'}</span></div>
                        <div>Python：<span style={{ color: C('color-text') }}>{comfyUIPythonPath || 'C:\\AI\\ComfyUI\\standalone-env\\python.exe（默认）'}</span></div>
                      </div>
                    </div>
                  )}

                  {/* 图片模型选择 */}
                  <div>
                    <Typography.Text strong style={{ color: C('color-text'), fontSize: 14, display: 'block', marginBottom: 8 }}>图片模型</Typography.Text>
                    <Select
                      size="middle" style={{ width: 320 }} value={imageModel}
                      onChange={setImageModel}
                      options={[
                        ...(imageBackend === 'comfyui' ? COMFY_IMAGES : []).map(m => ({ value: m.modelId, label: `${m.modelName}（ComfyUI 本地）` })),
                        ...imageModels.map(m => ({ value: m.modelId, label: m.modelName })),
                        ...(imageBackend !== 'comfyui' ? COMFY_IMAGES.map(m => ({ value: m.modelId, label: `${m.modelName}（ComfyUI 本地）` })) : []),
                        { value: 'grok-imagine-image-quality', label: 'Grok Imagine（xAI）' },
                      ]}
                    />
                  </div>

                  {/* 保存 */}
                  <div>
                    <Button type="primary" onClick={handleSaveImageBackend} loading={imageBackendSaving} style={{ borderRadius: 8 }}>
                      💾 保存图片后端设置
                    </Button>
                  </div>
                </div>
              </Card>

              {/* 发现的图片模型（含 ComfyUI 本地模型 krea2 / z-image-turbo） */}
              {(imageModels.length > 0 || imageBackend === 'comfyui') && (
                <div style={{ marginBottom: 24 }}>
                  <Typography.Text strong style={{ color: C('color-text'), fontSize: 15, display: 'block', marginBottom: 10 }}>发现的图片模型</Typography.Text>
                  <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(220px, 1fr))', gap: 10 }}>
                    {[...COMFY_IMAGES, ...imageModels.filter(m => m.engineId !== 'comfyui')].map(m => (
                      <Card key={`${m.engineId}:${m.modelId}`} size="small" style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 10 }}>
                        <Typography.Text strong style={{ color: C('color-text'), fontSize: 13, display: 'block', marginBottom: 6, wordBreak: 'break-all' }}>{m.modelName}</Typography.Text>
                        <Space>
                          <Tag color={engineColor(m)} style={{ fontSize: 10, margin: 0 }}>{engineLabel(m)}</Tag>
                          <Tag color="orange" style={{ fontSize: 10, margin: 0 }}>图片</Tag>
                          <Tag color={m.status === 'running' ? 'green' : 'default'} style={{ fontSize: 10, margin: 0 }}>{m.status === 'running' ? '● 运行中' : '○ 已停止'}</Tag>
                        </Space>
                      </Card>
                    ))}
                  </div>
                </div>
              )}
              <Collapse ghost size="small" defaultActiveKey={['img-cfg']} items={[{
                key: 'img-cfg', label: <span style={{ color: C('color-text-secondary'), fontSize: 13 }}><SettingOutlined style={{ marginRight: 6 }} />图片存储</span>,
                children: (
                  <Card style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12 }}>
                    <Space direction="vertical" size={12} style={{ width: '100%' }}>
                      <SettingField label="图片保存目录" value={imageSaveDir} onChange={v => setImageSaveDir(v)} placeholder="默认: Pictures/gaea" />
                      <Button type="primary" onClick={handleSaveImageBackend} loading={imageBackendSaving} style={{ borderRadius: 8 }}>💾 保存</Button>
                    </Space>
                  </Card>
                ),
              }]} />
            </>
  )
}