import React, { useEffect, useState } from 'react'
import { Typography, Card, Progress, Row, Col, Spin, Empty, Statistic, Tooltip } from 'antd'
import {
  EditOutlined, BookOutlined, UserOutlined, EnvironmentOutlined,
  FireOutlined, AimOutlined,
} from '@ant-design/icons'
import { C } from '../../utils/theme'

/**
 * WritingDashboard — 写作仪表盘 2.0
 *
 * 展示：统计总览 + 每日字数柱状图 + 成就系统 + 章节字数分布
 */
interface Achievement {
  id: string; name: string; description: string; icon: string
  unlocked: boolean; progress: number; target: number
}

interface DashboardData {
  total_words: number; chapter_count: number; avg_words: number
  character_count: number; location_count: number
  daily_goal: number; today_words: number; goal_progress: number
  streak_days: number; completed_days: number
  chapter_word_counts: { chapter_num: number; title: string; word_count: number }[]
  achievements: Achievement[]
}

const WritingDashboard: React.FC = () => {
  const [data, setData] = useState<DashboardData | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const load = async () => {
      setLoading(true)
      try {
        // @ts-ignore
        const result = await window.go.app.App.GetDashboard(2000)
        setData(result)
      } catch (_) {
        setData(null)
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [])

  if (loading) return <div style={{ textAlign: 'center', padding: 40 }}><Spin tip="加载仪表盘..." /></div>
  if (!data) return <Empty description="请先打开小说" />

  const maxChapterWords = Math.max(...data.chapter_word_counts.map(c => c.word_count), 1)

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* ── 总览统计 ── */}
      <Row gutter={[12, 12]}>
        {[
          { icon: <EditOutlined />, label: '总字数', value: data.total_words.toLocaleString(), color: '#4ade80' },
          { icon: <BookOutlined />, label: '章节数', value: String(data.chapter_count), color: '#60a5fa' },
          { icon: <UserOutlined />, label: '角色数', value: String(data.character_count), color: '#f59e0b' },
          { icon: <EnvironmentOutlined />, label: '地点数', value: String(data.location_count), color: '#c084fc' },
          { icon: <FireOutlined />, label: '连续天数', value: String(data.streak_days), color: '#f87171' },
        ].map((stat, i) => (
          <Col span={4} key={i} style={i === 4 ? { flex: 1 } : undefined}>
            <Card size="small" style={{ textAlign: 'center', background: 'var(--bg-glass)' }}>
              <div style={{ fontSize: 20, color: stat.color }}>{stat.icon}</div>
              <Statistic value={stat.value} valueStyle={{ fontSize: 16, color: 'var(--color-text)' }} />
              <Typography.Text type="secondary" style={{ fontSize: 10 }}>{stat.label}</Typography.Text>
            </Card>
          </Col>
        ))}
      </Row>

      {/* ── 每日目标进度 ── */}
      <Card size="small" style={{ background: 'var(--bg-glass)' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
          <Typography.Text strong style={{ fontSize: 13 }}><AimOutlined /> 今日目标</Typography.Text>
          <Typography.Text type="secondary" style={{ fontSize: 11 }}>
            {data.today_words.toLocaleString()} / {data.daily_goal.toLocaleString()} 字
          </Typography.Text>
        </div>
        <Progress
          percent={Math.round(data.goal_progress)}
          strokeColor={data.goal_progress >= 100 ? '#4ade80' : '#f59e0b'}
          trailColor="var(--bg-deep)"
          format={pct => `${pct}%`}
        />
      </Card>

      {/* ── 章节字数柱状图 ── */}
      <Card size="small" style={{ background: 'var(--bg-glass)' }} title={<Typography.Text strong style={{ fontSize: 13 }}>📊 章节字数分布</Typography.Text>}>
        <div style={{ display: 'flex', alignItems: 'flex-end', gap: 4, height: 120, padding: '8px 0' }}>
          {data.chapter_word_counts.map(ch => {
            const pct = (ch.word_count / maxChapterWords) * 100
            return (
              <Tooltip key={ch.chapter_num} title={`第${ch.chapter_num}章: ${ch.word_count.toLocaleString()}字`}>
                <div style={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 4 }}>
                  <div
                    style={{
                      width: '100%',
                      maxWidth: 40,
                      height: `${Math.max(pct, 2)}%`,
                      background: `linear-gradient(to top, ${C('color-accent')}, ${C('color-accent')}80)`,
                      borderRadius: '4px 4px 2px 2px',
                      transition: 'height 300ms ease',
                      cursor: 'pointer',
                    }}
                  />
                  <span style={{ fontSize: 8, opacity: 0.5, writingMode: 'vertical-rl' }}>
                    {ch.chapter_num}
                  </span>
                </div>
              </Tooltip>
            )
          })}
        </div>
      </Card>

      {/* ── 成就系统 ── */}
      <Card size="small" style={{ background: 'var(--bg-glass)' }} title={<Typography.Text strong style={{ fontSize: 13 }}>🏆 写作成就</Typography.Text>}>
        <Row gutter={[8, 8]}>
          {data.achievements.map(ach => (
            <Col span={6} key={ach.id}>
              <div
                style={{
                  padding: 10,
                  borderRadius: 'var(--radius-md)',
                  border: '1px solid var(--border-subtle)',
                  background: ach.unlocked ? 'var(--bg-elevated)' : 'transparent',
                  opacity: ach.unlocked ? 1 : 0.4,
                  textAlign: 'center',
                  transition: 'all 200ms ease',
                }}
              >
                <div style={{ fontSize: 24, marginBottom: 4 }}>{ach.icon}</div>
                <Typography.Text strong style={{ fontSize: 11, display: 'block' }}>
                  {ach.name}
                </Typography.Text>
                <Typography.Text type="secondary" style={{ fontSize: 9 }}>
                  {ach.description}
                </Typography.Text>
                <Progress
                  percent={Math.round((ach.progress / ach.target) * 100)}
                  size="small"
                  showInfo={false}
                  strokeColor={ach.unlocked ? '#4ade80' : '#f59e0b'}
                  style={{ marginTop: 4 }}
                />
                <Typography.Text type="secondary" style={{ fontSize: 8 }}>
                  {ach.progress.toLocaleString()} / {ach.target.toLocaleString()}
                </Typography.Text>
              </div>
            </Col>
          ))}
        </Row>
      </Card>
    </div>
  )
}

export default WritingDashboard
