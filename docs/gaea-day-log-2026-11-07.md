# 当日总账（2026-11-07 凌晨）：语音管线翻车双根修+五路蒸馏审计+三批修刀收官

> 用户语音实测翻车（「角色答非所问+历次输入重复输出+循环」）→ 生产库取证定性
> → 上游克隆回来同位核对 → 五路蒸馏审计 → 三批修刀全部落地装机。判决=B7 三刀
> 无罪，真凶是刀3 Python→Go 移植漏抄/错挂，长会话才引爆。

## 证据链与根修（昨夜双刀）

- 生产库=`$APPDATA/gaea/whisper_data/hermes.db`（chat_history.session_id='whisper_gaea'）
  +`$APPDATA/gaea/logs/gaea-20261106.log`：转写字符数逐轮爬升 2→6→…→54（每轮实说
  仅 1~1.5s），userText 逐轮累积——输入被污染成历史拼接大串，角色只能对大串回短句。
- 根因1（主犯）：vadhandler `audioPrefix`/`rawPrefix` 跨轮只增不清——上游 HF
  speech-to-speech vad_handler.py `_ensure_turn_for_speech_start`「新轮开清前缀」
  守卫刀3 移植漏抄。修=`ensureTurnForSpeechStart` 同位守卫收口两调用点。
- 根因2（从犯）：finishResponse `strings.Join(rs.parts, " ")`——上游句级 join
  错挂到 delta 粒度=「风 把 白 日」逐字撕裂，落库/面板/人格记忆全污染。修=join ""。
- sidecar 源复得：HF speech-to-speech（Apache-2.0）v1.1.0 克隆 `clones/speech-to-speech`
  （当初 editable 指向 .tmp 已清盘；dist-info direct_url.json 可查源头）。

## 五路蒸馏审计（义务清单先行，台账 docs/gaea-voice-port-distill-20261107.md）

五代理并行逐义务核对 vad_iterator/vad_handler/speculative_turns/response+history/
audio+conversation+session（基线 HEAD=8ebde04=v1.1.0，response.py:991 锚验证）。
结论：主干全同位；漏抄集中「跨态清理守卫」与「客户端控制面/配置面」两翼。
高 2（H1 陈旧 delta 不断流 / H2 被顶替响应 O3 回滚永不执行=assistant 尸体污染
上下文）+中 15+低一批。M11 已核=前端发送面仅 session.update+麦克风帧+response.cancel，
控制面记有意裁剪。

## 三批修刀（全部装机，未 push）

| 批 | 刀 | 内容 |
|---|---|---|
| 批0 | 2 | 前缀清零守卫+done 拼接（vad_prefix_test/transcript_join_test 钉契约） |
| 批1 | 5 | H1 dispatchText 跨 key+DROP 双守卫；H2 finishResponse 被顶替路径补 O3 回滚；M1 echoGuard/baseThreshold 会话复位；M2 audioBatch 属主键清理；M3 llmStage 切史按本轮 user 为界 |
| 批2 | 4 | M4 sttGateWait 死线到期即放行；M5 STT 失败发 transcription.failed；M6 audio.done 收口帧；M7 created 补发孤 done 封死 |
| 批3 | 3 | M12 槽位释放晚于 OnSessionEnd+closedKeys 墓碑替代清零；M13 interrupt_response/create_response 会话级开关生效（前端哑旋钮激活）；M14 silence_duration_ms 运行时可调 |

测试 18 例新钉（port_distill_batch1/2/3 + vad_prefix + transcript_join）；voice
全包×3 绿+前端 voice 53/53 绿+vet/build 零；装机 PID 37380（构建 01:03）。

## 挂账

- M8 记拍板：上游 failed/incomplete 终态与 gaea F1 兜底语/TTS 续播设计冲突
  （failed→flush 截断兜底语），默认维持现状记有意偏离。
- whisper_gaea 今晚污染行清理挂拍板（红线不确认不动持久化数据）。
- 复测要点：连续 5 轮以上看转写是否累积、插话后字幕语音是否脱节、响应是否慢半拍。
- 坑四枚新录：①测试 Handler 内嵌 nil 接口连接期 OnSessionConfig 即炸（五方法全
  no-op）②parseSessionUpdate 收的是 session.update 的 session 字段③OnSessionEnd
  置 p.session=nil 断言线缆须先捕获④已提交轮恒 relevant（DROP 只打未提交响应，
  已提交轮陈旧流靠跨 key 守卫拦；Close 不改 current 须再开新轮才失相关）。

## 追记：语音 UI 升级四刀+离线门（93e0b47b）

- ws_probe `-self-host` 隔离服务端（c44f3178）：协议门 turn 8/8+bargein 10/10+cancel 8/8（真引擎，不碰生产 8791）；顺手修 summarize nil buffer deref。
- 语音 UI 四刀（93e0b47b，ui-ux-pro-max 落地）：打断涟漪（OS/应用级减运动双守卫）/五杆真声纹（波形包络×电平历史去相关）/播放侧真电平 onPlayLevel（speaking 相位声纹由助手真实语音驱动）/现场感字幕；voice 契约 83/83 绿+events 计数锁漂移顺手修（M5 刀二 28→29 存量红）。
- 真语料多轮实弹探针（d37cc151）：真 silero+真 paraformer 同语料两轮雪球签名断言两遍绿——批0 根修端到端复验。坑：探针喂入须 1x 实时节奏（爆发式喂入=STT 门时序竞速伪象）；cgo 引擎测试族 CGO_ENABLED=0 静默全跳（test -c 落 DLL 同目录配方）。
- 污染记忆清理脚本入库（5105f534，默认 dry-run）：删 7 伪影事实（含「明确表达爱意」回声⚠）+1 情节+chat_history t123-128；留真实挫败信号；**等用户一句「清」**。
- 存量移交小说线：NovelSettingPage「重新检查」断言=文件内隔离污染（单跑稳定绿，全文件连跑被前置测试异步残留打翻），非产品回归零交集。
