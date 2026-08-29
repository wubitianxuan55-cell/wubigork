import { wailsApp } from '../lib/wailsApp';
import React, { useState, useRef, useCallback, useEffect } from 'react'
import { Button, Space, Tooltip, message, Progress, Typography } from 'antd'
import {
  SoundOutlined, PauseOutlined, PlayCircleOutlined,
  StopOutlined, LoadingOutlined,
} from '@ant-design/icons'
import { C } from '../utils/theme'

interface TTSPlayerProps {
  getText: () => string
  onStatusChange?: (playing: boolean) => void
  /** 每个音频块开始合成/播放时回调对应句子（朗读跟随高亮用） */
  onSentence?: (sentence: string) => void
  /** 停止/重新开始时清除朗读高亮 */
  onClear?: () => void
}

interface AudioChunk {
  audio: Uint8Array
  mimeType: string
}

/** tts-stream 事件动态载荷（最小消费面） */
interface TTSStreamEvent {
  type?: string
  index?: number
  total?: number
  engine?: string
  audio?: string
  mimeType?: string
  text?: string
  done?: boolean
  error?: string
}

const TTSPlayer: React.FC<TTSPlayerProps> = ({ getText, onStatusChange, onSentence, onClear }) => {
  const [playing, setPlaying] = useState(false)
  const [loading, setLoading] = useState(false)
  const [progress, setProgress] = useState({ index: 0, total: 0 })
  const [engine, setEngine] = useState<string | null>(null)
  const audioRef = useRef<HTMLAudioElement | null>(null)
  const audioUrlRef = useRef<string | null>(null)
  const chunkQueue = useRef<AudioChunk[]>([])
  const playingIndex = useRef(0)
  const stoppedRef = useRef(false)
  const pausedRef = useRef(false)
  const onStatusChangeRef = useRef(onStatusChange)
  onStatusChangeRef.current = onStatusChange
  const onSentenceRef = useRef(onSentence)
  onSentenceRef.current = onSentence
  const onClearRef = useRef(onClear)
  onClearRef.current = onClear

  // 清理音频资源
  const cleanupAudio = useCallback(() => {
    if (audioUrlRef.current) {
      URL.revokeObjectURL(audioUrlRef.current)
      audioUrlRef.current = null
    }
    if (audioRef.current) {
      audioRef.current.pause()
      audioRef.current.src = ''
      audioRef.current = null
    }
  }, [])

  // 播放队列中的下一个音频块
  const playNextChunk = useCallback(() => {
    if (stoppedRef.current || pausedRef.current) return

    cleanupAudio()

    if (playingIndex.current >= chunkQueue.current.length) {
      // 队列播完了，但可能还有更多块在生成中
      setPlaying(true) // 保持播放状态等待新块
      return
    }

    const chunk = chunkQueue.current[playingIndex.current]
    playingIndex.current++

    const blob = new Blob([chunk.audio as unknown as BlobPart], { type: chunk.mimeType || 'audio/wav' })
    const url = URL.createObjectURL(blob)
    audioUrlRef.current = url

    const audio = new Audio(url)
    audioRef.current = audio

    audio.onended = () => {
      playNextChunk()
    }

    audio.onerror = () => {
      // 播放失败，跳过此块继续下一个
      playNextChunk()
    }

    audio.play().catch(() => {
      // play() 也可能失败，但 onerror 已处理，此处忽略
    })
  }, [cleanupAudio])

  // 将音频块加入队列并触发播放
  const enqueueChunk = useCallback((chunk: AudioChunk) => {
    const wasIdle = chunkQueue.current.length === 0
    chunkQueue.current.push(chunk)
    if (wasIdle) {
      playNextChunk()
    }
  }, [playNextChunk])

  // 监听 Wails TTS 流式事件
  useEffect(() => {
    if (!window.runtime?.EventsOn) return

    window.runtime.EventsOn('tts-stream', (ev: TTSStreamEvent) => {
      if (!ev?.type) return

      if (ev.type === 'progress') {
        setProgress({ index: ev.index ?? 0, total: ev.total ?? 0 })
        setLoading(true)
        if (ev.text) onSentenceRef.current?.(ev.text)
      } else if (ev.type === 'chunk') {
        if (ev.engine) setEngine(ev.engine)
        if (ev.audio && !stoppedRef.current) {
          const bytes = Uint8Array.from(atob(ev.audio), c => c.charCodeAt(0))
          enqueueChunk({ audio: bytes, mimeType: ev.mimeType || 'audio/wav' })
          setProgress({ index: (ev.index ?? 0) + 1, total: ev.total ?? 0 })
          setPlaying(true)
          onStatusChangeRef.current?.(true)
        }
        if (ev.done) {
          setLoading(false)
        }
      } else if (ev.type === 'error') {
        message.error(ev.error || '语音合成失败')
        setLoading(false)
        setPlaying(false)
        setEngine(null)
        onStatusChangeRef.current?.(false)
      } else if (ev.type === 'done') {
        setLoading(false)
        setEngine(null)
      }
    })

    return () => {
      try {
        window.runtime?.EventsOff?.('tts-stream')
      } catch (_) {}
    }
  }, [enqueueChunk])

  const handlePlay = async () => {
    const text = getText()
    if (!text || text.trim().length === 0) {
      message.warning('当前没有可朗读的内容')
      return
    }

    cleanupAudio()
    chunkQueue.current = []
    playingIndex.current = 0
    stoppedRef.current = false
    pausedRef.current = false
    setProgress({ index: 0, total: 0 })
    setEngine(null)
    setLoading(true)
    onClearRef.current?.()

    try {
      await wailsApp().TTSSpeakStreaming(text)
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '语音合成失败')
      setLoading(false)
    }
  }

  const handlePause = () => {
    pausedRef.current = true
    audioRef.current?.pause()
    setPlaying(false)
    onStatusChangeRef.current?.(false)
  }

  const handleStop = () => {
    stoppedRef.current = true
    pausedRef.current = false
    chunkQueue.current = []
    playingIndex.current = 0
    cleanupAudio()
    setPlaying(false)
    setLoading(false)
    setProgress({ index: 0, total: 0 })
    setEngine(null)
    onStatusChangeRef.current?.(false)
    onClearRef.current?.()
  }

  const handleResume = () => {
    pausedRef.current = false
    if (chunkQueue.current.length > 0) {
      playNextChunk()
      setPlaying(true)
      onStatusChangeRef.current?.(true)
    }
  }

  const progressPercent = progress.total > 0 ? Math.round((progress.index / progress.total) * 100) : 0

  return (
    <Space size={4} align="center">
      {engine && (playing || loading) && (
        <Typography.Text style={{ fontSize: 9, color: 'var(--color-text-secondary)', background: 'var(--bg-elevated)', borderRadius: 4, padding: '1px 6px', lineHeight: '18px' }}>
          {engine === 'xai' ? 'xAI' : engine === 'edge' ? 'Edge' : engine === 'sapi' ? 'SAPI' : engine}
        </Typography.Text>
      )}
      {loading || playing ? (
        <>
          {loading && !playing ? (
            <Button type="text" size="small" disabled
              icon={<LoadingOutlined spin style={{ color: '#60a5fa' /* hex-exempt 品牌识别色 */ }} />}
              style={{ color: C('color-text-secondary'), fontSize: 11 }}
            >
              {progress.total > 0 ? `${progress.index}/${progress.total}` : '合成中'}
            </Button>
          ) : (
            <>
              <Tooltip title="暂停">
                <Button type="text" size="small"
                  icon={<PauseOutlined style={{ color: 'var(--color-warning)' }} />}
                  onClick={handlePause}
                />
              </Tooltip>
              <Tooltip title="继续">
                <Button type="text" size="small"
                  icon={<PlayCircleOutlined style={{ color: 'var(--color-success)' }} />}
                  onClick={handleResume}
                />
              </Tooltip>
            </>
          )}
          <Tooltip title="停止">
            <Button type="text" size="small"
              icon={<StopOutlined style={{ color: 'var(--color-destructive)' }} />}
              onClick={handleStop}
            />
          </Tooltip>
          {progress.total > 0 && (
            <Progress
              percent={progressPercent}
              size="small"
              style={{ width: 60, margin: 0 }}
              strokeColor="var(--color-success)"
              showInfo={false}
            />
          )}
        </>
      ) : (
        <Tooltip title="流式朗读（逐句生成并播放）">
          <Button type="text" size="small"
            icon={<SoundOutlined style={{ color: 'var(--color-success)' }} />}
            onClick={handlePlay}
            style={{ fontSize: 12 }}
          >
            朗读
          </Button>
        </Tooltip>
      )}
    </Space>
  )
}

export default TTSPlayer
