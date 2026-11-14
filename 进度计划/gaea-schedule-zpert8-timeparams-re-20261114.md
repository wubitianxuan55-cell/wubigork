# 斑马进度 ZPert8 · 进度计划的「六个时标」（时间参数）逆向 · 2026-11-14

> **修正说明**：前两轮（`gaea-schedule-zpert8-timescale-re-20261114.md`）挖的是 **时标刻度设置**（时间轴刻度尺），
> 不是本文的「进度计划的六个时标」。本文是**计划的六个时间参数**专题，两者是不同的东西：
> 前者是**横轴怎么画**，后者是**计划算出了哪六个时间量**。
>
> **样本**：`C:\Program Files\ZebraTech\ZPert8\ZPert.exe`（7.0.706.0）
> SHA256 `C61972FAA6A1A08AB846037564CAAD7B2AB23ABCEE3A11B27885DBD24E094E68`
>
> **方法**：Ghidra 定点反编译 + `.rdata` UTF-8 键池直读 + `.text` 立即数（列 ID）扫描定位派发函数。

---

## 0 一页结论

| # | 结论 | 强度 |
|---|---|---|
| 1 | **六个时标 = 列 ID 10170–10220 的六个连续时间参数列**：最早开始 / 最早完成 / 最晚开始 / 最晚完成 / 最晚开始(自由) / 最晚完成(自由) | 实测 |
| 2 | 六列**数据类型统一为 4**（计算型数值，非日期），默认宽度 110/140，**ID 连续无空洞** → 是刻意设计的一组，不是散列凑出来的 | 实测 |
| 3 | 六个参数**各有一个 JSON 键**，集中在同一段键池 `0x1408d0458..0x1408d0608` | 实测 |
| 4 | 键名有拼写错误：`earlist_start_time` / `earlist_stop_time`（不是 earliest）—— 检索时容易漏 | 实测 |
| 5 | **总时差 / 自由时差不在那六列里**：它们是 10300 / 10290 两个独立列，键 `total_time_difference` / `free_time_difference`，是六参数的**派生量** | 实测 |
| 6 | 六参数与「自由版最晚值」是**两套**：`latest_start_free_time` / `latest_stop_free_time` 是独立键，不与 `latest_start_time` 共用 | 实测 |
| 7 | 六参数列的消费点是**列集构造器** `FUN_1402f1490`，六条按固定顺序写进同一数组（视图归属待证） | 实测 |
| 8 | 正推/逆推/时差的计算次序：本报告给出**标准口径**并在 §5 标注哪几条只有键名佐证、哪几条已实测 | 混合 |

---

## 1 身份：六个时标是哪六列

列注册表 `FUN_1400d07f0`（列 ID → 字段键）与列定义派发 `FUN_1400d10f0`（列 ID → 类型/宽度/显示名）
两处交叉验证后，六列如下：

| 列 ID | 列名 | JSON 键 | 地址 | 数据类型 | 默认宽 |
|---|---|---|---|---|---|
| 10170 | 最早开始 | `earlist_start_time` | `0x1408d05c0` | 4 | 110 |
| 10180 | 最早完成 | `earlist_stop_time` | `0x1408d05d8` | 4 | 110 |
| 10190 | 最晚开始 | `latest_start_time` | `0x1408d05f0` | 4 | 110 |
| 10200 | 最晚完成 | `latest_stop_time` | `0x1408d0608` | 4 | 110 |
| 10210 | 最晚开始(自由) | `latest_start_free_time` | `0x1408d0458` | 4 | 140 |
| 10220 | 最晚完成(自由) | `latest_stop_free_time` | `0x1408d0470` | 4 | 140 |

**为什么能确定是这一组**（三条硬信号）：

1. **ID 连续**：10170/10180/10190/10200/10210/10220，步长 10，中间无其它列插入；
2. **数据类型一致**：全部 `type=4`；而其它时间列（计划开始/完成、实际开始/完成、要求开始/完成、预计完成）是 `type=8`（日期型）—— 说明这六列是**算出来的数值量**，不是存下来的日期；
3. **宽度成对**：前四列 110、后两列 140（"最晚开始(自由)"比"最晚开始"多两个字符，正好加宽）—— 是手工配的一组。

---

## 2 与另两组"6"的区别（避免再次混淆）

同一张表的日期/时间列共 119 列，其中会被人叫成"六个时间"的其实有三组，务必分清：

