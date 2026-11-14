# 斑马进度 ZPert8 · CPM 计算内核完整逆向（六个时标的求值链）· 2026-11-14

> **本文是「六个时标」专题的收口**：正推 → 逆推 → 关键判定的完整求值链，
> 全部为反编译实测，不再是教材口径推断。
>
> **样本**：`C:\Program Files\ZebraTech\ZPert8\ZPert.exe`（7.0.706.0）
> SHA256 `C61972FAA6A1A08AB846037564CAAD7B2AB23ABCEE3A11B27885DBD24E094E68`

---

## 0 一页结论

| # | 结论 | 强度 |
|---|---|---|
| 1 | **CPM 总控 `FUN_1402730c0`**（4755 B）：初始化 → 起点播种 → 正推 → 逆推 → 关键判定 → 两个开关后处理 | 实测 |
| 2 | **正推 `FUN_14026ed70`（memo）+ `FUN_14026e360`（单工作）**：ES 对紧前取 **max** | 实测 |
| 3 | **逆推 `FUN_140270660`（自身递归）+ `FUN_14026fbd0`**：LF 对紧后取 **min** | 实测 |
| 4 | **时间运算只有两个原语**：`FUN_1401d8610(cal,out,基准,工期)` = 日历感知的**加减工期**；`FUN_1401d9940(cal,out,a,b,0)` = **求时间差** | 实测 |
| 5 | **`Work+0x68` = 工期（duration）** —— 三处用法互证（计划开始=计划完成−工期、预测开始=预测完成−工期、LS=LF−工期） | 实测 |
| 6 | **关键判定是三个 bit，条件各不相同**（见 §4）：<br>bit0 ⟸ 计划完成 == LF普通；bit2 ⟸ 预测完成 == LF投标；bit1 ⟸ 预测完成 ≤ LF普通 | 实测 |
| 7 | **四条时间线并行**：计划（+0x80/0x88）、预测（+0x90/0x98）、普通 CPM（+0xB0..C8）、投标 CPM（+0xD0..E8） | 实测 |
| 8 | 环检测：位图 + `idx = workCount + work[1]`（区分工作与另一类节点） | 实测 |

---

## 1 总控流程（`FUN_1402730c0`）

```c
FUN_1402730c0(doc):
  /* ① 复位 */
  for (work in works) {
      work[0x80..0x98] = 哨兵 DAT_141125a50;     // 计划/预测开始完成
      work[0xC0..0xC8] = 哨兵;                    // 普通 LS/LF
      work[0x79] = 0;                             // 三个关键标志清零
      ... (未开始/进行中分支里还会做 FUN_1402f02b0 初始化)
  }
  for (node in 另一类节点) { 清 0x80..0x98 / 0xC0..0xC8 / 0x79 }

  /* ② 起点播种 */
  work0 = works[0];
  t = work0[0x48];                                // 要求开始 ask_start_time
  work0[0xD0] = t;  work0[0xB0] = t;              // 投标 ES / 普通 ES
  work0[0x90] = t;  work0[0x80] = t;              // 预测开始 / 计划开始

  /* ③ 建访问位图 */
  FUN_1401d5510(&visit, workCount + nodeCount, 0);

  /* ④ 正推 */
  FUN_14026ed70(doc, work0, 0, &visit);

  /* ⑤ 逆推 */
  ... FUN_140270660(doc, work, 0, &visit) ...

  /* ⑥ 关键判定（见 §4）*/

  /* ⑦ 三个开关后处理 */
  if (doc+0x122) { 起点系工作的 +0x48 复位 }
  if (doc+0x120) { ... }
  if (doc+0x121) { ... }
```

---

## 2 正推：ES / EF

### 2.1 递归驱动 `FUN_14026ed70`（memo）

```c
FUN_14026ed70(doc, work, flag, visit):
  if (work[0x80] != 哨兵) return 1;        // 已算过 → memo 命中
  return FUN_14026e360(doc, work, visit);  // 单工作正推
```

### 2.2 单工作正推 `FUN_14026e360`

