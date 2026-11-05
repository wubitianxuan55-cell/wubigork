import React, { useCallback, useEffect, useState } from 'react'
import { Alert, Button, Input, Space, Tag, Typography, message } from 'antd'
import { ApiOutlined, SaveOutlined, SearchOutlined } from '@ant-design/icons'
import SettingsSection from './SettingsSection'
import {
  webSearchSettings,
  saveWebSearchSettings,
  testWebSearchConnection,
  type WebSearchSettingsView,
  type WebSearchTestResult,
} from '../../api/settings'

const { Text } = Typography

/** SearchPanel — 网络搜索主源设置（众神殿刀3 主源向导）。三源任配其一即为主源：
 * Tavily（tavily.com 免费注册，1000 次/月）/ Brave（api.search.brave.com，2000 次/月）/
 * 自建 SearXNG（docker run -d -p 8080:8080 searxng/searxng）。key 落 ~/.env（与 CLI
 * 同源），保存即热生效；未配置时免费兜底（Bing）仍可用，结果会如实标注降级。
 * 检索核聊天与办公两板块共用同一条链。文案为 zh 直出（locale 三文件他方在途占用，
 * i18n 键待其收口后补）。 */
const SearchPanel: React.FC = () => {
  const [view, setView] = useState<WebSearchSettingsView | null>(null)
  const [tavilyKey, setTavilyKey] = useState('')
  const [braveKey, setBraveKey] = useState('')
  const [searxngURL, setSearxngURL] = useState('')
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [testResult, setTestResult] = useState<WebSearchTestResult | null>(null)

  const reload = useCallback(async () => {
    setLoading(true)
    setLoadError('')
    try {
      const v = await webSearchSettings()
      setView(v)
      setSearxngURL(v.local_searxng_url || '')
      setTavilyKey('')
      setBraveKey('')
    } catch (e) {
      setLoadError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { void reload() }, [reload])

  const save = async () => {
    setSaving(true)
    try {
      await saveWebSearchSettings(tavilyKey.trim(), braveKey.trim(), searxngURL.trim())
      message.success('搜索主源设置已保存并即时生效')
      setTestResult(null)
      await reload()
    } catch (e) {
      message.error(e instanceof Error ? e.message : String(e))
    } finally {
      setSaving(false)
    }
  }

  const test = async () => {
    setTesting(true)
    setTestResult(null)
    try {
      setTestResult(await testWebSearchConnection())
    } catch (e) {
      message.error(e instanceof Error ? e.message : String(e))
    } finally {
      setTesting(false)
    }
  }

  const readyTag = (ready: boolean | undefined, label: string) =>
    ready ? <Tag color="success">{label} 已就绪</Tag> : <Tag>{label} 未配置</Tag>

  return (
    <SettingsSection
      title="网络搜索"
      desc="主源向导：聊天与办公共用的联网检索核，配一个主源即显著提升命中率"
      icon={<SearchOutlined />}
    >
      {loadError && (
        <Alert type="error" showIcon message={`读取搜索设置失败：${loadError}`} style={{ marginBottom: 12 }} />
      )}
      <div style={{ marginBottom: 12, display: 'flex', gap: 6, flexWrap: 'wrap' }}>
        {loading ? <Tag>读取中…</Tag> : (
          <>
            {readyTag(view?.tavily_ready, 'Tavily')}
            {readyTag(view?.brave_ready, 'Brave')}
            {readyTag(view?.searxng_ready, '自建 SearXNG')}
            {view?.has_primary
              ? <Tag color="processing">主源已配置</Tag>
              : <Tag color="warning">未配置主源（仅免费兜底）</Tag>}
          </>
        )}
      </div>

      <Space direction="vertical" style={{ width: '100%' }} size={8}>
        <div>
          <Text type="secondary">Tavily API key（tavily.com 免费注册，1000 次/月）</Text>
          <Input.Password
            value={tavilyKey}
            onChange={(e) => setTavilyKey(e.target.value)}
            placeholder={view?.tavily_ready ? '已配置（输入新 key 可替换；留空清除）' : 'tvly-…'}
            autoComplete="off"
          />
        </div>
        <div>
          <Text type="secondary">Brave Search API key（api.search.brave.com，2000 次/月）</Text>
          <Input.Password
            value={braveKey}
            onChange={(e) => setBraveKey(e.target.value)}
            placeholder={view?.brave_ready ? '已配置（输入新 key 可替换；留空清除）' : 'BSA…'}
            autoComplete="off"
          />
        </div>
        <div>
          <Text type="secondary">自建 SearXNG 地址（如 http://localhost:8080，docker 可达时）</Text>
          <Input
            value={searxngURL}
            onChange={(e) => setSearxngURL(e.target.value)}
            placeholder="http://localhost:8080"
          />
        </div>
        <Space>
          <Button icon={<SaveOutlined />} type="primary" loading={saving} onClick={save}>保存</Button>
          <Button icon={<ApiOutlined />} loading={testing} onClick={test}>测试连接</Button>
        </Space>
        <Text type="secondary" style={{ fontSize: 12 }}>
          保存后立即生效，无需重启；未配置主源时免费兜底（Bing）仍可用，结果会如实标注降级。
        </Text>
        {testResult && (
          <Alert
            type={testResult.ok ? (testResult.state === 'ok' ? 'success' : 'warning') : 'error'}
            showIcon
            message={
              testResult.ok
                ? `连通正常（${testResult.state}${testResult.degraded_reason ? `：${testResult.degraded_reason}` : ''}，${testResult.elapsed_ms}ms）`
                : `未获得可用结果（${testResult.error || testResult.state}，${testResult.elapsed_ms}ms）`
            }
            description={
              <div style={{ fontSize: 12 }}>
                {testResult.engines?.length > 0 && (
                  <div>引擎：{testResult.engines.map((e) => `${e.name}:${e.outcome}(${e.elapsed_ms}ms)`).join('；')}</div>
                )}
                {testResult.sample?.length > 0 && (
                  <div>样例：{testResult.sample.join('；')}</div>
                )}
                {testResult.hint && <div>{testResult.hint}</div>}
              </div>
            }
          />
        )}
      </Space>
    </SettingsSection>
  )
}

export default SearchPanel
