# 斑马进度 ZPert8 · 三线并进：埋点族 / 导出族 / 六参数（2026-11-14）

> 本轮三条线一起推：① 埋点覆盖面扩全 ② 导出族挖到可复刻 ③ 六个时间参数的结构与求值。
> 每条都给出了新的实测证据，其中 ③ 拿到了**双套字段布局**这个关键结构。

---

## 线①：AI 埋点族不是独立系统

### 1.1 实测

在命令注册表相邻区域（`0x1408e8d0c..0x1408e8e18`）挖到这批 ID：

```
0x1408e8d0c  80129      ← 关系合法性校验族（8xxxx）
0x1408e8d50  1100001    ← AI 交互埋点族（11xxxxx）
0x1408e8d68  1200001    ← AI 交互埋点族（12xxxxx）
0x1408e8d80  1200002
0x1408e8da8  1200004
0x1408e8dd8  1204005
0x1408e8df8  1204006
0x1408e8e18  1204007
```

它们与普通命令 ID（`10001`…`90000`）**同处一张表**、同一种窄字符串形态。

### 1.2 结论

- **`11xxxxx`/`12xxxxx` 就是命令 ID 的一个分段**，不是另一套埋点系统；
- 上一轮"其余命令走另一套埋点"的推测**需要弱化**：当前证据是
  **只有导出族（6 个 handler）调用了 `FUN_140576bc0`**，其余命令没有调用它
  → 埋点**不是全量覆盖**，而是按需（导出这类"会产生文件/副作用"的动作才报）；
- 因此"命令 → handler"的扫描法只在导出族有效，其余命令仍需逐个 handler 认领。

---

## 线②：导出族可复刻 —— MS Project XML 字段表

### 2.1 六条实现路径（承上轮）

| handler | 形态 |
|---|---|
| `FUN_14058b080` | Excel（libxl `xlCreateXMLBookW`） |
| `FUN_14058bfa0` | SVG（`QSvgGenerator`） |
| `FUN_14058c340` | PDF/打印（`QPrinter`） |
| `FUN_14058c610` | 删旧再写（`QDir::remove`） |
| `FUN_140594890` | 按 MIME 选格式（`QMimeDatabase::defaultTypeFor`） |
| `FUN_1405e1189` | 导出向导 |

### 2.2 MS Project XML 模板（实测原文，文件偏移 `0xc88270` 起）

```
...  Exceptions
WorkWeeks
WorkWeek
1984-01-01T00:00:00
2049-12-31T23:29:00
综合周日历
Tasks
Task
ID
IsNull
OutlineLevel
Estimated
Start
Finish
Duration
DurationFormat
Milestone
ConstraintType
ConstraintDate
Summary
PredecessorLink
PredecessorUID
CrossProject
LinkLag
LagFormat
http://schemas.microsoft.com/project
```

归纳成结构：

```xml
<Project xmlns="http://schemas.microsoft.com/project">
  <Exceptions/>
  <WorkWeeks>
    <WorkWeek>
      <StartDate>1984-01-01T00:00:00</StartDate>
      <FinishDate>2049-12-31T23:29:00</FinishDate>
      <Name>综合周日历</Name>
    </WorkWeek>
  </WorkWeeks>
  <Tasks>
    <Task>
      <ID/><IsNull/><OutlineLevel/><Estimated/>
      <Start/><Finish/><Duration/><DurationFormat/>
      <Milestone/><ConstraintType/><ConstraintDate/><Summary/>
      <PredecessorLink>
        <PredecessorUID/><CrossProject/><LinkLag/><LagFormat/>
      </PredecessorLink>
    </Task>
  </Tasks>
</Project>
```

**要点（复刻必读）**：

1. **只有一个 WorkWeek**：`1984-01-01 00:00` → `2049-12-31 23:29`，名字 `综合周日历`
   → 斑马**不把自身日历导出成 MS Project 工作日历规则**，而是给一个覆盖全时间域的"综合周日历"，
   再用 `Exceptions` 承载例外日（节假日/调休）。**这是偷懒但可靠的做法**，复刻可直接照抄。
