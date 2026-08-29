# gaea (盖亚) — 2025/2026 Competitive Landscape & Positioning Report

## Executive summary
China's AI-assistant market in 2025–26 has split into three layers: (1) general chat clients and agent platforms (Cherry Studio, Manus, Kimi, Coze/Dify, 豆包 PC) racing to become the "desktop agent hub"; (2) office-suite AI (WPS 灵犀, 千问表格 Agent, 腾讯文档/飞书) locked to SaaS suites and paid memberships; (3) proprietary ecosystems (Claude Cowork, Gemini desktop, Microsoft 365 Copilot) that assume cloud accounts and subscriptions. In parallel, local-first deployment has gone mainstream — open-weight Qwen/DeepSeek models now run comfortably on consumer hardware, with data sovereignty the stated driver. No mainstream product combines a **local-first desktop office agent + personal memory + a construction-cost vertical database + creative/companion modules** in one app. That combination — curated breadth plus one defensible vertical — is gaea's niche; its structural risk is that every individual capability is matched by a better-funded specialist.

## Category-by-category landscape

**A. Desktop AI chat / work clients**
- **Cherry Studio** — open-source multi-provider desktop client; crossed 40k GitHub stars and is described as a "desktop agent hub" ([Agent Times](https://theagenttimes.com/articles/cherry-studio-crosses-40k-stars-the-desktop-agent-hub-the-west-hasnt-noticed-yet)). Strength: provider/MCP ecosystem, knowledge base. Gap: chat+RAG shell — no document-editing agent, no verticals, no personal memory.
- **AnythingLLM** — all-in-one desktop/Docker app with built-in RAG, agents, no-code builder, MCP ([GitHub](https://github.com/Chunde/anything-llm)). Strength: easiest private KB. Gap: workspace-centric retrieval, not an office-document *actor*.
- **LobeChat / Chatbox / Jan / LM Studio** — polished chat shells and local-model runners ([heise](https://www.heise.de/hintergrund/Sprachmodelle-lokal-betreiben-Fuenf-Tools-vorgestellt-10312843.html)). Strength: model coverage. Gap: no plan-then-apply file editing or domain databases.
- **Manus / Kimi Agent** — cloud general agents; Manus opened registration and monetizes via credits ([CLS](https://www.cls.cn/detail/2029121); [Kimi help](https://www.kimi.ai/zh-hant/help/agent/agent-overview)). Strength: autonomous breadth. Gap: cloud-only, credit pricing, no local data control.
- **Claude Cowork** (Windows desktop: files, coding, tasks; [ET CIO](https://cio.economictimes.indiatimes.com/news/artificial-intelligence/anthropic-launches-claude-cowork-feature-for-windows-desktop-app/129381536)) and **Gemini Mac app** ([Bianews](https://www.bianews.com/news/details?id=236130)) — closest architectural analogues; strength: first-party models; gap: subscription, cloud, weak China availability. **豆包 PC** dominates mass-market China desktops ([Baike](https://baike.baidu.com/item/%E8%B1%86%E5%8C%85PC%E7%AB%AF/67468345)) but is chat-first.

**B. AI office / document agents (China)**
- **WPS 灵犀** — suite-native docx/xlsx/pdf AI, now separately billed, sparking membership backlash ([SMZDM](https://post.smzdm.com/p/anvqx482/)). Strength: document fidelity. Gap: closed suite, cloud processing, no review step before table edits.
- **千问表格 Agent** — Qwen's spreadsheet agent "抢跑 Excel 智能化" ([Toutiao](https://www.toutiao.com/article/7628484181877637659/)); 腾讯/字节/阿里 are all pushing AI office ([21jingji](https://m.21jingji.com/article/20260805/herald/e37d6ef40136f7605ea2796e952c8ef5.html)). Gap: cloud-bound, quota-limited, no persistent per-project memory.
- **Kimi office features** — long-doc reading, PPT, file workflows ([163](https://m.163.com/dy/article/KT9E6TNQ05169CIL.html)). Strength: huge-context reading. Gap: thin structured-file editing.
- gaea fills: plan-then-apply xlsx review, local file access, 三脑 memory, task templates, .gaea/AGENTS.md conventions — none of the above offer a reviewable edit plan or local execution.

**C. 工程造价 / cost vertical**
- **广联达 (Glodon)** — incumbent champion now branding "AI+数据，让造价更精准" ([Xinhua](https://app.xinhuanet.com/news/article.html?articleId=6d5b7ac49b10c3a5dbbcd88ef19983a9)) and AI+全场景 at BAU China 2025 ([Stockstar](https://4g.stockstar.com/detail/IG2025111300011679)). Strength: 定额/计价 standards dominance, firm-level distribution. Gap: enterprise-priced suites; AI bolted onto legacy products; nothing for the individual engineer.
- **斯维尔/鲁班/品茗** — regional incumbents with little conversational AI ([Askci report](https://m.askci.com/reports/20251126/1639170278276414635757322026.shtml)).
- **CostOS** — international AI estimating/benchmarking platform ([NexusAI](https://www.nexusai-tech.com/zh/ai-apps/cost-os-ai-driven-cost-estimating-and-benchmarking-platform)); an "AI 工程造价数据库" patent shows the space is only now emerging ([Tianyancha](https://m.tianyancha.com/zhuanli/d8d9e557bca0a9e4f9265f17f294d231)).
- Verdict: **no AI-native, personal cost database with 综合单价/人材机 + 分位数对标 + 复盘笔记 exists** — gaea's most defensible vertical.

**D. Novel-writing AI**
- **阅文 作家助手·妙笔** — platform-locked, first with 千万字-level comprehension ([Baike](https://baike.baidu.com/item/%E4%BD%9C%E5%AE%B6%E5%8A%A9%E6%89%8B%E5%A6%99%E7%AC%94%E7%89%88/63227250); [Sohu](https://www.sohu.com/a/944943008_100116740)). Strength: author distribution, IP pipeline. Gap: cloud/platform-tied.
- **NovelAI / Sudowrite** — subscription prose tools with story-bible workflows ([Popi.ai](https://popi.ai/compare/writing-tools/sudowrite-vs-novelai/)). Gap: English-first, no reading loop.
- **微信读书 AI 朗读** — TTS listening, criticized as flat ([SMZDM](https://post.smzdm.com/p/agg6nkr3/)). gaea uniquely merges writing + reading + 伴读 + TTS + EPUB export locally.

**E. Companion / persona AI**
- **星野 (MiniMax)** and **猫箱 (ByteDance)** — leaders; 猫箱 reaches ~125 min/day DAU, catching 星野 ([Oriental Securities](https://www.sgpjbg.com/labelsyh/maoxiangaiqingganpeiban/1/6749051.html)). Strength: engagement, character ecosystem. Gap: mobile-entertainment-first, cloud memory, and now regulatory tightening on 情感陪伴类 AI ([SFCCN](https://m.sfccn.com/2025/12-29/xNMDE0NDlfMjA5MjQxNw.html)); the segment monetizes heavily (AI 伴侣半年吸金超5亿元, 75 min/day; [iResearch](https://news.iresearch.cn/content/202508/531576.shtml)) but burns cash ([Jiemian](https://m.jiemian.com/article/14081072.html)). gaea's 轻语 niche: a local, private, memory-continuous companion that also sees your real work — nobody does both.

**F. Local-first momentum**
- Local stacks are mainstream: Qwen "装进电脑" ([Aliyun](https://developer.aliyun.com/article/1688634)), Ollama+AnythingLLM zero-cost RAG ([Baidu](https://cloud.baidu.com/article/3891916)); 地端 SLM for 自主可控 is an enterprise trend ([NetAdmin](https://www.netadmin.com.tw/netadmin/zh-tw/magazine/-Trend/D8CBAFD29A8F4D7393564E0617431BA3)); open weights are a pricing hedge ([Kafkai](https://kafkai.ai/articles/ai-technology/ai-bubble-pricing-implosion-part-3/)); desktop is the under-measured trust surface ([Olakai](https://olakai.ai/blog/desktop-renaissance-ai-measurement-gap/)).
- Verdict: "data not leaving the machine" **is** a real differentiator in China — strongest when paired with cloud models via OAuth, exactly gaea's model center design.

**G. Embedded coding-agent board**
- Claude Code drove Anthropic to ~$14B ARR ([SaaStr](https://www.saastr.com/anthropic-just-hit-14-billion-in-arr-up-from-1-billion-just-14-months-ago/)) with reported doubling of annualized run-rate within months ([Bitget](https://www.bitget.com/zh-CN/amp/news/detail/12560605396205)); Codex shipped desktop/IDE upgrades ([OpenAI](https://openai.com/fr-CA/index/introducing-upgrades-to-codex/)); Cursor remains the benchmark ([CNBC](https://www.cnbc.com/2026/05/19/cursor-cnbc-disruptor-50-ranking.html)). Claude Cowork proves the pattern: coding + file management inside a desktop assistant ([guide](https://github.com/FlorianBruniaux/claude-code-ultimate-guide/blob/614dcc46/guide/cowork.md)). gaea's embedded DeepSeek Harness board is the China-local, low-cost version of Cowork.

## gaea's differentiation
1. **Only all-in-one pairing** a local-first office agent (plan-then-apply xlsx, docx/pdf) with 三脑 memory + dream distillation; rivals offer chat+RAG (Cherry/AnythingLLM) or cloud agents (Manus/Kimi), never both.
2. **造价数据库** — a genuine blue ocean: incumbents are enterprise-priced, no AI-native personal cost DB with 分位数对标 exists.
3. **Cross-module flywheel** — worldview/characters feed companion chat; documents feed memory; cost DB feeds the office agent.
4. **Model-agnostic + embedded coding board** — Wails/Go desktop app, no subscription lock, free local coding agent.

## Gaps / risks
- **Single-user, no mobile/cloud sync** — every major rival is multi-device; caps reach and retention.
- **Breadth vs depth** — each module meets a richer specialist (WPS in docs, 猫箱 in companionship, 广联达 in 造价) with more polish and data moats.
- **Compliance** — 情感陪伴 rules and generative-AI filing add cost if commercialized; the WeChat assistant sits in a grey zone.
- **Distribution** — zero brand vs 豆包/Manus marketing machines; output quality still depends on Grok/DeepSeek APIs.

## Blue-ocean opportunities (next iteration)
1. **造价 prosumer SaaS**: sell the personal cost DB + 分位数对标 to individual 造价工程师/咨询人 — the gap between Excel and 广联达.
2. **"Personal data vault" positioning**: local-first compliance story for lawyers, design institutes, freelancers handling sensitive documents.
3. **Template/convention marketplace**: shareable task templates + .gaea/AGENTS.md packs create a user ecosystem incumbents can't copy.
4. **Companion × productivity fusion**: a 轻语 that actually does your paperwork — defensible against pure-entertainment apps and regulators alike.
5. **China-local "Claude Cowork alternative"**: market the embedded coding board + model center as the open, free, local-first Cowork.
