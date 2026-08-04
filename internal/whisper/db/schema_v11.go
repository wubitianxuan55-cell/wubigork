// Package db — Schema V11: FTS 表独立化重建
// 修复 V2 外部内容表缺陷：列名与主表不匹配（fact_id vs id）+ MATCH 需回查 content 表。
// 重建为独立 FTS5 表（无 content=），索引数据由 repos.RebuildFactsFTS 显式全量同步，
// 列名保持 fact_id/episode_id（与 repos 函数一致）。
package db

const SchemaV11 = `
DROP TABLE IF EXISTS memory_facts_fts;
DROP TABLE IF EXISTS episodes_fts;

CREATE VIRTUAL TABLE IF NOT EXISTS memory_facts_fts USING fts5(
  fact_id,
  subject,
  summary,
  triggers_text
);

CREATE VIRTUAL TABLE IF NOT EXISTS episodes_fts USING fts5(
  episode_id,
  summary,
  keywords_text,
  dominant_emotion
);
`