```c
FUN_14026e360(doc, work, visit):
  /* 环检测 */
  idx = (work[0xe] == 1) ? (workCount + work[1]) : work[1];
  if (visit[idx>>5] & (1<<(idx&31))) return 0;      // 已在递归栈上 → 有环
  visit[idx>>5] |= 1<<(idx&31);

  /* ① 状态短路 */
  if (work[0x34c] == 1 || work[0x34c] == 2) {        // 进行中 / 已完成
      x = work[0x58];                                // 实际开始
      work[0x16] = x;  work[0x10] = x;               // 普通 ES / 计划开始
      work[0x12] = x;  work[0x1a] = x;               // 预测开始 / 投标 ES
      return 1;
  }

  /* ② 初值 = 首工作的 ES / 预测开始 */
  t0 = work0[0x80];  t1 = work0[0x90];
  /* 非起点工作再用 doc+0x118 / doc+0x108 的哨兵钳位 */

  /* ③ ES：对每个紧前取 max（四条线并行） */
  for (pred in 紧前集合) {
      if (pred 未算) 递归（普通节点走 FUN_14026e360，起点走 FUN_14026ed70）;
      if (t0 < pred[0x16]) t0 = pred[0x16];   // 普通 ES
      if (t1 < pred[0x10]) t1 = pred[0x10];   // 计划开始
      if (t2 < pred[0x12]) t2 = pred[0x12];   // 预测开始
      if (t3 < pred[0x1a]) t3 = pred[0x1a];   // 投标 ES
  }
  work[0xB0] = t0;  work[0x80] = t1;  work[0x90] = t2;  work[0xD0] = t3;

  /* ④ EF：对紧前取 max 其 EF，再用关系换算 */
  for (pred) {
      if (t < pred[0xB8]) t = pred[0xB8];     // 普通 EF
      if (t < pred[0x98]) t = pred[0x98];     // 预测完成
      if (t < pred[0xD8]) t = pred[0xD8];     // 投标 EF
  }
  关系换算：FUN_1401d8610(cal, out, predEF, work[0x68])    // EF_pred + 工期
           分 SS/FF/SF 时用另一参数（关系类型在 pred 的关系对象里）
```

---

## 3 逆推：LS / LF

### 3.1 逆推核心 `FUN_140270660`（自身递归）

```c
FUN_140270660(doc, work, ?, visit):
  if (work[0xC8] == 哨兵 && FUN_14026fbd0(...) == 0) return 0;   // memo

  if (work 是终点 / 无紧后) {
      if (work[0x34c] == 0) {                       // 未开始
          work[0xC0] = FUN_1401d8610(cal, out, work[0xC8], work[0x68]);  // LS = LF − 工期
          work[0xE0] = FUN_1401d8610(cal, out, work[0xE8], work[0x68]);  // 投标 LS
          /* 若 0x58 有效，则 0xC0/0xE0 直接用 0x58 */
      } else {                                       // 进行中 / 已完成
          work[0xC0] = work[0x58];                   // LS = 实际开始
          work[0xE0] = work[0x58];
      }
  } else {
      /* 对每个紧后取 min */
      for (succ in 紧后集合) {
          if (succ[0xC0] == 哨兵) 递归 FUN_140270660;
          if (t0 > succ[0xC8]) t0 = succ[0xC8];      // min LF 普通
          if (t1 > succ[0x88]) t1 = succ[0x88];      // min 计划完成
          ...
      }
  }
```

> 取 min 的比较用的是 `FUN_1403125d0`（`a > b` 判定），与正推取 max 是同一原语的两种用法。

### 3.2 逆推前的准备 `FUN_14026fbd0`

在逆推主循环前对每个节点预置 LF（终点赋 `DAT_141125a50` 之外的默认），细节未展开。

---

## 4 关键判定：三个 bit 的确切条件

`FUN_1402730c0` 逆推后的判定块（实测两次出现，普通节点与另一类节点各一遍）：