| 组 | 内容 | 列 ID | 类型 | 性质 |
|---|---|---|---|---|
| **A. 六个时标（本报告）** | 最早开始 / 最早完成 / 最晚开始 / 最晚完成 / 最晚开始(自由) / 最晚完成(自由) | 10170–10220 | 4 | CPM 算出的**时间量** |
| B. 计划·实际·预测 三对端点 | 计划开始/完成、实际开始/完成、预测开始/完成 | 10060/10070/10230/10240/10250/10260 | 8/8/8/8/4/4 | 横道图三条横道 = **六个端点** |
| C. 时标刻度（前两轮） | fov 1..6 / 分段表类型 日·一~六 | — | — | 横轴**怎么画** |

- **A** 是"计划算出了什么"；**B** 是"图上用哪几个时间画条"；**C** 是"横轴刻度怎么排"。
- A 与 B 有交集但不是一回事：B 里的 计划开始/完成 才是 CPM 的输入（`compute_start_time` / `compute_stop_time`），
  A 是在 B 的基础上往上算的结果。
- **总时差(10300) / 自由时差(10290) 属于 A 的派生列，不属于 A 那六列本身。**

---

## 3 存储与键池

六个键在 `.rdata` 同一段 UTF-8 键池里（`0x1408d0380..0x1408d0740`），周边是关键的工作时间字段：

```
0x1408d0390  plan_delay_days          计划滞后天数
0x1408d03a0  estimate_delay_days      预测滞后天数
0x1408d03b8  stop_delay_days          完成滞后天数
0x1408d0410  real_duration_days       实际工期(天)
0x1408d0428  real_start_diff_days     开始时间差异(天)
0x1408d0440  real_stop_diff_days      完成时间差异(天)
0x1408d0458  latest_start_free_time   最晚开始(自由)   ★ 六时标
0x1408d0470  latest_stop_free_time    最晚完成(自由)   ★ 六时标
0x1408d0488  plan_left_days           计划剩余时间
0x1408d0498  predict_left_days        预测剩余时间
0x1408d0590  free_time_difference     自由时差（派生）
0x1408d05a8  total_time_difference    总时差（派生）
0x1408d05c0  earlist_start_time       最早开始         ★ 六时标
0x1408d05d8  earlist_stop_time        最早完成         ★ 六时标
0x1408d05f0  latest_start_time        最晚开始         ★ 六时标
0x1408d0608  latest_stop_time         最晚完成         ★ 六时标
0x1408d0620  start_time_difference    开始时间差异
0x1408d0638  stop_time_difference     完成时间差异
0x1408d0650  relative_start_time      相对开始
0x1408d0668  relative_stop_time       相对完成
0x1408d0680  estimate_delay_time      预测延隔时间
0x1408d0698  plan_delay_time          计划延隔时间
0x1408d06a8  duration_difference      工期差异
0x1408d06c0  show_duration_difference 工期差异显示
0x1408d06e0  effect_on_key_path       影响关键线路
0x1408d06f8  last_plan_duration       原计划工期
0x1408d0710  start_time_change        开始时间变动
0x1408d0728  stop_time_change         完成时间变动
```

配套的计划开始/完成（CPM 输入）在工作键池另一段：

```
0x140872948  real_start_time     实际开始
0x140872968  real_stop_time      实际完成
0x140872988  plan_start_time     计划开始
0x1408729a8  plan_stop_time      计划完成
0x1408729c8  predict_stop_time   预测完成
0x140872908  ask_start_time      要求开始
0x140872928  ask_stop_time       要求完成
0x140872a20  relative_start_time 相对开始
0x140872a48  relative_stop_time  相对完成
0x140894448  compute_start_time  排程算出的开始
0x140894460  compute_stop_time   排程算出的完成
```

> **更正（第二轮实测）**：`estimate_start_time` 与 `estimate_stop_time` **都存在**，
> 但它们是**窄字符（ASCII）字面量**，不在 UTF-16 键池里 —— 第一轮用 UTF-16 搜所以漏了。
> 序列化器 `FUN_1402fba10` 里两行挨着：
> ```
> local_248 = "estimate_start_time"; len 0x13
> local_1e8 = "estimate_stop_time";  len 0x12
> ```
> 登记表把它们都写成 `estimate_start_time` 是 Ghidra 对同形字面量的显示合并，不是程序缺陷。
> `predict_stop_time`（0x1408729c8）是另一套命名（"预计完成" 列 10270）。

---

## 4 六列的消费点

六个键的 `lea` 引用各只有 1 处，全在 `FUN_1402f1490`（0x1402f1490，13881 B）：

```
0x1402f2862 → free_time_difference
0x1402f28e0 → total_time_difference
0x1402f29a7 → earlist_start_time
0x1402f2c45 → latest_start_time      （另有 0x1402f2862/0x1402f28e0 同函数）
```

该函数把键指针逐个写进一个列描述数组，**写入顺序固定**（反编译行号即写入序）：

