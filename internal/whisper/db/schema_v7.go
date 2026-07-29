// Package db — Schema V7: 情绪涌现模块
// 100% 对齐 ackem src/main/db/schemaV7.ts
package db

const SchemaV7 = `
ALTER TABLE companion_state ADD COLUMN emergence_json TEXT;
`
