# 聊天人格实弹基线对照报告（众神殿 action-1 执行）

- 日期：2026-11-06 凌晨
- 模型：qwen35 = `Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive-Q4_K_P-MTP-r2`（llama 自管引擎 127.0.0.1:8124，与 herdsman 槽位**同一份 GGUF**，同模型对照成立；herdsman 槽位当时 stopped，未动用户引擎状态）
- 角色：deredere（非 gaea 内置人格，论证钉住的「同一角色」）
- 环境：测试 TempDir 数据根隔离，不触碰生产会话；搜索腿固定罐头（离线确定性）
- 挂具：`internal/app/whisper_probe_baseline_test.go`（环境门控 `GAEA_PROBE=1`，CI 跳过）
- 两轮：baseline-13b65f37（刀0 tag，四切片修前）vs head-16b52dda（刀1~4 修后），各 ~57 轮 / 287s vs 268s

## 一、四切片修前→修后对照（决定性差异）

| 指标 | baseline | head | 结论 |
|---|---|---|---|
| 重启差分·事实回忆（「我说想去哪里旅行来着？」） | **挂**——答「想去有蓝色天空的地方」（跑题） | **过**——「嗯，是想去成都旅行。还要去吃火锅」 | 刀4 生效 |
| 重启差分·续接（「继续刚才那个」） | **挂**——继续聊蓝色天空 | **过**——「嗯，成都吃火锅」 | 刀4 生效 |
| 生日假事实 | **活**——ackemBirthday==firstMetDate（implicit=true），首聊日周年必注「我的生日」 | **灭**——ackemBirthday=null（implicit=false） | 刀2 生效 |
| 空回复/假成功 | 0/57（与旧库 1/99 一致=低频） | 0/57，且终态全 done | 刀1 为纵深防御，不计主因（DSH 判断实弹复核成立） |
| 并发首轮 | done×2（未撞恢复窗口） | done×2 | 刀4 发布序为窗口关闭（确定性保障） |
| 全部 118 轮终态 | 31+15+5+4+2 全 done 零 error | 同左 | 无回归 |

## 二、两臂实验的靶子（两轮一致，压制服未动=预期保留）

1. **strategy whisper 恒中：46/46 轮**（五桶 31+连续 15）——Stage 57 轮**全程 STRANGER**（Trust 终值 38~40、Aff 终值 65 也不晋级），「话要少，不要开启新话题」每轮注入。
2. **gate silent 只在前期**：Aff<30 的 16/31 轮命中，连续闲聊段 Aff 破 30 后 0 命中——**ZCode 的「Aff<30 永久稳态」可证伪预测被部分证伪**：Aff 会过线，但 **Stage 不晋级才是 whisper 恒注的真根**（`relationship.go` positiveTypes 只有 praise/tease/vulnerable/apology，普通问答不推进）。
3. **钳制命中 20/46**（≤5 字/6–9 字祈使桶为主），长句追问 0 命中（豁免生效）。
4. 压制命中后追问：head 均值 49 字 vs baseline 23 字（n=5+LLM 温度噪声，不作判据）；两轮「回复短但贴题」，未出现论证担心的「极短冷漠」——**本轮证据不支持「回复长度」作为验收指标**（论证口径再次成立）。
5. 生日注入对 baseline 前 30 轮的 psycheBlock 有扰动（钳制 13→14 的小漂移），未产生「今天是我的生日」式跑题回复——该病在首聊日**周年**才必现，本轮只证机制存活/已灭（artifact 级），不证回复面差异。

## 三、下一步定级建议（交两臂实验）

- **主修靶 = Stage 晋级机制**（whisper 的 Stage==Stranger 谓词）而非 gate aff 阈值：臂 A（纯文案 A′）对此结构性无效的预测被实弹强化；臂 B（意图/义务门控+慢热显式模式化）应把「Stage 不晋级」列为并列根因。
- 钳制维持短祈使桶 P1（20/46 集中在短句桶）。
- 多轮续接：重启差分里「继续刚才那个」在 HEAD 已能借回灌历史接住——但**同进程二次检索缺口未测**（本批搜索腿为罐头），维持「复现门槛」。

## 附：原始报告

- `.tmp/probe/head/{report.json,summary.md,run.log}`、`.tmp/probe/baseline/{…}`（.tmp 不入库，关键数据已录入本文）
- 复跑配方：`GAEA_PROBE=1 GAEA_PROBE_LABEL=<tag> GAEA_PROBE_OUT=<dir> go test ./internal/app/ -run TestWhisperProbeBaseline -count=1 -timeout 35m`（需 8124 引擎在线；baseline 侧在 `git worktree` 于 tag `chat-persona-baseline` 上挂入同一挂具）
