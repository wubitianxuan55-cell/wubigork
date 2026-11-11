# gaea-contextup2-20261112-332976d1

- 提交：332976d1（feat/context-upgrade 分支头，完善批三刀 332976d1 + 本包记录）
- 内容：assistant 三拆消费（StatsCard 四 tile+TokenCard 分解行）+ 洞察统计行
  点击过滤会话（InsightsSession 名称清单）+ 趋势悬停 Actual；刀2 遗留测试
  夹具 bug 修正（entry Ts 硬编码）
- 构建：.tmp/wt-ctx-polish 干净工作树（npm build + wails build -s -trimpath，
  ucrt64+CGO+系统 TEMP）；装机 546B0A83B78F… PID 47960>构建 02:01:15，冒烟 200
- 三 DLL 与上包一致；zip/tar 磁盘保留不入库（P2-3）