```
行 481  latest_start_free_time    0x1408d0458
行 512  latest_stop_free_time     0x1408d0470
行 670  outline_number            0x1408d04b0
行 721  change_details            0x1408d0518
行 752  change_resolution         0x1408d0528
行 797  front_works               0x1408d0540
行 852  behind_works              0x1408d0580
行 915  earlist_start_time        0x1408d05c0   ← 六时标第一项
行 957  earlist_stop_time         0x1408d05d8
行 999  latest_start_time         0x1408d05f0
行 1041 latest_stop_time          0x1408d0608
行 1155 show_duration_difference  0x1408d06c0
```

→ **六参数（含自由版两条）在同一数组里成组、按"最晚(自由) ×2 → 最早 ×2 → 最晚 ×2"顺序写入**，
中间夹着 大纲数字 / 变动说明 / 解决措施 / 前置后置工作 / 工期差异显示。
这是一个**固定的列集**，不是各视图自由拼装 → 复刻时做成不可拆的列组即可。

（该数组挂在哪个视图上：键集含 `change_details` / `change_resolution` / `start_time_change`，
与「计划变动分析 / 进度对比」族的字段重合，**视图归属为推断**，未定位到视图注册点。）

---

## 5 六时标的运作关系

### 5.1 标准口径（CPM 双代号）

```
设 工作 i 的工期 D_i，紧前集合 P(i)，紧后集合 S(i)，项目计算终点 T

① 正推（最早时间）
   ES_i = max( EF_p + lag_p )   p ∈ P(i)   ；无紧前时 ES_i = 计划开始
   EF_i = ES_i + D_i

② 逆推（最晚时间）
   LF_i = min( LS_s − lag_s )   s ∈ S(i)   ；无紧后时 LF_i = 要求完成 或 T
   LS_i = LF_i − D_i

③ 时差
   TF_i = LS_i − ES_i = LF_i − EF_i          （总时差）
   FF_i = min( ES_s ) − EF_i                 （自由时差，无紧后时 = TF_i）

④ 关键工作
   TF_i == 0  ⇔  关键工作（列 10720 is_keywork / 10735 effect_on_key_path）

⑤ 自由时差版最晚值（ZPert 特有的一套）
   LS'_i = ES_i + FF_i
   LF'_i = LS'_i + D_i
   → 即"只让自身自由时差为 0 时"的最晚开始/完成，比 ③ 的 LS/LF 更早，
     用于「不影响紧后工作的前提下最多能拖到哪天」
```

### 5.2 哪几条有证据、哪几条是推断

| 环节 | 状态 |
|---|---|
| 六列存在、ID 连续、类型 4、宽度成对 | **实测**（列定义派发 `FUN_1400d10f0`） |
| 六个键名与地址 | **实测**（`.rdata` 直读） |
| 六列被 `FUN_1402f1490` 按固定顺序成组取用 | **实测**（`lea` 引用） |
| `compute_start_time` / `compute_stop_time` 是排程输出 | **实测**（键存在） |
| TF/FF 是派生列、与六列同级但不属于六列 | **实测**（10300/10290 独立列 + 独立键） |
| `latest_*_free_time` 与 `latest_*_time` 是两套 | **实测**（两套独立键） |
| **正推→逆推→时差 的求值次序与公式** | **推断**（教材口径 + 键名语义；计算函数本体尚未定位） |
| FF′ 版最晚值 = ES + FF | **推断**（由"自由"二字与独立键推得，未读计算代码） |

### 5.3 复刻口径（可直接落地）

```ts
interface WorkTimeParams {
  es: number;      // earlist_start_time   最早开始
  ef: number;      // earlist_stop_time    最早完成
  ls: number;      // latest_start_time    最晚开始
  lf: number;      // latest_stop_time     最晚完成
  lsf: number;     // latest_start_free_time  最晚开始(自由)
  lff: number;     // latest_stop_free_time   最晚完成(自由)
  tf: number;      // total_time_difference   总时差（派生）
  ff: number;      // free_time_difference    自由时差（派生）
}
// 计算顺序：正推 es/ef → 逆推 lf/ls → tf/ff → lsf/lff
// 单位：与 plan_start_time 等一致（内部为时间刻度值，非 Date）
// 类型码 4 = 计算值；不要按日期型(8)序列化，否则与原格式不兼容
```

---

### 5.4 工作对象内的字段偏移（第二轮实测，序列化器 `FUN_1402fba10` 读出）

