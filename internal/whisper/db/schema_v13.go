// Package db — Schema V13: 清理废弃的 weixin_* 表
// weixin_account/weixin_sync/weixin_context/weixin_seen 自 SchemaV9 建表后
// 全仓无任何读写代码（T6-9.2 已 grep 核实），属死表；此处随迁移链 DROP 移除。
package db

const SchemaV13 = `
DROP TABLE IF EXISTS weixin_account;
DROP TABLE IF EXISTS weixin_sync;
DROP TABLE IF EXISTS weixin_context;
DROP TABLE IF EXISTS weixin_seen;
`

