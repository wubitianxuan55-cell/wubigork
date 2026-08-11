import React from 'react'
import { Button, Typography } from 'antd'
import { WarningOutlined } from '@ant-design/icons'
import { app } from '../gaea/lib/bridge'

interface Props {
  children: React.ReactNode
  fallback?: React.ReactNode
}

interface State {
  hasError: boolean
  error: Error | null
}

/**
 * ErrorBoundary — 捕获渲染异常，防止单页崩溃拖垮整个应用
 */
export class ErrorBoundary extends React.Component<Props, State> {
  state: State = { hasError: false, error: null }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, info: React.ErrorInfo) {
    // 渲染错误默认只会让 React 静默卸载整棵应用（窗口变死图、点不了）。
    // 这里把错误打到控制台并写入 gaea.log，下次可直接定位。
    console.error("[ErrorBoundary]", error, info.componentStack)
    try {
      void app
        .LogFrontendError(
          `[ErrorBoundary] ${error.message}\n${error.stack ?? ""}\n${info.componentStack ?? ""}`,
        )
        .catch(() => {})
    } catch {
      /* 日志通道不可用时静默 */
    }
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) return this.props.fallback
      return (
        <div style={{
          display: 'flex', flexDirection: 'column', alignItems: 'center',
          justifyContent: 'center', height: '100%', minHeight: 300,
          padding: 40, color: 'var(--color-text-secondary)',
        }}>
          <WarningOutlined style={{ fontSize: 48, color: '#f87171', marginBottom: 16 }} />
          <Typography.Title level={5} style={{ color: 'var(--color-text)', margin: 0 }}>
            页面渲染出错
          </Typography.Title>
          <Typography.Text style={{ color: 'var(--color-text-secondary)', marginBottom: 16 }}>
            {this.state.error?.message || '未知错误'}
          </Typography.Text>
          <Button onClick={() => this.setState({ hasError: false, error: null })}>
            重试
          </Button>
        </div>
      )
    }
    return this.props.children
  }
}
