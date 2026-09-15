# 记忆质量受控测评题集（Memory Eval Set · 三脑 + 经验习得）

> 用途：阶段七 7.1-1 记忆质量可评测的**零外部依赖**受控题集（docs/gaea-stage7-plan-2026-09.md §1）。
> 与 `docs/retrieval-eval-set.md`（书斋四库·运行态·需 Herdsman embedding）分层：
> 本题集配 `internal/memoryeval` 内置种子语料，CI（`go test ./internal/memoryeval`）
> 一键跑出四域基线。
>
> - **story**：项目 StoryMemory（internal/memory 纯 Go BM25）——章节摘要种子（修真长篇，
>   人物/地名一致）+ 市井支线陪衬，needle-in-haystack。
> - **work**：办公事实/知识（internal/gaea/bm25 零 token 零网络）——组价/文档/造价纪律
>   条目 + 行政事务陪衬。
> - **persona**：轻语人格记忆（whisper SQLite FTS + 中文 2-gram LIKE 降级）——关于用户
>   的事实 + 无关事实陪衬；查询按生产路径真实能力设计（含独特二字组）。
> - **experience**：经验习得题集（对齐 LongMemEval-V2「习得未来任务所需经验」方向，
>   ≥20 题）——查询为未来任务口吻，期望命中 = 承载该经验的记忆条目；为 7.2 技能结晶
>   的复用成功率度量预置基建。
>
> ## 匹配规则
> - 每条查询标注 `expected`：种子语料中的稳定 ID（即报告标签），**精确相等**记命中
>   （种子 ID 由 seeds.go 提供，漂移由测试硬断言兜住）。
> - 单条 recall = 命中预期数 / 预期总数（取前 10 命中计分，recall@10）；
>   precision = 命中列表中相关条目占比（均值口径）。
> - 门槛：recall@10 ≥ 0.8（沿 T5-6 口径）。CI 判定=**软告警不阻断**（质量退化告警、
>   结构性错误如 ID 漂移/题量不足才硬失败）。
>
> 基线（2026-09-15，Windows x64，go test 单机跑批）：见本档 §基线数字。

## 查询集总览

| 域 | 题数 | 说明 |
|---|---|---|
| story | 11 | 含 1 道双预期（赤炎谷两段经历） |
| work | 12 | 含 1 道双预期（周报与纪要纪律） |
| persona | 10 | 按生产 FTS/2-gram 真实能力设计 |
| experience | 22 | LongMemEval-V2 方向对齐（≥20 达标） |