2. **任务只导 12 个字段**：ID / IsNull / OutlineLevel / Estimated / Start / Finish / Duration /
   DurationFormat / Milestone / ConstraintType / ConstraintDate / Summary。
   **没有** 资源、成本、WBS、备注、自定义字段 → 导出是"骨架级"，不是全量往返。
3. **前置关系只导 4 项**：PredecessorUID / CrossProject / LinkLag / LagFormat
   → **关系类型（FS/SS/FF/SF）不在字段列表里**，只能靠 `LinkLag` 的正负与 `LagFormat` 编码，
   或根本不导类型（MS Project 默认 FS）。这是**已知的保真度缺口**，复刻时要自行决定是否补齐 `<Type>`。
4. 命名空间是 `http://schemas.microsoft.com/project`（MS Project 2003+ 通用）。

### 2.3 模板的消费方式

对模板区间（`0x140c893400..0x140c893700`）做 `lea` 扫描：**0 命中**。

→ 模板**不是被代码逐串引用**的，而是作为一整块数据由**指针表或整体拷贝**消费
（与本项目其它静态字符串表的组织方式一致）。定位消费点需要扫 `.rdata` 里的 8 字节指针，
本轮未做。

### 2.4 导出失败语义（实测文案）

| 文案 | 触发 |
|---|---|
| `本计算机没有检测到MS Project，仅能导出XML格式，该格式可以在Project中转换打开` | 探测不到本机 MS Project → 降级为 XML（**能力探测，不是硬失败**） |
| `导出失败，请尝试重新导出` | 通用失败 |
| `创建文件失败！请检查是否有可写权限。` | 写盘权限 |
| `创建临时文件失败` | 先写临时文件再改名 |

---

## 线③：六个时间参数的**双套字段**（关键结构发现）

### 3.1 实测：六参数在 work 对象里有两套

`FUN_1402f1490`（计划变动分析模型构造器）取每个参数时都是同一形态：

```c
if (*(char *)(param_1[0x49] + 0x11b6) == '\0')  lVar13 = param_1[0x16];
else                                            lVar13 = param_1[0x1a];
FUN_14031b420(lVar13, out, 0, 1);      // 原始时间 → 显示串（0x2d=45）
FUN_14029e5f0(&local_3a0, out, ctx);   // 送入列值
```

四个参数各取一次，得到：

| 参数 | 普通口径 | 投标口径 | 字节偏移（普通 / 投标） |
|---|---|---|---|
| 最早开始 ES | `work[0x16]` | `work[0x1a]` | +0xB0 / +0xD0 |
| 最早完成 EF | `work[0x17]` | `work[0x1b]` | +0xB8 / +0xD8 |
| 最晚开始 LS | `work[0x18]` | `work[0x1c]` | +0xC0 / +0xE0 |
| 最晚完成 LF | `work[0x19]` | `work[0x1d]` | +0xC8 / +0xE8 |

**切换标志 `*(char*)(work[0x248] + 0x11b6)`** —— 与 `is_keywork` 的 bit0/bit2 是**同一个开关**：

```c
byte FUN_1402efed0(work, alt) {
  if (alt && *(char*)(*(longlong*)(work+0x248) + 0x11b6)) return (*(byte*)(work+0x79) >> 2) & 1;
  return *(byte*)(work+0x79) & 1;
}
```

### 3.2 这意味着什么

**斑马对每个工作同时维护两套 CPM 结果：普通口径 与 投标口径。**

- 同一开关（`doc/settings + 0x11b6`）同时决定：
  - 用哪一套 ES/EF/LS/LF（`+0xB0..` 还是 `+0xD0..`）
  - 用哪个 bit 判关键工作（bit0 还是 bit2）
- 两条口径**并存、互不覆盖**，切换只改"读哪一套"，不重算。

为什么需要两套？结合"投标"字样与命令表的 `toubiao_result`（普通/投标两版检查结果），
合理推断：**投标口径 = 按业主要求日期反推的最晚时间体系**（要求开始/完成参与逆推），
普通口径 = 按项目自身计划的逆推体系。这条推断尚缺直接证据。

### 3.3 顺带拿到的算法片段