| 偏移 | 类型 | 字段 | 序列化键 |
|---|---|---|---|
| +0x58 | int64 | 计算开始 | （加载器读 JSON `compute_start_time`） |
| +0x60 | int64 | 计算完成 | （加载器读 JSON `compute_stop_time`） |
| +0x79 | byte | **标志位**：bit0 = 关键工作；bit2 = 另一套"投标口径"关键工作 | `is_keywork`（由 `FUN_1402efed0` 取出后写入） |
| +0x80 | QString | 计划开始 | `plan_start_time` |
| +0x88 | QString | 计划完成 | `plan_stop_time` |
| +0x90 | QString | 预测开始 | `estimate_start_time` |
| +0x98 | QString | 预测完成 | `estimate_stop_time` |
| +0xf0 | QString | 原计划开始 | `last_plan_start_time` |
| +0xf8 | QString | 原计划完成 | `last_plan_stop_time` |
| +0x104 | int | 完成百分比 | `stop_percent` |
| +0x129 | byte | 仅要求时间告警 | `is_ask_time_only_waring` |

**两个关键结论**：

1. **计划/实际/预测时间是 QString，计算时间是 int64** —— 混合类型。复刻时不能统一按数字存，
   否则与原文件不兼容（`plan_start_time` 等键在 JSON 里是**日期文本**）。
2. **`is_keywork` 不是每次查表算 TF==0，而是 `work+0x79` 的一个 bit**：
   ```c
   byte FUN_1402efed0(longlong w, char alt) {
     if (alt && *(char*)(*(longlong*)(w+0x248) + 0x11b6)) return (*(byte*)(w+0x79) >> 2) & 1;
     return *(byte*)(w+0x79) & 1;
   }
   ```
   `alt` 分支由 `*(work+0x248)+0x11b6` 这个开关决定（投标/普通两种口径并存）。
   → **关键线路判定是"标记位"语义，不是"实时比较总时差"**；复刻时必须在 CPM 结束后
   统一刷这个 bit，查询侧只读位。

3. **`compute_start_time` / `compute_stop_time` 只被加载器读、不被工作序列化器写**：
   说明它们是 CPM 的**缓存输出**，由排程路径单独落盘；工作保存路径不动它们。

### 5.5 六参数的求值位置（收窄结果）

六个键（`earlist_*` / `latest_*`）在整个 exe 里**只有 1 处 `lea` 引用**，全部落在
`FUN_1402f1490`（13.8 KB）—— 该函数同时引用 `change_details` / `change_resolution` /
`effect_on_key_path` / `last_plan_duration` / `show_duration_difference`，
**是「计划变动分析 / 进度对比」的模型构造器**：

- 它按固定顺序把六参数 + 六项差异指标写成一批列描述；
- **六参数不进入工作 JSON**（加载器与序列化器都无这些键）；
- 因此六参数是**分析时现算的派生量**，与 `total_time_difference` / `free_time_difference` 同级。

> 这解释了为什么六列是 `type=4`（计算型）而不是 `type=8`（日期型）：
> 它们从来不是"存下来的日期"，而是**每次分析时按 CPM 口径重算的数值**。

### 5.6 附带的旁证：计划检查器

`FUN_140520550` 引用一组规则键 `total / key_path / logical_depends / depends / work_content /
match_target / fs_works_count / resource_cover / must_works_counts / long_duration_works_counts /
break_works / break_links`，输出 `normal_result` / `toubiao_result`（普通版/投标版）。
→ ZPert 有独立的「计划检查」规则引擎，**关键线路是被检对象之一**，
说明关键线路除了算，还要过一遍规则校验。
## 6 待证项

| # | 项 | 影响 |
|---|---|---|
| 1 | 正推/逆推的计算函数本体（未定位，但已收窄到"分析时现算"这条路） | 决定 lag/日历/跨假期 的精确处理；目前只有教材口径 |
| 2 | `latest_*_free_time` 的实际求值式 | 决定"自由版最晚值"是否等于 `ES + FF` |
| 3 | ~~六参数的存储形态~~ → **已收口：不落工作 JSON，分析时现算** | 加载器/序列化器均无这些键，只在 `FUN_1402f1490` 出现 |
| 4 | 关键工作判定用的是 `TF==0` 还是 `is_keywork` 列缓存 | 影响 关键线路 复刻一致性 |
| 5 | 预测开始/完成的 `type=4` 语义（非日期） | 与六参数同型，疑似同为派生数值 |

---

## 7 产物

| 位置 | 内容 |
|---|---|
| `re-zpert/evidence/decomp_time6.c` | 列注册 `FUN_1400d07f0`（ID→键）+ 引用 10060 的列构造 |
| `re-zpert/evidence/decomp_timedic.c` | 列定义派发 `FUN_1400d10f0`（119 列的 类型/宽度/显示名/键） |
| `re-zpert/evidence/decomp_cpm7.c` | 六参数列组取用 `FUN_1402f1490` |
| `re-zpert/evidence/time_params.txt` | 键池地址表 + 列 ID 表（grep 版） |