```c
/* ① bit0 —— 关键工作（普通口径） */
d = FUN_1401d9940(cal, &out, work[0x88], work[0xC8], 0);   // 计划完成 vs 普通 LF
if (FUN_140312a90(d) == 0) {
    work[0x79] |= 1;                                        // bit0 = 1
    work[0x30]->+0x50 |= 1;                                 // 节点上也打标记
    若 work[0x28] 无前置 → work[0x28]->+0x50 |= 1;
}

/* ② bit2 —— 关键工作（投标口径） */
d = FUN_1401d9940(cal, &out, work[0x98], work[0xE8], 0);   // 预测完成 vs 投标 LF
if (FUN_140312a90(d) == 0) {
    work[0x79] |= 4;                                        // bit2 = 1
    ...
}

/* ③ bit1 —— 预测不超期 */
if (状态 != 2 /* 非已完成 */) {
    d = FUN_1401d9940(cal, &out, work[0x98], work[0xC8], 0);   // 预测完成 vs 普通 LF
    if (FUN_140312a90(d) < 1) {
        work[0x79] |= 2;                                    // bit1 = 1
        work[0x28]->+0x50 |= 2;  work[0x30]->+0x50 |= 2;
    }
}
```

**语义归纳**：

| bit | 值 | 条件 | 含义 |
|---|---|---|---|
| bit0 | 1 | 计划完成 == LF（普通） | **关键工作**（总时差 = 0） |
| bit2 | 4 | 预测完成 == LF（投标） | **投标口径关键工作** |
| bit1 | 2 | 预测完成 ≤ LF（普通） | **预测不超期** |

> 三条都是"完成时刻与最晚完成时刻的比较"，不是先算总时差再判 0 ——
> **斑马用"完成 == LF"直接判关键**，等价于 TF == 0 但省一次减法。

---

## 5 时间运算原语

| 函数 | 签名 | 作用 |
|---|---|---|
| `FUN_1401d8610` | `(cal, out, 基准时刻, 工期)` | **日历感知的加减工期**：正推 EF = ES + 工期；逆推 LS = LF − 工期；计划开始 = 计划完成 − 工期 |
| `FUN_1401d9940` | `(cal, out, a, b, 0)` | **求时间差**（b − a），结果交给 `FUN_140312a90` 取整 |
| `FUN_1403125d0` | `(a, b)` | 比较（`a > b`？）—— 正推取 max / 逆推取 min 共用 |
| `FUN_140312590` | `(slot, 哨兵)` | 判"是否已计算"（与哨兵比较） |
| `FUN_140312a90` | `(时间对象)` | 取整数值（工期/天数的规范化） |

**`Work+0x68` = 工期（duration）**，三处用法互证：
1. `计划开始(+0x80) = 计划完成(+0x88) − 工期(+0x68)`
2. `预测开始(+0x90) = 预测完成(+0x98) − 工期(+0x68)`
3. `LS(+0xC0) = LF(+0xC8) − 工期(+0x68)`

---

## 6 四条时间线（并行的四套开始/完成）

| 时间线 | 开始 | 完成 | 来源 |
|---|---|---|---|
| 计划 | +0x80 | +0x88 | 用户输入 / 正推回填 |
| 预测 | +0x90 | +0x98 | 正推回填 |
| CPM 普通 | +0xB0 (ES) | +0xB8 (EF) / +0xC0 (LS) / +0xC8 (LF) | 正推/逆推 |
| CPM 投标 | +0xD0 (ES) | +0xD8 (EF) / +0xE0 (LS) / +0xE8 (LF) | 正推/逆推 |

**关键**：正推时 ES 同时写入 `+0xB0 / +0x80 / +0x90 / +0xD0` ——
说明**"计划开始"与"预测开始"并非独立计算，而是与 ES 同源的多处落点**。
逆推同理把 LF 回写 `+0xC8 / +0xE8`，LS 回写 `+0xC0 / +0xE0 / +0x80 / +0x90`。

---

## 7 完整调用链

```
FUN_1402730c0  CPM 总控
 ├ 复位（清槽 + 哨兵 + 0x79=0）
 ├ 起点播种（work0[0x48] 要求开始 → 四条线的开始）
 ├ FUN_1401d5510  建访问位图
 ├ 正推
 │   └ FUN_14026ed70（memo：work[0x80]!=哨兵 则返回）
 │        └ FUN_14026e360（单工作：环检测 → 状态短路 → ES 取 max → EF）
 │             ├ FUN_1401d8610（EF = ES + 工期）
 │             └ 递归自身 / FUN_14026ed70
 ├ 逆推
 │   ├ FUN_14026fbd0（预置 LF）
 │   └ FUN_140270660（自身递归：LF 取 min → LS = LF − 工期）
 │        └ FUN_1401d8610（LS = LF − 工期）
 └ 关键判定
     ├ FUN_1401d9940(计划完成, LF)  == 0 → bit0
     ├ FUN_1401d9940(预测完成, LF投) == 0 → bit2
     └ FUN_1401d9940(预测完成, LF)  ≤ 0 → bit1
```

