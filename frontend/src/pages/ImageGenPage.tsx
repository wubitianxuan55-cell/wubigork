// ImageGenPage（T6-10.1 巨型组件拆分后的编排层，行为零变化）
// 职责：状态编排 + 跨 hook 装配；配置/引擎（useImageGenConfig）、生成队列
// （useImageGenQueue）、历史/灯箱（useImageGenHistory）、自定义模板
// （useCustomTemplates）与纯工具（components/imagegen/meta）拆分见各产物文件。
import React, { useState, useCallback } from 'react'
import { Typography, Button, Space, message } from 'antd'
import {
  PictureOutlined, FolderOpenOutlined,
  SwapOutlined, VideoCameraOutlined,
} from '@ant-design/icons'
import { C } from '../utils/theme'
import Lightbox from '../components/Lightbox'
import CustomTemplateModal from '../components/imagegen/CustomTemplateModal'
import TemplatePickerModal from '../components/imagegen/TemplatePickerModal'
import { ControlPanel } from '../components/imagegen/ControlPanel'
import { ResultStage } from '../components/imagegen/ResultStage'
import { GenerationBar } from '../components/imagegen/GenerationBar'
import { TaskCenter } from '../components/imagegen/TaskCenter'
import { StatusDot } from '../components/imagegen/ui'
import { TEMPLATES, type Template } from '../data/imageTemplates'
import { useImageGenConfig } from '../hooks/useImageGenConfig'
import { useImageGenQueue } from '../hooks/useImageGenQueue'
import { useImageGenHistory } from '../hooks/useImageGenHistory'
import { useCustomTemplates } from '../hooks/useCustomTemplates'
import { BACKEND_OPTIONS, resolveResultImage } from '../components/imagegen/meta'
import { downloadFileName } from '../components/imagegen/media'
import '../components/imagegen/imagegen.css'

