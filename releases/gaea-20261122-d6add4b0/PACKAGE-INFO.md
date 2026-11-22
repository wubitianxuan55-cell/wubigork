# gaea-20261122-d6add4b0

- 提交：`d6add4b0`（feat(gaea): 挂池五项清零+众神殿父子交互裁决执行批——F+G 两线合一）
- 构建：独立干净工作树 `.tmp/gaea-pkg-d6add4b0`（该提交检出），frontend `pnpm install --frozen-lockfile` + `npm run build`，`CGO_ENABLED=1 CC=ucrt64 wails build -s -ldflags "-s -w" -trimpath`（57.3s）
- 冒烟：payload 正式路径 `/api/health` 200（PID 19528 已回收）。
  坑复现（第二次实锤）：`.tmp` 路径直接跑 exe 被本机策略拦，须先落 releases 正式路径再冒烟
- 内容：F 批=沿江反馈挂池五项清零（P1-6 layout 版面机检门/P1-12 quarantine_list+目录级原子
  隔离 mtime 随迁判据/P1-7 wait sa_ 分流+lastAction 进度/_draft 豁免/RelIfWithin 三出口——
  沿江 14 条反馈至此全闭环）；G 批=众神殿 pan_1795306273871 父子交互裁决四 action（基线
  0% 报告率实锤/首刀 A complete_subtask artifacts+blockers+恒附 compact 完成块/回执面核验
  verified-unverified/第二刀四件：output_schema 事件面+死分支删除+dualSendMessage 双向+wait
  文案退役工具修正）；B1 blocking 回合不立项
- 剔除不抢：DSH 能力面板线（capabilities 前端族+todos.md 记账）+缓存命中线+运行态 meta+各 zip
- 安装：装机位 `C:\AI\gaea-app`（hash 同源核验见下）

## SHA256

| 文件 | 值 |
| --- | --- |
| payload/gaea.exe | BCBEA091EF210C4438FD37BF7350B7ABF19D0C066F07D43D82607A878BBFBF83 |
| payload/onnxruntime.dll | 68D0D5F502BE4509E2A16F932627A95137D748B2FAE519027F80125C0B1F550B |
| payload/sherpa-onnx-c-api.dll | 5DD36DBD3572846B72C4A40A4017C5A770160EE8E00DA5053BEBC8B2632501E8 |
| payload/sherpa-onnx-cxx-api.dll | 4C56327796CDB980DCBF90025C28093D89CA286088FB74C3B107F2F633E67FFE |
| gaea-20261122-d6add4b0.zip（payload 打包，留盘不入库） | 6EA95E3E493A047F7EFE7AC8C393216776565CE3C2E84C4A900441C6D4A6E2D7 |
| gaea-20261122-d6add4b0-src.zip（源码包，留盘不入库） | 23F8431071F53F2FF94C321803F3D765E5FC8BB17BF188998A300CB76DCCF267 |

三 DLL 与前包（gaea-20261122-a4b5b0a7 / gaea-20261121-01f88477 等）一致。