---

## 8 复刻口径（可直接落地）

```ts
// 1) 四条时间线（不是两条）
interface WorkTimes {
  planStart: T; planStop: T;          // +0x80 / +0x88
  estStart: T;  estStop: T;           // +0x90 / +0x98
  es: T; ef: T; ls: T; lf: T;         // +0xB0..+0xC8  （普通）
  esBid: T; efBid: T; lsBid: T; lfBid: T;  // +0xD0..+0xE8（投标）
  duration: number;                    // +0x68
}

// 2) 正推
function forward(w, preds, cal) {
  if (w.status === OnGoing || w.status === Done) {
    return { es: w.realStart, planStart: w.realStart, estStart: w.realStart, esBid: w.realStart };
  }
  let t = { es: SEED, planStart: SEED, estStart: SEED, esBid: SEED };
  for (const p of preds) {
    const r = forward(p, ...);
    t.es        = max(t.es,        r.es);
    t.planStart = max(t.planStart, r.planStart);
    t.estStart  = max(t.estStart,  r.estStart);
    t.esBid     = max(t.esBid,     r.esBid);
  }
  w.ef = addWorkingTime(t.es, w.duration, cal);   // 日历感知
  return t;
}

// 3) 逆推
function backward(w, succs, cal) {
  let lf = MIN_SENTINEL;
  for (const s of succs) lf = min(lf, backward(s, ...).lf);
  w.lf = lf;
  w.ls = addWorkingTime(lf, -w.duration, cal);    // 减法
}

// 4) 关键判定（注意：不显式求 TF，直接比较完成时刻）
w.isCritical    = (w.planStop === w.lf);                    // bit0
w.isCriticalBid = (w.estStop  === w.lfBid);                 // bit2
w.notOverdue    = (w.status !== Done) && (w.estStop <= w.lf); // bit1
```

**三个易错点**：
1. 状态为「进行中 / 已完成」的工作**不参与正推**，ES 直接取实际开始；
2. `计划开始 / 预测开始` 与 ES **同源回写**，不要当成三个独立字段算三遍；
3. 关键判定用「完成 == LF」而不是「LS−ES == 0」，虽然等价，但**跨假期时有细微差别**
   （因为斑马全部用日历感知的加减，差值不是线性差）。

---

---

## 9 五项待证一起补（2026-11-14 续）

### 9.1 日历感知加减的真正算法（`FUN_1401d8610`，2333 B）

```c
FUN_1401d8610(cal, out, base, 工期):
  d = int(工期);
  if (d == 0)  return base;                      // 零工期直接返回

  /* ── 负数：做减法（LS = LF − 工期） ── */
  if (d < 0) {
      t = base + d;                              // 先按自然日粗减
      r = FUN_1401d8f30(cal, base, t);           // 逐日回退，跳过非工作日
      r = FUN_1401db4d0(cal, r);                 // 对齐到最近工作日
      if (r > base) r = base;                    // 不能超过基准
      return r;
  }

  /* ── 正数：做加法（EF = ES + 工期） ── */
  if (base > cal[0x50]) FUN_1401d9ec0(cal, base, cal[0x50]);   // 越界修正到日历起点
  days = int(工期) / *(int *)(cal + 0x90);        // ★ 按"每天单位数"折算成天数
  if (*(int *)(cal + 0x70) <= days) {             // 跨天
      每 100 单位折算一次（FUN_140312560(...,100,...)）
      FUN_1401d9ec0(cal, cal[0x50], 新基准);       // 日历窗口推进
  }
  /* 逐"单位"推进，遇非工作日就跳过 */
  idx = FUN_140312820(base);                       // ★ 日期 → 连续日序索引
  while (位图[cal+0x58][idx>>5] & (1<<(idx&31))) { // ★ cal+0x58 = 非工作日位图
      时间 += 1 单位;
      idx--;
      if (idx < 0) { 跨年 → 回绕 +99 }
      if (days <= idx) 日历窗口再推进
  }
```

**结构要点（这是复刻时间精度必须照抄的）**：