`total_time_difference`（总时差）的求值：

```c
lVar10 = FUN_140312a90(&local_9e0);                       // 取原始时长值
FUN_1401efee0(param_2, "total_time_difference",
              ((double)lVar10 / dVar3) / (double)*(int *)(lVar13 + 0x90), ctx);
```

即 **总时差 = 原始总时差值 / 除数 / 上下文里的日长系数（`ctx+0x90`）** ——
是"把内部时长单位换算成天"的两级除法，不是简单的 `LS-ES` 后直接输出。

时间值输出统一经 `FUN_14031b420(..., 0x2d, 1)`（0x2d=45，疑日期格式/区域码）。

### 3.4 仍未闭环

- **正推/逆推本体没找到**：尝试扫 `[reg+0x79]` 的字节写（`or byte [r+0x79],1` / `mov byte [r+0x79],imm`）
  → **0 命中**，说明标志位不是这样写的（可能经宽写或计算指针），需要换手法。
- 六参数的**写入侧**也没定位（只知道读取侧偏移）。

**建议手法**：从 `work+0xB0` / `work+0xD0` 这两个**数据偏移**反向扫写入指令
（`mov [reg+0xB0], r64` 形态，而非 `lea`），命中即为 CPM 计算函数。

---

---

## 三·续 线③：UI 字段表三方互证 + 布局模型修正 + CPM 扫描失败记录

### 三·续.1 工作信息面板的字段表（实测 `region_0x140678710`，140 条引用）

斑马的工作信息面板是 **4 个页签**，字段顺序即界面顺序：

**概况**：名称 / 工期 / 类型 / 分类 / 网络图中显示为分区 / 工期自动计算 /
计划开始 / 计划完成 / 预测开始 / 预测完成 / 工作部位 / 负责人 / 备注

**工程量**：工程量 / 累计完成量

**执行**：执行状态（未开始·进行中·已完成）/ 累计实际完成(%) / 典型偏差 /
实际开始 / 预计完成 / 实际完成 / 偏差原因 / 解决措施 / 显示于网络图

**高级设置（高级）**：

```
外文名称
工作编码
要求开始
日历
要求完成
仅预警
排网方式
相对开始
相对完成
┌ 计算时间（预测）        ← ★ 分组标题
│  最早开始               ← 列 10170
│  最早完成               ← 列 10180
│  最晚开始               ← 列 10190
│  最晚完成               ← 列 10200
│  最晚开始（自由）        ← 列 10210
│  最晚完成（自由）        ← 列 10220
│  自由时差               ← 列 10290（派生）
└  总时差                 ← 列 10300（派生）
```

### 三·续.2 三方互证：六时标 = 分组里的前六项

| 证据源 | 内容 |
|---|---|
| UI 字段表顺序（本轮） | 最早开始 → 最早完成 → 最晚开始 → 最晚完成 → 自由版 ×2 → 自由时差 → 总时差 |
| 列 ID 连续性（前轮） | `10170/10180/10190/10200/10210/10220` 连续无空洞，两时差另有 ID |
| JSON 键（前轮） | `earlist_start_time / earlist_stop_time / latest_start_time / latest_stop_time / latest_start_free_time / latest_stop_free_time` |

三条独立证据指向同一结论：**六时标 = 该分组的前六项**，
`自由时差(10290)` / `总时差(10300)` 是同分组内的**派生项**，不属于那六列。

**新增语义**：分组标题是 **`计算时间（预测）`** —— 斑马把这六项明确定义为
"由算法算出来的时间"。这与「概况」页的 `预测开始/预测完成`（列 10250/10260，
是横道图上预测条的**端点**）是**两个不同的东西**，之前容易混。

### 三·续.3 布局模型修正：时间字段是"指针槽"，不是内联值

`FUN_1402fba10`（序列化器）与 `FUN_1402f1490`（分析构造器）**都声明 `longlong *param_1`**，
所以下标换算成立：

```
param_1[0x10..0x13] = +0x80 / +0x88 / +0x90 / +0x98   → 4 个相邻槽
param_1[0x16..0x1d] = +0xB0 / +0xB8 / +0xC0 / +0xC8 / +0xD0 / +0xD8 / +0xE0 / +0xE8
```