const ImageGenPage: React.FC = () => {
  const cfg = useImageGenConfig()
  const {
    mode, setMode, prompt, setPrompt, negative, setNegative, size, setSize,
    initImage, setInitImage, denoise, setDenoise, frames, setFrames, fps, setFps,
    model, setModel, seed, setSeed, count, setCount,
    customWidth, setCustomWidth, customHeight, setCustomHeight,
    backend, selectedLoras, setSelectedLoras,
    loraOptions, loraLoading, loraError, refreshComfyLoras,
    backendSwitching, engineRunning, engineStarting, engineModelCount, sysStats,
    modelOptions, characters,
    handleSwitchBackend, handleStartEngine, handleStopEngine,
    handleOpenDir, handleOpenNovelDir,
  } = cfg

  const historyApi = useImageGenHistory({ setPrompt, setNegative, setSeed, setSize })
  const {
    history, setHistory, lightboxIndex, setLightboxIndex,
    handleDownload, handleReuse, handleDelete, handleSetPortrait,
  } = historyApi

  const queueApi = useImageGenQueue({
    setHistory,
    setLightboxIndex,
    config: {
      prompt, mode, initImage, backend, negative, size,
      customWidth, customHeight, model, seed, count, selectedLoras,
      denoise, frames, fps,
    },
  })
  const {
    generating, elapsed, lastTime, genError, comfyProgress,
    results, setResults, pendingCount, queueItems, setQueueItems, canvasRef,
    handleGenerate, handleRegenerateMeta, handleCancel,
  } = queueApi

  const tmpl = useCustomTemplates()
  const {
    customTemplates, customModalOpen, setCustomModalOpen,
    editingCustom, customLabel, setCustomLabel, customDescription, setCustomDescription,
    customSize, setCustomSize, customPrompt, setCustomPrompt,
    customNegative, setCustomNegative,
    openCustomAdd, openCustomEdit, saveCustom, deleteCustom,
  } = tmpl

  const [templatePickerOpen, setTemplatePickerOpen] = useState(false)

  // ── 跨 hook 操作 ──

  // ── 切换模式：非 ComfyUI 后端仅保留文生图 ──
  const handleSwitchMode = useCallback((m: 'txt2img' | 'img2img' | 't2v') => {
    setMode(m)
    setResults([])
    setLightboxIndex(-1)
  }, [setMode, setResults, setLightboxIndex])

  // ── 结果操作 ──
  const handlePreviewResult = useCallback((i: number) => {
    const r = results[i]
    if (!r) return
    const historyIndex = history.findIndex((item) => item === r)
    setLightboxIndex(historyIndex >= 0 ? historyIndex : -1)
  }, [results, history, setLightboxIndex])

  const handleDownloadResult = useCallback(async (i: number) => {
    const r = results[i]
    if (!r) return
    const href = await resolveResultImage(r)
    if (!href) {
      message.warning('图片数据不可用，请重新生成')
      return
    }
    const a = document.createElement('a')
    a.href = href
    // T6-4.2：按实际媒体类型命名（t2v 输出 webp 则 .webp，不再固定 .mp4）
    a.download = downloadFileName(r)
    a.click()
  }, [results])

  const handleReuseResult = useCallback((i: number) => {
    const r = results[i]
    if (!r) return
    setPrompt(r.prompt)
    if (r.negative) setNegative(r.negative)
    if (r.seed) setSeed(r.seed)
    if (r.size) setSize(r.size)
  }, [results, setPrompt, setNegative, setSeed, setSize])

  const handleDeleteResult = useCallback((i: number) => {
    const r = results[i]
    if (!r) return
    setResults((prev) => prev.filter((item) => item !== r))
    setHistory((prev) => prev.filter((item) => item !== r))
  }, [results, setResults, setHistory])

  // ── 模板操作 ──
  const applyTemplate = useCallback((t: Template) => {
    setPrompt((p) => p ? p + '，' + t.prompt : t.prompt)
    const neg = t.negative
    if (neg) setNegative((n) => n ? n + ', ' + neg : neg)
    message.success(`已套用模板「${t.label}」`)
  }, [setPrompt, setNegative])

  // ── 引擎启停派生 ──
  const isLocalEngine = ['comfyui', 'herdsman', 'ollama'].includes(backend)
  const needsComfy = (mode === 't2v' && backend !== 'comfyui')
    || (mode === 'img2img' && backend !== 'comfyui' && backend !== 'herdsman')

  // ── 渲染 ──
  return (
    <div className="ig-studio" style={{ display: 'flex', flexDirection: 'column', flex: 1, minHeight: 0, overflow: 'hidden' }}>
      {/* 顶栏 */}
      <div className="ig-studio-header" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 12, flexShrink: 0 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <div style={{
            width: 34, height: 34, borderRadius: 10, display: 'flex', alignItems: 'center', justifyContent: 'center',
            background: 'rgba(var(--accent-rgb), 0.14)', border: '1px solid rgba(var(--accent-rgb), 0.25)',
            color: 'var(--color-primary)', fontSize: 16,
          }}>
            <PictureOutlined />
          </div>
          <div>
            <Typography.Title level={5} style={{ color: C('color-text'), margin: 0, fontSize: 16, fontWeight: 600, lineHeight: 1.2 }}>
              AI 绘梦
            </Typography.Title>
            <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: 3 }}>
              <StatusDot tone={engineStarting ? 'warn' : isLocalEngine ? (engineRunning ? 'ok' : 'idle') : 'ok'} />
              <span style={{ fontSize: 11, color: C('color-text-secondary'), whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', maxWidth: 280 }}>
                {engineStarting
                  ? '引擎启动中...'
                  : isLocalEngine
                    ? (engineRunning ? `${BACKEND_OPTIONS.find(b => b.value === backend)?.label || backend} 运行中` : '引擎未连接')
                    : `${BACKEND_OPTIONS.find(b => b.value === backend)?.label || backend} 云端`}
                {!engineStarting && model ? ` · ${model}` : ''}
              </span>
            </div>
          </div>
        </div>
        <Space size={6}>
          <Button type="text" size="small" icon={<FolderOpenOutlined />}
            onClick={handleOpenNovelDir} title="小说图片目录" aria-label="打开小说图片目录"
            style={{ color: C('color-text-secondary'), fontSize: 13, padding: '0 6px' }} />
          <Button type="text" size="small" icon={<FolderOpenOutlined />}
            onClick={handleOpenDir} title="生成图片目录" aria-label="打开生成图片目录"
            style={{ color: C('color-text-secondary'), fontSize: 13, padding: '0 6px' }} />
        </Space>
      </div>

      {/* 模式导航：文生图 / 图生图 / 文生视频 */}
      <div className="ig-mode-nav" role="tablist" aria-label="生成模式">
        <button
          type="button"
          role="tab"
          aria-selected={mode === 'txt2img'}
          className={`ig-mode-item${mode === 'txt2img' ? ' is-active' : ''}`}
          onClick={() => handleSwitchMode('txt2img')}
        >
          <PictureOutlined /> 文生图
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={mode === 'img2img'}
          className={`ig-mode-item${mode === 'img2img' ? ' is-active' : ''}`}
          onClick={() => handleSwitchMode('img2img')}
        >
          <SwapOutlined /> 图生图
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={mode === 't2v'}
          className={`ig-mode-item${mode === 't2v' ? ' is-active' : ''}`}
          onClick={() => handleSwitchMode('t2v')}
        >
          <VideoCameraOutlined /> 文生视频
        </button>
      </div>

      {/* 主工作区：左控制面板 + 中结果舞台 + 右历史胶片 */}
      <div className="ig-workspace" style={{ flex: 1, display: 'flex', gap: 12, minHeight: 0 }}>
        {/* 左栏 — 控制面板 */}
        <div className="ig-control-rail" style={{ width: 300, flexShrink: 0, overflowY: 'auto', overflowX: 'hidden', paddingRight: 2 }}>
          <ControlPanel
            mode={mode}
            prompt={prompt} negative={negative}
            onPromptChange={setPrompt} onNegativeChange={setNegative}
            onOpenTemplatePicker={() => setTemplatePickerOpen(true)}
            model={model} modelOptions={modelOptions}
            onModelChange={setModel}
            size={size} onSizeChange={setSize}
            customWidth={customWidth} customHeight={customHeight}
            onCustomWidthChange={setCustomWidth} onCustomHeightChange={setCustomHeight}
            seed={seed} onSeedChange={setSeed}
            count={count} onCountChange={setCount}
            initImage={initImage} onInitImageChange={setInitImage}
            denoise={denoise} onDenoiseChange={setDenoise}
            frames={frames} onFramesChange={setFrames}
            fps={fps} onFpsChange={setFps}
            selectedLoras={selectedLoras}
            loraOptions={backend === 'comfyui' ? loraOptions : []}
            loraLoading={loraLoading}
            loraError={loraError}
            onRefreshLoras={refreshComfyLoras}
            onLorasChange={setSelectedLoras}
            backend={backend} backendSwitching={backendSwitching}
            engineRunning={engineRunning} engineStarting={engineStarting} engineModelCount={engineModelCount}
            onSwitchBackend={handleSwitchBackend}
            onStartEngine={handleStartEngine} onStopEngine={handleStopEngine}
            sysStats={sysStats}
          />
        </div>

        {/* 中间 — 结果舞台 */}
        <div ref={canvasRef} className="ig-result-canvas" style={{ flex: 1, overflow: 'auto', minWidth: 0 }}>
          <ResultStage
            results={results} generating={generating} error={genError} mode={mode}
            initImage={initImage}
            onPreview={handlePreviewResult}
            onDownload={handleDownloadResult}
            onReuse={handleReuseResult}
            onDelete={handleDeleteResult}
            onRetry={handleGenerate}
            onOpenTemplatePicker={() => setTemplatePickerOpen(true)}
          />
        </div>

        {/* 右侧 — 任务中心：队列 / 最近结果 / 模板 */}
        <TaskCenter
          queueItems={queueItems}
          onClearQueue={() => setQueueItems([])}
          onCancelQueue={handleCancel}
          history={history}
          selectedIndex={lightboxIndex}
          onSelectHistory={setLightboxIndex}
          onClearHistory={() => { setHistory([]); setResults([]); setLightboxIndex(-1) }}
          onRegenerateMeta={handleRegenerateMeta}
          templates={TEMPLATES}
          customTemplates={customTemplates}
          onApplyTemplate={applyTemplate}
          onManageTemplates={() => setTemplatePickerOpen(true)}
        />
      </div>

      {/* 底部常驻生成栏 */}
      <GenerationBar
        mode={mode}
        backend={backend}
        model={model}
        count={count}
        frames={frames}
        fps={fps}
        generating={generating}
        elapsed={elapsed}
        lastTime={lastTime}
        pendingCount={pendingCount}
        queueTotal={queueItems.filter((q) => q.status === 'pending' || q.status === 'running').length}
        comfyProgress={comfyProgress}
        needsComfy={needsComfy}
        onGenerate={handleGenerate}
        onCancel={handleCancel}
      />

      {/* 灯箱 */}
      {lightboxIndex >= 0 && (
        <Lightbox
          results={history}
          index={lightboxIndex}
          characters={characters}
          onClose={() => setLightboxIndex(-1)}
          onIndexChange={setLightboxIndex}
          onDownload={handleDownload}
          onReuse={handleReuse}
          onSetPortrait={handleSetPortrait}
        />
      )}

      {/* 自定义模板弹窗 */}
      <CustomTemplateModal
        open={customModalOpen}
        editing={!!editingCustom}
        label={customLabel} onLabelChange={setCustomLabel}
        description={customDescription} onDescriptionChange={setCustomDescription}
        size={customSize} onSizeChange={setCustomSize}
        prompt={customPrompt} onPromptChange={setCustomPrompt}
        negative={customNegative} onNegativeChange={setCustomNegative}
        onSave={saveCustom}
        onCancel={() => setCustomModalOpen(false)}
      />

      {/* 模板选择弹窗 */}
      <TemplatePickerModal
        open={templatePickerOpen}
        onClose={() => setTemplatePickerOpen(false)}
        customTemplates={customTemplates}
        onSelect={applyTemplate}
        onAddCustom={openCustomAdd}
        onEditCustom={openCustomEdit}
        onDeleteCustom={deleteCustom}
      />
    </div>
  )
}

export default ImageGenPage
