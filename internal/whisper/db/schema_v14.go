package db

// SchemaV14 图谱情绪维度（v4.9，审计 §C 欠账收口）：knowledge_triples 增加
// 情绪标签/强度/效价列——三元组情绪随事实落库，前端按情绪着色。
// ALTER TABLE ADD COLUMN 带 NOT NULL DEFAULT，存量行自动回填默认值，安全。
const SchemaV14 = `
ALTER TABLE knowledge_triples ADD COLUMN emotion_label TEXT NOT NULL DEFAULT '';
ALTER TABLE knowledge_triples ADD COLUMN emotional_intensity REAL NOT NULL DEFAULT 0;
ALTER TABLE knowledge_triples ADD COLUMN valence REAL NOT NULL DEFAULT 0;
`