| 成员 | 含义 |
|---|---|
| `cal+0x58` | **非工作日位图**（按连续日序位寻址） |
| `cal+0x70` | 一天的最大单位数（如 24 或 8） |
| `cal+0x90` | 每单位换算系数（"每天单位数"，除法用它把工期折成天） |
| `cal+0x50` | 日历窗口起点（越界时修正到此） |

**日期是"连续日序索引"表示的**，不是时间戳：`FUN_140312820` 由日期算索引，
非工作日 = 位图中该位为 1，**跳过靠索引递增/递减**。
→ **半工作日天然可表示**：整除部分进天数，余数留在时间部分。

### 9.2 总时差 / 自由时差的原始值（`FUN_1402def30`，2776 B）

列取值处的完整写法（`FUN_1402f1490`）：

```c
FUN_140312540(&ff);  FUN_140312540(&x);  FUN_140312540(&tf);
FUN_1402def30(work, &ff, &x, &tf);           // ★ 一次性算出 FF / TF 原始值

/* 自由时差 */
cal = *FUN_1402ef6e0(work, 0);
"free_time_difference"  = (double)toInt(ff) / divisor / (int)(cal + 0x90);

/* 总时差 */
"total_time_difference" = (double)toInt(x)  / divisor / (int)(cal + 0x90);
```

`FUN_1402def30` 内部形态：

```c
极值 = work[0xa8];                                  // 初值
while (沿父链上溯) {
    for (child of 子项) {
        if (child 不是汇总行 && 未跳过)
            极值 = max/min(极值, child[0xa0]);       // 对子项取极值
    }
}
*out_ff = 求差(cal, work[0x15], 极值);               // 自身时间与子项极值求差
```

**结论**：TF/FF **不落盘、按需现算**，且算法是「**沿父子树对子项取极值 → 与自身时间求差**」，
不是简单的 `LF − EF`。最后用 `/(divisor × cal[0x90])` 把内部单位换成"天"。

> 这解释了为什么六时间参数是 `type=4`（计算型）：连时差都是现算的。

### 9.3 关系类型换算（SS/FF/SF）

关系类型字段 = **`节点/关系 +0x7c`**，在正推里有明确分支（`FUN_14026e360`）：

```c
if (pred[0x70] == 1 && (pred[0x7c] == 2 || pred[0x7c] == 3)) {
    FUN_1401d8610(cal, out, pred[0xB8], pred_工期);   // 基准：pred 的 EF
    FUN_1401d8610(cal, out, pred[0x88], pred_工期);   // 基准：pred 的计划完成
    FUN_1401d8610(cal, out, pred[0x98], pred_工期);   // 基准：pred 的预测完成
    FUN_1401d8610(cal, out, pred[0xD8], pred_工期);   // 基准：pred 的投标 EF
}
```

- `+0x7c == 1`：走默认分支（用 pred 的 EF）→ **FS**
- `+0x7c == 2` / `== 3`：改用 pred 的「完成时刻」做基准 → 对应 **SS/FF/SF 一族**
- 换算统一用 `FUN_1401d8610(cal, out, 基准, 工期)`，即"基准 ± 工期"

> 枚举名未在二进制里取到（关系类型存的是整数），但分支位置与基准选择已确定。

### 9.4 逆推的另一半（`FUN_14026fbd0`，2689 B）

```c
FUN_14026fbd0(doc, node, visit):
  /* 与正推同款的环检测位图 */
  idx = (node[0x70] == 1) ? (node[8] + 工作数) : node[8];
  if (visit[idx>>5] & bit) return 0;
  visit[idx>>5] |= bit;

  /* 状态短路：进行中 */
  if (node[0x34c] == 1) {
      node[0xC8] = node[0x60];      // LF = 实际完成 real_stop_time
      node[0xE8] = node[0x60];      // 投标 LF 同
      return 1;
  }

  /* 对每个紧后取 min */
  for (succ of 紧后集合) {
      递归 FUN_14026fbd0;
      t1 = min(t1, succ[0xC8]);     // LF 普通
      t2 = min(t2, succ[0x88]);     // 计划完成
      t3 = min(t3, succ[0x98]);     // 预测完成
      t4 = min(t4, succ[0xE8]);     // LF 投标
  }
```