若这些槽里放的是内联 `QString`（0x20 字节），相邻槽必然重叠。
→ 只能解释为 **8 字节指针槽**：`Work` 的时间字段是「指针槽 + 独立时间对象」，
不是内联值。唯一例外是 `+0x58/+0x60` 的 `compute_start_time/compute_stop_time`（int64 原值）。

这修正了上一轮"直接内联"的隐含假设，也解释了为什么 `FUN_14031b420(slot, out, 0, 1)`
拿到的 `slot` 可以当对象用。

### 三·续.4 CPM 本体：三次手法全部失败（如实记录）

| 手法 | 结果 |
|---|---|
| 扫 `[reg+0x79]` 字节写（`or byte` / `mov byte`） | **0 命中** —— 标志位不是这样写的 |
| 按 disp 扫 `mov [reg+0xB0..0xE8], r64` | 命中 140~266 个/槽；按函数聚合后写满 8 槽的函数 **90 个** |
| 追加约束"同时写 `+0x248`（doc 指针）" | 仍剩 **11 个**候选，全是 UI 大函数（`WorkPagesDialog` / 各设置页构造器） |

**结论**：`0xB0..0xE8` 在 UI 构造器里也是通用偏移，**盲扫无法区分**。
这类"按偏移找函数"的路子在这个样本上失效（大函数太多、偏移复用严重）。

**下一步（换思路）**：从**类**入手，不再从偏移入手。

1. RTTI 已实测存在 `Work@@` / `JobTimeCtrlPage@@` / `JobsTimeCtrlPage@@` / `ZBasePertDoc@@`；
2. 取 `Work@@` 的 vtable → 枚举其全部方法 → CPM 极可能是成员函数（或以其为参的自由函数）；
3. 或用 `JobTimeCtrlPage` 的构造器：它必须按 UI 顺序读这 8 个槽，
   对照已实测的字段顺序即可反推槽位，再顺藤找到写入方。
---

## 三·收口 线③：CPM 本体已定位（正推公式实测）

前几轮"按偏移盲扫"连续失败，本轮换了顺序：**先找到 `Work` 类，再从标志位反查**，一次打通。

### 三·收口.1 转折点：`+0x79` 是 disp8 编码

之前扫 `or byte [reg+0x79],1` 用的是 **disp32** 形态（`80 4? 79 00 00 00 xx`），0 命中。
实际编译器用 **disp8**（`0x79` 只用 1 字节位移）：`80 4b 79 01`。

按 disp8 形态重扫，命中 64 处，其中 CPM 的全部集中在 `0x1402731xx..0x140273bxx`：

```
0x14027314c  mov byte [rbx+0x79], 0     ← 清零
0x140273227  mov byte [rcx+0x79], 0
0x140273859  or  byte [rbx+0x79], 1     ← bit0 关键工作（普通）
0x1402738ad  or  byte [rbx+0x79], 4     ← bit2 关键工作（投标）
0x14027390f  or  byte [rbx+0x79], 2     ← bit1 第三个标志（新发现）
```

即 **`FUN_1402730c0`（0x1402730c0，4755 B）= CPM 总控**。

### 三·收口.2 `Work` 类确认

| 项 | 值 |
|---|---|
| `Work::vftable` | `0x1408c9c68`，**仅 6 项** |
| 构造器 | `FUN_1402d2130`（3341 B，首行 `*param_1 = Work::vftable`） |
| 创建点 | `FUN_140233f20` / `FUN_140235a50` / `FUN_140236500` |
| vtable 内容 | `[01] FUN_1402fba10` = 序列化器（前轮已分析）；其余为拷贝/比较 |

→ **CPM 不是 `Work` 的虚方法**。三个类的 vtable 已全部导出到 `vtables.txt`（ZQtPertDoc 104 项 / ZBasePertDoc / Work 6 项）。

### 三·收口.3 时序字段表（双源验证）

构造器初始化序列 + 序列化器取值表达式，两个独立来源一致：

