// Package db — Schema V12: 追踪按角色会话隔离
// turn_traces 增加 session_id（此前无归属列，无法按角色查看/隔离追踪）
package db

const SchemaV12 = `
ALTER TABLE turn_traces ADD COLUMN session_id TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_traces_session ON turn_traces(session_id);
`