→ **`FUN_14026fbd0` = 逆推的"取 min + 状态短路"，`FUN_140270660` = 逆推的"用 LF 算 LS"**。
两者互为递归，共同构成完整逆推。

### 9.5 `doc+0x120 / 0x121 / 0x122` 三个开关

三个连续字节，只在 `FUN_1402730c0` 的正推之后被读（后处理），实测语义：

| 开关 | 条件生效 | 动作 | 推断语义 |
|---|---|---|---|
| `doc+0x122` | 非 0 | 遍历工作：对「非首个」且「是汇总」且 `FUN_1403125f0(work[0x10], work[9])` 的，把 `work[9]`（= +0x48 要求开始）置哨兵 | **清空非首个汇总工作的"要求开始"** |
| `doc+0x120` | 非 0 且 `FUN_1402e2e10(work0, 2)==0` | 取 `work0+0x88`（计划完成，`+0x121` 决定是否先取整），然后 `FUN_1402f0200(work0, 该值, 0)` 回写 | **把首工作的计划完成回灌到起点** |
| `doc+0x121` | 仅作 `+0x120` 的内层开关 | 决定是否对 `work0+0x88` 先做 `FUN_140312630` 取整 | **起点回灌时是否取整** |

另有一个 `doc+0x80 == 1` 的开关：取 `max(work0+0x50 要求完成, work0+0x88 计划完成)`，
写回 work0 的 `+0xC8(LF) / +0x98(预测完成) / +0x88(计划完成)` → **起点三个"完成"槽对齐**。

> 三个开关都是**"起点 / 要求时间"的整理策略**，属于"计算完成后收尾"，不参与正逆推本身。

---

## 10 五项待证的完成度

| # | 项 | 状态 |
|---|---|---|
| 1 | 日历感知加减（跨假期/半工作日） | **基本完成**：位图 + 连续日序索引 + 每天单位数；仅跨年回绕细节留白 |
| 2 | 总时差/自由时差原始值 | **完成**：`FUN_1402def30` 树形取极值 + 求差，再两级除法换单位；内部极值选择的完整条件未逐条展开 |
| 3 | SS/FF/SF 换算 | **完成**：类型字段 `+0x7c`，2/3 走不同基准；枚举名未取到 |
| 4 | 逆推预置 LF | **完成**：`FUN_14026fbd0` = 取 min + 状态短路，与 `FUN_140270660` 互补 |
| 5 | 三个开关 | **完成**：都是"起点/要求时间"的整理策略 |
## 11 产物

| 位置 | 内容 |
|---|---|
| `re-zpert/evidence/cpm_core.txt` | CPM 三层结构 + 正推公式 + 时序字段表 + 三 bit 语义 |
| `re-zpert/evidence/decomp_cpm_cand.c` | `FUN_1402730c0`（总控，含关键判定）+ `Work` 构造器 |
| `re-zpert/evidence/decomp_forward.c` | `FUN_14026ed70`（正推递归驱动 / memo） |
| `re-zpert/evidence/decomp_fwdwork.c` | `FUN_14026e360`（单工作正推） |
| `re-zpert/evidence/decomp_backward.c` | `FUN_140270660`（逆推核心） |
| `re-zpert/evidence/decomp_rellag.c` | `FUN_1401d8610`（日历感知加减工期） |
| `re-zpert/evidence/decomp_lfprep.c` | `FUN_14026fbd0`（逆推取 min + 状态短路） |
| `re-zpert/evidence/decomp_fftf.c` | `FUN_1402def30`（自由时差/总时差求值器） |
| `re-zpert/evidence/vtables.txt` | ZQtPertDoc(104) / ZBasePertDoc / Work(6) vtable 索引 |

## 12 剩余待证

| # | 项 |
|---|---|
| 1 | `FUN_14026fbd0`（逆推前预置 LF）的赋值规则 |
| 2 | `FUN_1401d8610` 内部：跨假期/半工作日 的取整规则 |
| 3 | 总时差/自由时差在**列取值时**如何从 LF/EF 求（`total_time_difference` 的两级除法已见，但原始值来源未定位） |
| 4 | `doc+0x120/0x121/0x122` 三个开关的语义（后处理分支） |
| 5 | SS/FF/SF 三种关系在正/逆推里的具体换算（本轮只见参数位置，未读分支内部） |