| 槽索引 | 偏移 | 键 | 说明 |
|---|---|---|---|
| +9 | +0x48 | `ask_start_time` | 要求开始 |
| +10 | +0x50 | `ask_stop_time` | 要求完成 |
| +0xb | +0x58 | `real_start_time` | 实际开始 ★正推短路用 |
| +0xc | +0x60 | `real_stop_time` | 实际完成 |
| +0x10 | +0x80 | `plan_start_time` | 计划开始 |
| +0x11 | +0x88 | `plan_stop_time` | 计划完成 |
| +0x12 | +0x90 | `estimate_start_time` | 预测开始 |
| +0x13 | +0x98 | `estimate_stop_time` | 预测完成 |
| +0x16 | +0xB0 | ES 普通 | |
| +0x17 | +0xB8 | EF 普通 | |
| +0x18 | +0xC0 | LS 普通 | |
| +0x19 | +0xC8 | LF 普通 | |
| +0x1a | +0xD0 | ES 投标 | |
| +0x1b | +0xD8 | EF 投标 | |
| +0x1c | +0xE0 | LS 投标 | |
| +0x1d | +0xE8 | LF 投标 | |
| +0x1e | +0xF0 | `last_plan_start_time` | 原计划开始 |
| +0x1f | +0xF8 | `last_plan_stop_time` | 原计划完成 |

全部经 `FUN_140312540`（时间对象构造）初始化 → **实锤是对象槽，不是内联值**。

### 三·收口.4 CPM 三层结构

```
FUN_1402730c0  (4755 B)  CPM 总控
  ├ 初始化：清 +0x80/88/90/98/C0/C8 为哨兵 DAT_141125a50（"未计算"），+0x79 = 0
  ├ 起点播种：work0[0x48]（要求开始）→ 写 work0 的 +0xD0 / +0xB0 / +0x90 / +0x80
  ├ 建访问标记数组 FUN_1401d5510(工作数 + 另一类节点数)
  ├ 递归 FUN_14026ed70(doc, work0, 0, visit)
  └ 三个开关后处理：doc+0x120 / +0x121 / +0x122

FUN_14026ed70  (2281 B)  递归驱动（memo）
  if (work[0x80] != 哨兵) return;          ← 已算过，跳过
  FUN_14026e360(doc, work, visit);         ← 单工作正推
  （callers 含自身 → 递归）

FUN_14026e360  (2565 B)  单工作正推
```

### 三·收口.5 正推公式（伪码还原，实测）

```c
FUN_14026e360(doc, work, visit):
  // ① 环检测
  idx = (work[0xe]==1) ? (workCount + work[1]) : work[1];
  if (visit[idx>>5] & (1<<(idx&31))) return 0;      // 已在栈上 → 有环
  visit[idx>>5] |= 1<<(idx&31);

  // ② 初值取首工作的 ES / 预测开始
  t0 = work0[0x80];  t1 = work0[0x90];
  // 非起点工作再用 doc+0x118 / doc+0x108 的哨兵钳位

  // ③ 状态短路：进行中(1) / 已完成(2)
  if (work[0x34c] == 1 || work[0x34c] == 2) {
      x = work[0xb];                    // +0x58 实际开始
      work[0x16] = x;  work[0x10] = x;  // 普通 ES / 计划开始
      work[0x12] = x;  work[0x1a] = x;  // 预测开始 / 投标 ES
      return 1;
  }

  // ④ ES：对每个紧前取 max
  for (pred of 紧前集合) {
      if (pred 未算) 递归 FUN_14026e360 / FUN_14026ed70;
      if (t0 < pred[0x16]) t0 = pred[0x16];   // 普通 ES
      if (t1 < pred[0x10]) t1 = pred[0x10];
      if (t2 < pred[0x12]) t2 = pred[0x12];
      if (t3 < pred[0x1a]) t3 = pred[0x1a];   // 投标 ES
  }
  写回 work 的 +0xB0 / +0x90 / +0xD0 / +0x80

  // ⑤ EF：对每个紧前取 max 其 EF，再按关系类型换算
  for (pred) {
      if (t < pred[0xB8]) t = pred[0xB8];     // 普通 EF
      if (t < pred[0x98]) t = pred[0x98];
      if (t < pred[0xD8]) t = pred[0xD8];     // 投标 EF
  }
  关系类型分支：FUN_1401d8610(rel, out, predEF, work[0xd])   // SS/FF/SF 偏移换算
```