```json
{
  "story": [
    {"query": "林昭在哪里初遇沈青梧", "expected": ["第3章·药庐初见"]},
    {"query": "青冥剑的来历", "expected": ["第7章·剑冢得剑"]},
    {"query": "落霞渡口伏击是谁出手相救", "expected": ["第12章·落霞渡伏击"]},
    {"query": "听雨楼的情报交易", "expected": ["第15章·听雨楼交易"]},
    {"query": "赤炎谷的两段经历", "expected": ["第18章·赤炎谷秘藏", "第45章·金丹初成"]},
    {"query": "云隐城被围谁守的南门", "expected": ["第21章·云隐之围"]},
    {"query": "沈青梧中了什么毒怎么解", "expected": ["第28章·寒毒入体"]},
    {"query": "裴照为什么阵前倒戈", "expected": ["第34章·裴照反戈"]},
    {"query": "玄夜司首领现身说了什么", "expected": ["第38章·司主现世"]},
    {"query": "九曲灵阵是怎么破的", "expected": ["第40章·九曲破阵"]},
    {"query": "沈青梧的身世", "expected": ["第50章·青梧身世"]}
  ],
  "work": [
    {"query": "组价确认前要复核什么", "expected": ["证据链复核纪律"]},
    {"query": "混凝土信息价从哪里来", "expected": ["信息价导入链"]},
    {"query": "周报格式偏好", "expected": ["周报三段式"]},
    {"query": "投标文件要准备几份", "expected": ["投标文件编制规范"]},
    {"query": "台班询价怎么走", "expected": ["询价飞轮口径"]},
    {"query": "多文件任务先做什么", "expected": ["DAG 编排纪律"]},
    {"query": "交付 PPT 前的检查", "expected": ["幻灯成品直出清单"]},
    {"query": "钢筋图集中标注和原位标注", "expected": ["平法标注要点"]},
    {"query": "基线怎么命名", "expected": ["基线命名约定"]},
    {"query": "会议纪要归档", "expected": ["纪要归档路径"]},
    {"query": "大 Excel 导入注意", "expected": ["大表导入分块"]},
    {"query": "周报与纪要的纪律", "expected": ["周报三段式", "纪要归档路径"]}
  ],
  "persona": [
    {"query": "用户喝咖啡的习惯", "expected": ["咖啡偏好"]},
    {"query": "猫叫什么名字", "expected": ["橘猫汤圆"]},
    {"query": "作息习惯几点睡", "expected": ["晚睡作息"]},
    {"query": "家乡在哪里", "expected": ["川南家乡"]},
    {"query": "最好的朋友是谁", "expected": ["挚友阿珍"]},
    {"query": "职场八卦不爱聊", "expected": ["八卦边界"]},
    {"query": "生日几月", "expected": ["十一月生日"]},
    {"query": "喜欢什么音乐", "expected": ["后摇偏好"]},
    {"query": "午休多久", "expected": ["午休二十分钟"]},
    {"query": "语音消息烦不烦", "expected": ["语音条嫌烦"]}
  ],
  "experience": [
    {"query": "给领导发周报前注意什么", "expected": ["周报先结论后过程"]},
    {"query": "组价对不上差在哪", "expected": ["量差价差排查顺序"]},
    {"query": "导出 PPT 字体丢了怎么办", "expected": ["PPT 字体嵌入"]},
    {"query": "客户砍价让多少合适", "expected": ["让价守三"]},
    {"query": "批量改名被占用报错", "expected": ["改名退避重试"]},
    {"query": "台账排序前先做什么", "expected": ["台账先备份再排序"]},
    {"query": "招标澄清件什么时候发", "expected": ["澄清件提前四小时"]},
    {"query": "询价单要附什么", "expected": ["询价单附图纸版本"]},
    {"query": "敏感报价怎么发", "expected": ["敏感报价走导出件"]},
    {"query": "评审会材料怎么准备", "expected": ["评审会先发结论页"]},
    {"query": "文档里插图注意什么", "expected": ["图片先压缩"]},
    {"query": "客户资料怎么归档", "expected": ["按项目归档"]},
    {"query": "邮件附件太大发不出", "expected": ["长附件改链接"]},
    {"query": "Excel 日期比对出错", "expected": ["日期列转文本"]},
    {"query": "现场签证什么时候记", "expected": ["签证当场登记"]},
    {"query": "印章外带的要求", "expected": ["印章外带登记"]},
    {"query": "接口被限流怎么办", "expected": ["接口限流分页拉"]},
    {"query": "导出报表数据不全", "expected": ["导出先查筛选器"]},
    {"query": "群发通知前试一下", "expected": ["群发前先发自己"]},
    {"query": "设计变更怎么算数", "expected": ["变更单闭环"]},
    {"query": "报价单要写有效期吗", "expected": ["报价有效期写死"]},
    {"query": "双周复盘怎么列", "expected": ["双周复盘只列三件"]}
  ]
}
```

## 基线数字

2026-09-15 · Windows x64 · `go test ./internal/memoryeval -count=1`（种子语料固定，数字可复现）：

| 域 | 题数 | recall@10 | precision 均值 | 门槛 0.8 |
|---|---|---|---|---|
| story | 11 | **1.000** | 0.433 | PASS |
| work | 12 | **1.000** | 0.478 | PASS |
| experience | 22 | **1.000** | 0.661 | PASS |
| persona | 10 | **1.000** | 0.727 | PASS |

已知信号（评测产出的真实发现，非缺陷修复项）：
- **persona 域泛主语查询淹没**：查询「用户喝咖啡的习惯」经 2-gram LIKE 降级后，
  「用户」二字组命中全部事实（subject 通用值），top-10 被灌满、该条 precision 仅
  0.1——recall 靠 rowid 顺序侥幸保住。生产启示：persona 检索对含通用主语的
  自然问句排序弱（候选改进：按 2-gram IDF 加权或跳过通用主语词），先记录
  基线画像，不动引擎。
- story/work/experience 的 precision 均值 0.43~0.66：BM25 对多词查询返回的
  相关但不命中预期的条目（同主角/同主题章节）——recall 达标下属正常画像。
- 书斋四库（embedding 路径）基线由既有 `GaeaRetrievalEvalRun` 运行态测评覆盖，
  不在本档（需 Herdsman bge-m3 引擎）。
