package db

// SchemaV15 记忆/情节按会话恢复补索引（2026-09-19 审计 P2）：V12 给
// turn_traces 补了 idx_traces_session（同 pattern），facts/episodes 两表的
// source_session_id 等值过滤（LoadFactsFromDBForSession/LoadEpisodesFromDB-
// ForSession，每会话恢复触发）漏建索引，每次恢复全表扫。
const SchemaV15 = `
CREATE INDEX IF NOT EXISTS idx_facts_session ON memory_facts(source_session_id);
CREATE INDEX IF NOT EXISTS idx_episodes_session ON episodes(source_session_id);
`