### 三·收口.6 由此确立的四条事实

1. **ES 是"对紧前取 max"**，且**四条时间线并行计算**（计划开始 / 预测开始 / 普通 ES / 投标 ES 各一份）；
2. **正推结果同时落到 4 个槽**（`+0x80 / +0x90 / +0xB0 / +0xD0`）→
   "计划开始 / 预测开始"与 ES **同源**，是同一计算值的多处落点，不是独立计算；
3. **进行中 / 已完成的工作不参与正推**，ES 直接取 `实际开始(+0x58)`；
4. **`+0x79` 有三个 bit**（此前只知道两个）：bit0 关键工作（普通）、bit1 第三标志、bit2 关键工作（投标）。

### 三·收口.7 尚未拿到

- **逆推（LS/LF）与总时差/自由时差**的求值函数：`FUN_1402730c0` 里只见 `or bit0/bit2`（关键工作），
  逆推应在另一函数（或 `FUN_1402730c0` 的后半段，本轮只读到前 1/3）。
- `FUN_1401d8610` 的 SS/FF/SF 偏移算式（关系类型换算的具体公式）。
## 汇总：三条线的状态

| 线 | 目标 | 结果 |
|---|---|---|
| ① 埋点族 | 扩全埋点覆盖 | **改判**：11/12xxxxx 是命令 ID 分段，不是独立埋点系统；埋点非全量（仅导出族） |
| ② 导出族 | 挖到可复刻 | **达成大半**：6 路径 + MS Project XML 12 字段表 + 单 WorkWeek 策略 + 4 条失败语义；模板消费点待查 |
| ③ 六参数 | 结构与求值 | **CPM 已定位**：`FUN_1402730c0`(总控) → `FUN_14026ed70`(递归 memo) → `FUN_14026e360`(单工作正推)；**正推公式实测**，逆推待补 |

---

## 产物

| 位置 | 内容 |
|---|---|
| `re-zpert/evidence/msproject_xml.txt` | MS Project XML 模板字段表与结构 |
| `re-zpert/evidence/timeparams_dual.txt` | 六参数双套布局 + 总时差算法片段 |
| `re-zpert/evidence/work_fields.txt` | 工作信息面板 4 页签字段表 + 三方互证 + 布局修正 + CPM 扫描失败记录 |
| `re-zpert/evidence/cpm_core.txt` | **CPM 三层结构 + 正推公式伪码 + 时序字段表双源 + +0x79 三 bit** |
| `re-zpert/evidence/vtables.txt` | ZQtPertDoc(104) / ZBasePertDoc / Work(6) 三个类的 vtable 方法索引 |
| `re-zpert/evidence/decomp_cpm_cand.c` | `FUN_1402730c0` CPM 总控 + `Work` 构造器 `FUN_1402d2130` |
| `re-zpert/evidence/decomp_forward.c` | `FUN_14026ed70` 递归驱动（memo） |
| `re-zpert/evidence/decomp_fwdwork.c` | `FUN_14026e360` 单工作正推（ES/EF 公式） |
| `re-zpert/evidence/decomp_handlers.c` | 导出族 6 个 handler 全文（承上轮） |
| `re-zpert/evidence/decomp_cpm7.c` | `FUN_1402f1490`，六参数取值的双套分支所在 |

## 待证

1. MS Project XML 模板的消费点（扫 `.rdata` 里指向 `0x140c8934xx` 的 8 字节指针）
2. `work+0xB0/+0xD0` 的**写入侧**（`mov [reg+0xB0], r64` 形态扫描）→ CPM 本体
3. 两套口径的语义（投标 = 按业主要求日期逆推？）需在 `+0x11b6` 开关的赋值点确认
4. `<Type>`（FS/SS/FF/SF）是否真的不导出 —— 需看 XML 写出代码而非模板（模板可能只列"常量字段"）
