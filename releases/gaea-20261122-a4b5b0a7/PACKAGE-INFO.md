# gaea-20261122-a4b5b0a7

- 提交：`a4b5b0a7`（feat(gaea): 沿江技术标反馈闭环批——传输层根修+标书模板+跨章引用+图文成册对齐）
- 构建：独立干净工作树 `.tmp/gaea-pkg-a4b5b0a7`（该提交检出），frontend `pnpm install --frozen-lockfile` + `npm run build`，`CGO_ENABLED=1 CC=ucrt64 wails build -s -ldflags "-s -w" -trimpath`（54.7s）
- 冒烟：payload 正式路径 `/api/health` 200（PID 3984 已回收）。
  坑复现：`.tmp` 路径直接跑 exe 被本机策略拦（ExitCode 1/0xC0000135），须先落 releases 正式路径再冒烟
- 内容：沿江 14 条反馈 A~D 四批（信封断头根修/pageBreak val/ledger event/空参回显+guard 豁免/
  标书主题+静态目录两遍法/md_validate paths 全册+语义初筛/配额/锚点豁免/斜杠归一）+
  图文同步（wenxueExport 同 stem 修复+toc-spec 透传）+图文成册（wenxue_book_build）+十/十一批造价门复盘
- 剔除不抢：DSH 工具面板线（capabilities 前端族）+缓存命中线两件+运行态 meta
- 安装：装机位 `C:\AI\gaea-app`（hash 同源核验见 SHA256SUMS）

## SHA256

| 文件 | 值 |
| --- | --- |
| payload/gaea.exe | 6A0FDBE50D27220E86DFF4B095DF0874F1506848E7DC1C7151068BC6F8EA70E3 |
| payload/onnxruntime.dll | 68D0D5F502BE4509E2A16F932627A95137D748B2FAE519027F80125C0B1F550B |
| payload/sherpa-onnx-c-api.dll | 5DD36DBD3572846B72C4A40A4017C5A770160EE8E00DA5053BEBC8B2632501E8 |
| payload/sherpa-onnx-cxx-api.dll | 4C56327796CDB980DCBF90025C28093D89CA286088FB74C3B107F2F633E67FFE |
| gaea-20261122-a4b5b0a7.zip（payload 打包，留盘不入库） | E3646329E93EB0C721FD64F5FA33777E44925404CBA90BFF8CF00839BF22CB1A |
| gaea-20261122-a4b5b0a7-src.zip（源码包，留盘不入库） | A15F5FB29F44F6AB97D0CDDAB0173C39F5F5D256C180AE7F97AF0BC498D8B92F |

三 DLL 与上包（gaea-20261121-01f88477 / gaea-20261121-57daf182）一致。
