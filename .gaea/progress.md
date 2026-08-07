# 任务进度

> 最后更新: 2026-08-07 10:15:38

## 2.0 P0 基线加固（✅ 已完成）

- frontend/package.json 重建并纳入版本控制（基线可构建）
- 根 dist/ 由前端构建产出，go build/vet/test 全绿
- scripts/ci.ps1 基线闸门（build/vet/test/frontend）
- E01-E24 评估集重建：docs/evaluation-set.md（可达性标注）
- 首批自动化回归：E11 工作区沙箱 / E06 解析合并去重 / E01 工作流稳定
- 基线修复：empty.pdf 坏夹具重建、parseLoraNames map 顺序稳定、findChangelogPath cwd 优先（E21 类测试隔离）
- 基线：main v1.21.0（9093930），P0 在 codex/gaea2-p0-baseline 分支

| 状态 | 任务 |
|------|------|
| ⬜ | Write tests |
| ✅ | Add parser |
