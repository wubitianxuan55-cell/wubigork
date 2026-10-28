# 当日总账（2026-10-29）：双版本日——本地 OCR 无限量线全线收官 + 亮色暖纸 + CI 清欠

> 批末=172f3b64；桌面 exe=build/bin/gaea.exe 55,513,600B SHA256 前缀 db51be26cfc24e61
> （桌面副本同哈希，含全部在册刀）。今日两发：**v4.526.0**（tag=157a4539，OCR 线
> 收官+造价清单库）与 **v4.527.0**（tag=42b28e65，暖纸+进度可见+CI 清欠）；tag 后
> 又五刀待随下版。

## 提交索引（17 笔，按线归组）

| 提交 | 线 | 一句话 |
|---|---|---|
| ecf52607 | 造价 | 清单库行级维护三件套（编辑/删除/手工新增，绑定复活 684→686，孤儿引用根修；10-28 拍板深夜落账） |
| 90b6fce3 | 知识库 | PDF 导入乱码根修：meaningfulText 双闸（符号垃圾行占比>30%+汉字占比≥2%），OCR 回退不再被假文本短路 |
| 0f3064e1 | OCR | uocr2 引擎计入（Unlimited-OCR-2=百度 3B MoE+SR-SWA；auto 链质量优先 uocr2→ovis→tesseract；常驻 python 服务 8138） |
| 3b251a63 | OCR | 多页批识别（PageBatchOCRProvider+5 页窗，跨页表格续排）+断连自愈+清洗器加固；核显 opt-in 定性 |
| 157a4539 | **发版** | **v4.526.0**（四提交整批；沙箱壳真机走查 PASS） |
| 608f6154 | OCR | 近空页守卫阈值 20→48（CLI 实弹：低清输入「仅页眉」退化形态） |
| 0b416027 | 知识库 | 导入进度可见（knowledge-import-progress 事件管道补齐+弹窗 N/总页）+scripts/uocr2 胶水入库 |
| 2c102f75 | OCR | 起步进度事件 done=0（批模式首窗前即时反馈） |
| 7d9de0a5 | CI | 全仓 CI 清欠到绿（lint 8 处含 SA4006 真 bug/测试工装 OCR 隔离/E26 锚迁 navigateBoard/docs 补登 13 篇） |
| 5c8d3ccd | 主题 | **亮色暖纸全量落地**（§7.6 销项：派生层 warmPaper，7×2 表零改动，AA 全预设重算，侧栏亮态改提亮后退） |
| 14b40e94 | 主题 | 暖纸契约 hex-exempt 五处（no-raw-hex 惯例）；硬编码底巡检零违例 |
| 42b28e65 | **发版** | **v4.527.0**（六刀同日整批） |
| c7e0e5fa | 子代理 | P2-G 深度记账辨伪销号（代码证据：元工具剔除→深度恒 1 无递归面） |
| 9937f8af | 审计清欠 | SuspendPrefix/ResumePrefix 孤儿机制退役（净 −100 行；V10.0 MCP 按会话禁用语义已被 spec 层物理过滤取代）；SetMemoryQueue=活线过时销项 |
| 5c678934 | CI | struct 字段对齐 gofmt（闸逮住，纯格式） |
| ce4619be | 测试 | PptxEditPanel initialSlide 满载假红**根修**（预选 effect 时间窗→waitFor；「连续同例红=挖根因」教训入册） |
| 172f3b64 | OCR | 清洗器标记逃逸收口（15 页全量实弹逮住：BLOCK…FORK 间含尖括号两形态，[^<]*→.*?） |

## 验证矩阵（OCR/知识线，全部真机证据）

- 单页 57.7s / 双页批 149.8s(CPU)·78.3s(GPU 实验) / **15 页全规模 1063s**（进度事件
  `0/15→5/15→10/15→15/15` 逐拍吻合，首中尾锚点全中）
- 三消费方：扫描件 PDF（知识库）✓ / 报价单图片（造价视觉 23 行全表三列全对）✓ /
  办公提取兜底腿（同管道低风险未单走）
- 全闭环：Preview→**Apply 落库**→List 查回（此前只验到 Preview）
- 亮态暖纸九页面逐页目检全协调（chat/gaea/cost/memoryhub/modelcenter/novel/
  imagegen/runtime/schedule）；暗态零触碰实证

## 本机 OCR 栈（不入仓库）

- **OvisOCR2**（次选引擎）：C:\AI\gaea-ocr（GGUF+llama-server Vulkan，8137）
- **Unlimited-OCR-2**（首选引擎）：C:\AI\unlimited-ocr2（6.7GB bf16+CPU venv+
  GPU 实验档 venv-rocm2；模型加载完才监听 8138；README 全录补丁与验收配方）
- 胶水代码版本控制副本：scripts/uocr2/（改实例须回拷同步）

## 新坑入册（详录各线记忆）

HF 远端代码改后须删 transformers_modules 缓存 / uocr2 测试须塞 GAEA_UOCR2_DIR 死值
防误拉真服务 / infer_multi 是 base 模式无切片勿用 / app 测试工装接新本地引擎必须同
步 OCR env 隔离（VkCache 落 TempDir 假红）/ ci.ps1 的 .tmp 卫生闸会删写进 .tmp 的
日志（重定向去 $TEMP）/ release.ps1 不含 lint+docs 守卫——发版后必须补跑全量 ci /
满载假红连续同例红=挖根因勿复跑过关 / bash -m 消息含尖括号标记会炸提交（走 -F 文
件）/ TheRock gfx1151 轮子 stamp 0502 HIP JIT bug、0128 老 stamp 可用 / PIL 重编码
PDF 降画质触发模型跳页（走查素材用原 PDF 直渲）/ model.infer 缺省 stream=True 流
式重印全文（server/CLI 都要 eval_mode=True）

## 余池（全部待用户拍板/实弹，无自主可动项）

- P2 两守：子代理 per-call 模型路由（等真实多模型需求）/ send_message（等长跑子代
  理需求+上游大改稳定）——池内原判维持
- 小说创作页状态条展开手感（实弹池）
- ComfyUI JobObject 兜底（观察池）；pptx 刀4 待反馈
- 下版发版：tag 后五刀（9937f8af/5c678934/ce4619be/172f3b64+c7e0e5fa docs）随下一
  列车
