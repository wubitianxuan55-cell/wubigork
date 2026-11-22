# gaea-memory-distill-20261122-98466698

- 提交：`98466698`（本包业务链=关联软件官方最新批 fc41b54a + 记忆中枢蒸馏 P1 4c1eb517 +
  P2/P3 34a550dc + 聊天席融合注入 1e50e689 + 完善批 52f5b187 + day-log 4ac120c7..98466698）
- 构建：独立干净工作树 `.tmp/gaea-pkg-98466698`（该提交检出），主树拷 DLL 三件套入
  `build/bin`（gen_bindings 子进程 PATH 依赖，System32 ORT 劫持坑），frontend
  `pnpm install --frozen-lockfile`（28.3s）+ `npm run build`（32.5s），
  `CGO_ENABLED=1 CC=ucrt64 wails build -s -ldflags "-s -w" -trimpath`（60.3s）
- 冒烟：payload 正式路径 `/api/health` 200（PID 18184 已回收）
- 内容：
  - 关联软件官方最新批：sherpa-onnx v2.0.1→**v2.0.2**（DLL 换装，本包三 DLL 与历届包
    **不一致**——sherpa 两件换 v2.0.2 新哈希）、ORT 钉 1.31.1（官方 GitHub 仓消失，
    不从野镜像拉）、官方 llama.cpp v1.0.0 nightly b12359 落 voice-v2\bin（qwen35 实弹），
    均不入包（引擎侧数据，包内只有 exe+三 DLL）
  - 记忆中枢蒸馏 P1：星云召回四臂融合（`GaeaMemoryRecall`+记忆中枢搜索「星云召回」组，
    臂徽章+分数 chip）
  - P2 FSRS 稳定性：StabilityDays（30d×1.5^引用封顶 240d，cite 事件聚合零迁移）+
    生命周期 additive 三字段+休眠徽章
  - P3 向量巩固臂：duplicates 面板收编语义余弦 ≥0.85 同义对（Source=vector，评审制不破）
  - 聊天席融合注入（用户拍板分席）：play 分区 RecallBlockFused（词法+图谱发现+置顶
    去重+稳定性衰减），办公 work/平铺逐字旧口径保缓存，文学独立引擎不在面
- 剔除不入库：zip/src.zip/payload 留盘；引擎与模型数据（voice-v2\bin、gaea-OCR、
  gaea-VOICE、gaea-ENGINES）不入仓
- 安装：装机位 `C:\AI\gaea-app`（与 payload 同码不同时间戳哈希，同源核验见下）

## SHA256

| 文件 | 值 |
| --- | --- |
| payload/gaea.exe | 6E8FB1DF00A3141ADBA48BC3D1D7486B4265D6ADF1C074D6CC79097E06280BEC |
| payload/onnxruntime.dll | 68D0D5F502BE4509E2A16F932627A95137D748B2FAE519027F80125C0B1F550B |
| payload/sherpa-onnx-c-api.dll | 408F9AB683D1B77CBF7F06D48BD3AF543ED783CC9093C19F520C7C8524619C8F |
| payload/sherpa-onnx-cxx-api.dll | DBAC55CECA8383FE28BD7C656FD125BEC60DD35D7DF1A3A28EA36A503EF6C562 |
| gaea-memory-distill-20261122-98466698.zip（payload 打包，留盘不入库） | 86DBD6819415C92A02C577AED15C1B5E900DCF2AD76328B26357F01A89526FC1 |
| gaea-memory-distill-20261122-98466698-src.zip（源码包，留盘不入库） | 48E50ABE071890606DB8889D9622BC19CE51EB02FC631EFE9B1B29A47D5152BA |
| 装机位 gaea.exe（同源不同时间戳，主树 19:29 构建） | 0BCAACE78EB7B1844C8C527B3458E0CBF34F224CA855BB9DDB71F1E9D08EF3A7 |

onnxruntime.dll 与历届包一致（1.31.1）；**sherpa 两 DLL 为 v2.0.2 新哈希**（与
gaea-20261122-d6add4b0 及更早包不同——go.mod bump fc41b54a 随本包首次进入发布件）。
