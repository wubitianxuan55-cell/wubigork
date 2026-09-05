// 聊天 markdown 渲染缝的 GenUI 覆盖件：只替换 code/pre，保留默认样式。
import { isValidElement, type ReactNode } from 'react'
import type { Components } from 'react-markdown'
import { GenuiMarkdownFence, genuiFenceStateKey, isGenuiFenceLang } from '../../genui/markdownFence'
import type { GenuiScope } from '../../genui/scope'

export function buildMarkdownGenuiOverrides(
  scope: GenuiScope | null,
  sourceKey?: string,
): Components {
  const fenceKeyFor =
    scope !== null && sourceKey !== undefined
      ? (body: string): string | undefined => genuiFenceStateKey(scope, sourceKey, body)
      : undefined

  const GenuiCode = ({ className, children }: { className?: string; children?: ReactNode }) => {
    const text = String(children ?? '').replace(/\n$/, '')
    const match = /language-([\w-]+)/.exec(className ?? '')
    const lang = match?.[1]
    const isBlock = match !== null || text.includes('\n')
    if (lang !== undefined && isGenuiFenceLang(lang) && isBlock) {
      return (
        <div data-genui-host="true">
          <GenuiMarkdownFence code={text} stateKey={fenceKeyFor?.(text)} />
        </div>
      )
    }
    return <code className={className}>{children}</code>
  }

  const GenuiPre = ({ children }: { children?: ReactNode }) => {
    if (
      isValidElement(children) &&
      (children.props as Record<string, unknown>)['data-genui-host'] === 'true'
    ) {
      return <>{children}</>
    }
    return <pre>{children}</pre>
  }

  return {
    code: GenuiCode as unknown as Components['code'],
    pre: GenuiPre as unknown as Components['pre'],
  }
}
