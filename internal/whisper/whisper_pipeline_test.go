// Package whisper — whisper_pipeline_test.go
// 管线测试深化：memory_ingest / association_cold_start / paced_stream / memory 检索路径

package whisper

import (
	"strings"
	"testing"
	"time"
)

// ─── mock LlmClient ──────────────────────────────────────────

type mockLlm struct {
	chatFunc func(system, user string) (string, error)
}

func (m *mockLlm) Chat(system, user string) (string, error) {
	if m.chatFunc != nil {
		return m.chatFunc(system, user)
	}
	return "", nil
}

// ─── memory_ingest: buildEpisodeSummary / extractKeywords ─────

func TestBuildEpisodeSummary_UserAndAssistant(t *testing.T) {
	got := buildEpisodeSummary([]ExchangePair{
		{User: "今天下雨了", Assistant: "记得带伞哦"},
		{User: "好的谢谢", Assistant: "不客气"},
	})
	if !strings.Contains(got, "用户说「今天下雨了…」") {
		t.Errorf("缺少用户侧摘要: %s", got)
	}
	if !strings.Contains(got, "Hermes回应「不客气…」") {
		t.Errorf("缺少助手侧摘要: %s", got)
	}
}

func TestBuildEpisodeSummary_OnlyUser(t *testing.T) {
	got := buildEpisodeSummary([]ExchangePair{{User: "早安"}})
	if !strings.Contains(got, "用户说「早安…」") {
		t.Errorf("只有用户消息时摘要异常: %s", got)
	}
	if strings.Contains(got, "Hermes") {
		t.Errorf("无助手消息不应出现 Hermes: %s", got)
	}
}

func TestBuildEpisodeSummary_Empty(t *testing.T) {
	if got := buildEpisodeSummary(nil); got != "" {
		t.Errorf("空输入应返回空串, got %q", got)
	}
}

func TestBuildEpisodeSummary_TruncatesLong(t *testing.T) {
	long := strings.Repeat("长", 200)
	got := buildEpisodeSummary([]ExchangePair{{User: long, Assistant: long}})
	if len([]rune(got)) > 200 {
		t.Errorf("摘要应截断超长文本, got %d runes", len([]rune(got)))
	}
}

func TestExtractKeywords_ChinesePunctuation(t *testing.T) {
	got := extractKeywords("我喜欢吃火锅，也喜欢甜品")
	if len(got) != 2 || got[0] != "我喜欢吃火锅" || got[1] != "也喜欢甜品" {
		t.Errorf("中文标点切分异常: %v", got)
	}
}

func TestExtractEpisodeKeywords_NoDupAndCaps(t *testing.T) {
	exchanges := make([]ExchangePair, 0, 12)
	for i := 0; i < 12; i++ {
		exchanges = append(exchanges, ExchangePair{User: "重复主题", Assistant: "呼应主题"})
	}
	got := extractEpisodeKeywords(exchanges)
	if len(got) > 10 {
		t.Errorf("关键词应上限 10, got %d", len(got))
	}
	seen := map[string]bool{}
	for _, k := range got {
		if seen[k] {
			t.Errorf("关键词重复: %s", k)
		}
		seen[k] = true
	}
}

// ─── memory_ingest: AfterTurn 管线 ───────────────────────────

func TestAfterTurn_AutoRetireAtInterval(t *testing.T) {
	fs := NewFactStore()
	// 低置信度事实满足退役条件（Confidence < 0.3）
	fs.Add(MemoryFact{Domain: "SOCIAL", Subcategory: "FRIENDS", Subject: "路人", Summary: "一次闲聊", Confidence: 0.2, Weight: 2})

	p := NewMemoryIngestPipeline(nil)
	args := IngestTurnArgs{
		SessionID:  "s1",
		TurnIndex:  10,
		FactStore:  fs,
		TotalTurns: AutoRetireCheckInterval, // 10
		Opts:       IngestOptions{SkipLlmExtraction: true},
	}
	p.AfterTurn(args)
	if fs.Count() != 0 {
		t.Errorf("TotalTurns=10 应触发自动退役, Count=%d", fs.Count())
	}
}

func TestAfterTurn_NoAutoRetireBelowInterval(t *testing.T) {
	fs := NewFactStore()
	fs.Add(MemoryFact{Domain: "SOCIAL", Subcategory: "FRIENDS", Subject: "路人", Summary: "一次闲聊", Confidence: 0.2, Weight: 2})

	p := NewMemoryIngestPipeline(nil)
	p.AfterTurn(IngestTurnArgs{
		SessionID:  "s1",
		TurnIndex:  5,
		FactStore:  fs,
		TotalTurns: 5,
		Opts:       IngestOptions{SkipLlmExtraction: true},
	})
	if fs.Count() != 1 {
		t.Errorf("TotalTurns=5 不应触发退役, Count=%d", fs.Count())
	}
}

func TestAfterTurn_ExtractsTriples(t *testing.T) {
	fs := NewFactStore()
	fs.Add(MemoryFact{
		Domain: "user_profile", Subcategory: "BASIC_PROFILE",
		Subject: "姓名", Summary: "叫小明", Confidence: 0.9, Weight: 3,
		SourceSessionID: "s1", SourceTurnIndex: 7,
	})
	kg := NewKnowledgeGraph()

	p := NewMemoryIngestPipeline(nil)
	p.AfterTurn(IngestTurnArgs{
		SessionID:  "s1",
		TurnIndex:  7,
		FactStore:  fs,
		TotalTurns: 7,
		KG:         kg,
		Opts:       IngestOptions{SkipLlmExtraction: true},
	})
	if kg.Size() == 0 {
		t.Error("BASIC_PROFILE 事实应提取为三元组")
	}
}

func TestAfterTurn_EpisodeGeneratedAtLowInterval(t *testing.T) {
	es := NewEpisodicStore()
	p := NewMemoryIngestPipeline(nil)
	exchanges := []ExchangePair{
		{User: "今天加班好累", Assistant: "辛苦了，早点休息"},
		{User: "嗯，明天还要开会", Assistant: "记得准备材料"},
		{User: "好的，晚安", Assistant: "晚安好梦"},
	}
	p.AfterTurn(IngestTurnArgs{
		SessionID:       "s1",
		TurnIndex:       10,
		UserMsg:         "晚安",
		CompanionMsg:    "晚安好梦",
		L1:              L1State{Trust: 60},
		L2:              EmotionState{Aff: 10, Sec: 5}, // 低强度 → 低间隔 10
		TotalTurns:      EpisodeIntervalTurnsLow,       // 10
		EpisodicStore:   es,
		RecentExchanges: exchanges,
		Opts:            IngestOptions{SkipLlmExtraction: true},
	})
	if es.Latest() == nil {
		t.Fatal("TotalTurns=10 低强度应生成情节")
	}
	if !strings.Contains(es.Latest().Summary, "用户说") {
		t.Errorf("情节摘要应包含用户侧文本: %s", es.Latest().Summary)
	}
}

func TestAfterTurn_EpisodeGeneratedAtHighInterval(t *testing.T) {
	es := NewEpisodicStore()
	p := NewMemoryIngestPipeline(nil)
	exchanges := []ExchangePair{
		{User: "我今天好开心", Assistant: "太好了"},
		{User: "升职了", Assistant: "恭喜你"},
		{User: "谢谢", Assistant: "不客气"},
	}
	p.AfterTurn(IngestTurnArgs{
		SessionID:       "s1",
		TurnIndex:       6,
		UserMsg:         "谢谢",
		CompanionMsg:    "不客气",
		L1:              L1State{Trust: 80},
		L2:              EmotionState{Aff: 80, Sec: 60}, // 高强度 → 高间隔 6
		TotalTurns:      EpisodeIntervalTurns,            // 6
		EpisodicStore:   es,
		RecentExchanges: exchanges,
		Opts:            IngestOptions{SkipLlmExtraction: true},
	})
	if es.Latest() == nil {
		t.Fatal("TotalTurns=6 高强度应生成情节")
	}
}

func TestAfterTurn_EpisodeNotGeneratedBeforeInterval(t *testing.T) {
	es := NewEpisodicStore()
	p := NewMemoryIngestPipeline(nil)
	exchanges := []ExchangePair{
		{User: "今天天气不错", Assistant: "是的"},
		{User: "适合散步", Assistant: "去吧"},
		{User: "好", Assistant: "嗯"},
	}
	p.AfterTurn(IngestTurnArgs{
		SessionID:       "s1",
		TurnIndex:       5,
		UserMsg:         "好",
		CompanionMsg:    "嗯",
		L1:              L1State{Trust: 60},
		L2:              EmotionState{Aff: 10, Sec: 5},
		TotalTurns:      5,
		EpisodicStore:   es,
		RecentExchanges: exchanges,
		Opts:            IngestOptions{SkipLlmExtraction: true},
	})
	if es.Latest() != nil {
		t.Error("TotalTurns=5 未到间隔不应生成情节")
	}
}

func TestAfterTurn_LlmExtractionWritesFacts(t *testing.T) {
	fs := NewFactStore()
	llm := &mockLlm{chatFunc: func(system, user string) (string, error) {
		return `{"facts":[{"domain":"user_profile","subcategory":"BASIC_PROFILE","subject":"宠物","summary":"养了一只猫","weight":0.8,"confidence":0.9,"selfRelevance":0.8}]}`, nil
	}}
	p := NewMemoryIngestPipeline(llm)
	p.AfterTurn(IngestTurnArgs{
		SessionID:    "s1",
		TurnIndex:    1,
		UserMsg:      "我家有只猫",
		CompanionMsg: "好可爱",
		L1:           L1State{Trust: 60},
		L2:           EmotionState{Aff: 10, Sec: 5},
		FactStore:    fs,
		TotalTurns:   1,
		Opts:         IngestOptions{},
	})
	if fs.Count() != 1 {
		t.Fatalf("LLM 抽取应写入 1 条事实, Count=%d", fs.Count())
	}
	f := fs.ListActive()[0]
	if f.Subcategory != "BASIC_PROFILE" || !strings.Contains(f.Summary, "猫") {
		t.Errorf("LLM 抽取事实字段异常: %+v", f.MemoryFact)
	}
	if f.PrivacyLevel != "" {
		t.Errorf("默认隐私级别应为空: %q", f.PrivacyLevel)
	}
}

func TestAfterTurn_LlmExtractionRespectsAdultPrivacy(t *testing.T) {
	fs := NewFactStore()
	llm := &mockLlm{chatFunc: func(system, user string) (string, error) {
		return `{"facts":[{"domain":"user_profile","subcategory":"BASIC_PROFILE","subject":"关系","summary":"处于亲密关系中","weight":0.9,"confidence":0.9,"selfRelevance":0.9}]}`, nil
	}}
	p := NewMemoryIngestPipeline(llm)
	p.AfterTurn(IngestTurnArgs{
		SessionID:    "s1",
		TurnIndex:    1,
		UserMsg:      "我有对象了",
		CompanionMsg: "祝福你们",
		L1:           L1State{Trust: 60},
		L2:           EmotionState{Aff: 10, Sec: 5},
		FactStore:    fs,
		TotalTurns:   1,
		Opts:         IngestOptions{AdultPrivacyLevel: "intimate"},
	})
	if fs.Count() != 1 {
		t.Fatalf("应写入 1 条事实, Count=%d", fs.Count())
	}
	if got := fs.ListActive()[0].PrivacyLevel; got != "intimate" {
		t.Errorf("隐私级别应透传 intimate, got %q", got)
	}
}

// ─── association_cold_start ──────────────────────────────────

func TestPickAssociationType_SameSubcategory(t *testing.T) {
	a := &MemoryFact{Subcategory: "FRIENDS"}
	b := &MemoryFact{Subcategory: "FRIENDS"}
	if got := pickAssociationType(a, b); got != "event_chain" {
		t.Errorf("同子类应为 event_chain, got %s", got)
	}
}

func TestPickAssociationType_DifferentSubcategory(t *testing.T) {
	a := &MemoryFact{Subcategory: "FRIENDS"}
	b := &MemoryFact{Subcategory: "CAREER"}
	if got := pickAssociationType(a, b); got != "thematic" {
		t.Errorf("不同子类应为 thematic, got %s", got)
	}
}

func TestTextOverlapScore_SharedGrams(t *testing.T) {
	a := &MemoryFact{Subject: "用户", Summary: "喜欢喝咖啡"}
	b := &MemoryFact{Subject: "用户", Summary: "咖啡是提神利器"}
	if got := textOverlapScore(a, b); got <= 0 {
		t.Errorf("共享词应产生重叠, got %v", got)
	}
}

func TestTextOverlapScore_NoOverlap(t *testing.T) {
	a := &MemoryFact{Subject: "张三", Summary: "围棋三段高手"}
	b := &MemoryFact{Subject: "李四", Summary: "量子物理研究员"}
	if got := textOverlapScore(a, b); got != 0 {
		t.Errorf("无共享词应返回 0, got %v", got)
	}
}

func TestBatchSeedAssociationsFromTextOverlap_LinksOrphans(t *testing.T) {
	fs := NewFactStore()
	fs.Add(MemoryFact{Domain: "SOCIAL", Subcategory: "FRIENDS", Subject: "用户", Summary: "喜欢和咖啡店老板聊天", Weight: 2})
	fs.Add(MemoryFact{Domain: "SOCIAL", Subcategory: "FRIENDS", Subject: "用户", Summary: "咖啡店老板也是常客", Weight: 2})
	fs.Add(MemoryFact{Domain: "SOCIAL", Subcategory: "FRIENDS", Subject: "用户", Summary: "咖啡店在楼下", Weight: 2})
	index := NewAssociationIndex()

	res := BatchSeedAssociationsFromTextOverlap(fs, index, 2, 10, 3)
	if res.EdgesCreated == 0 {
		t.Error("文本重叠应建边")
	}
	if res.FactsConsidered == 0 {
		t.Error("FactsConsidered 应 > 0")
	}
	if res.OrphansLinked == 0 {
		t.Error("孤儿事实应被链接")
	}
	if index.Count() == 0 {
		t.Error("关联索引应有边")
	}
}

func TestBatchSeedAssociationsFromTextOverlap_DifferentDomainNoLink(t *testing.T) {
	fs := NewFactStore()
	fs.Add(MemoryFact{Domain: "SOCIAL", Subcategory: "FRIENDS", Subject: "用户", Summary: "周末去爬山和野餐", Weight: 2})
	fs.Add(MemoryFact{Domain: "CAREER", Subcategory: "GOALS", Subject: "用户", Summary: "周末也要努力学编程", Weight: 2})
	index := NewAssociationIndex()

	res := BatchSeedAssociationsFromTextOverlap(fs, index, 1, 10, 3)
	if res.EdgesCreated != 0 {
		t.Errorf("不同域不应建边, EdgesCreated=%d", res.EdgesCreated)
	}
}

func TestLinkForColdStart_ExistingEdgeNoDoubleCount(t *testing.T) {
	index := NewAssociationIndex()
	index.Add(Association{FactIDA: "a", FactIDB: "b", AssociationType: "thematic", Strength: 0.35})

	created := linkForColdStart(index, "a", "b", "thematic")
	if created {
		t.Error("已存在边不应算新建")
	}
	// 反序同样不新建
	if created := linkForColdStart(index, "b", "a", "thematic"); created {
		t.Error("反序已存在边也不应算新建")
	}
	if index.Count() != 1 {
		t.Errorf("边数应保持 1, got %d", index.Count())
	}
}

func TestLinkForColdStart_NewEdgeCreated(t *testing.T) {
	index := NewAssociationIndex()
	created := linkForColdStart(index, "x", "y", "thematic")
	if !created {
		t.Error("新边应创建")
	}
	if index.Count() != 1 {
		t.Errorf("应恰有 1 条边, got %d", index.Count())
	}
}

// ─── paced_stream ────────────────────────────────────────────

func TestStripSplitMarkers(t *testing.T) {
	if got := StripSplitMarkers("你好[SPLIT]再见"); got != "你好再见" {
		t.Errorf("应去掉 [SPLIT], got %q", got)
	}
	if got := StripSplitMarkers("  [SPLIT]  "); got != "" {
		t.Errorf("纯分隔符应得到空串, got %q", got)
	}
}

func TestFirstDisplayUnitLen_Empty(t *testing.T) {
	if got := FirstDisplayUnitLen(""); got != -1 {
		t.Errorf("空输入应返回 -1, got %d", got)
	}
}

func TestFirstDisplayUnitLen_SplitPrefix(t *testing.T) {
	if got := FirstDisplayUnitLen("[SPLIT]abc"); got != len(SplitMarker) {
		t.Errorf("前缀分隔符应返回标记长度 %d, got %d", len(SplitMarker), got)
	}
}

func TestFirstDisplayUnitLen_SplitInside(t *testing.T) {
	got := FirstDisplayUnitLen("你好[SPLIT]再见")
	if got != 6 {
		t.Errorf("分隔符前中文应返回字节偏移 6, got %d", got)
	}
}

func TestFirstDisplayUnitLen_SentenceBreak(t *testing.T) {
	got := FirstDisplayUnitLen("你好。再见")
	if got != 9 {
		t.Errorf("完整句子含句号应返回字节偏移 9, got %d", got)
	}
}

func TestFirstDisplayUnitLen_NoBreak(t *testing.T) {
	if got := FirstDisplayUnitLen("abcdef"); got != -1 {
		t.Errorf("无断点应返回 -1, got %d", got)
	}
}

func TestPacedStreamEmitter_FlushesWholeText(t *testing.T) {
	var chunks []string
	var bubbleTexts []string
	p := NewPacedStreamEmitter(PacedStreamCallbacks{
		OnChunk:     func(c string) { chunks = append(chunks, c) },
		OnBubbleEnd: func(_ int, text string) { bubbleTexts = append(bubbleTexts, text) },
	}, 1)

	p.OnDelta("你好")
	p.MarkDone()

	if got := strings.Join(chunks, ""); got != "你好" {
		t.Errorf("OnChunk 累计应等于原文, got %q", got)
	}
	if len(bubbleTexts) != 1 || bubbleTexts[0] != "你好" {
		t.Errorf("应有一个完整气泡, got %v", bubbleTexts)
	}
}

func TestPacedStreamEmitter_SplitSeparatesBubbles(t *testing.T) {
	var chunks []string
	var bubbleTexts []string
	p := NewPacedStreamEmitter(PacedStreamCallbacks{
		OnChunk:     func(c string) { chunks = append(chunks, c) },
		OnBubbleEnd: func(_ int, text string) { bubbleTexts = append(bubbleTexts, text) },
	}, 1)

	p.OnDelta("你好[SPLIT]再见")
	p.MarkDone()

	if got := strings.Join(chunks, ""); got != "你好再见" {
		t.Errorf("OnChunk 累计应为去分隔符文本, got %q", got)
	}
	if len(bubbleTexts) != 2 || bubbleTexts[0] != "你好" || bubbleTexts[1] != "再见" {
		t.Errorf("[SPLIT] 应分隔两个气泡, got %v", bubbleTexts)
	}
}

// ─── memory 检索路径 ─────────────────────────────────────────

func TestRetrieve_TriggerBoost(t *testing.T) {
	fs := NewFactStore()
	fs.Add(MemoryFact{
		Domain: "SOCIAL", Subcategory: "FRIENDS", Subject: "用户", Summary: "最好的朋友叫阿珍",
		Weight: 2, Confidence: 0.9, SelfRelevance: 0.8, Triggers: []string{"阿珍"},
	})
	mr := NewRetriever(fs, nil)

	res := mr.Retrieve("阿珍今天要来", RelevanceHint{}, 500, 0, 30, nil, "s1", false)
	if res.FactsUsed != 1 {
		t.Errorf("触发词命中应检索到事实, FactsUsed=%d", res.FactsUsed)
	}
	if !strings.Contains(res.TierBBlock, "阿珍") {
		t.Errorf("TierBBlock 应含触发事实, got %q", res.TierBBlock)
	}
}

func TestRetrieve_PrivacyFilteredWithoutAdult(t *testing.T) {
	fs := NewFactStore()
	fs.Add(MemoryFact{
		Domain: "SOCIAL", Subcategory: "PARTNER", Subject: "用户", Summary: "亲密关系细节",
		Weight: 3, Confidence: 0.9, SelfRelevance: 0.9, PrivacyLevel: "intimate", Triggers: []string{"伴侣"},
	})
	mr := NewRetriever(fs, nil)

	res := mr.Retrieve("伴侣今天来了", RelevanceHint{}, 500, 0, 30, nil, "s1", false)
	if res.FactsUsed != 0 {
		t.Errorf("非成人模式应过滤 intimate 事实, FactsUsed=%d", res.FactsUsed)
	}
}

func TestRetrieve_PrivacyIncludedWithAdult(t *testing.T) {
	fs := NewFactStore()
	fs.Add(MemoryFact{
		Domain: "SOCIAL", Subcategory: "PARTNER", Subject: "用户", Summary: "亲密关系细节",
		Weight: 3, Confidence: 0.9, SelfRelevance: 0.9, PrivacyLevel: "intimate", Triggers: []string{"伴侣"},
	})
	mr := NewRetriever(fs, nil)

	res := mr.Retrieve("伴侣今天来了", RelevanceHint{}, 500, 0, 30, nil, "s1", true)
	if res.FactsUsed != 1 {
		t.Errorf("成人模式应包含 intimate 事实, FactsUsed=%d", res.FactsUsed)
	}
}

func TestRetrieve_BudgetCapsFacts(t *testing.T) {
	fs := NewFactStore()
	for i := 0; i < 20; i++ {
		fs.Add(MemoryFact{
			Domain: "DAILY_LIFE", Subcategory: "ROUTINES", Subject: "用户",
			Summary: "日常记录第" + strings.Repeat("项", 50) + string(rune('A'+i)),
			Weight: 2, Confidence: 0.8, SelfRelevance: 0.7,
		})
	}
	mr := NewRetriever(fs, nil)

	res := mr.Retrieve("日常", RelevanceHint{}, 130, 0, 30, nil, "s1", false)
	if res.FactsUsed == 0 {
		t.Error("budget 内应至少检索到 1 条")
	}
	if res.FactsUsed >= 20 {
		t.Errorf("小 budget 应限制事实数, FactsUsed=%d", res.FactsUsed)
	}
}

func TestRetrieve_EmptyStore(t *testing.T) {
	mr := NewRetriever(NewFactStore(), nil)
	res := mr.Retrieve("随便问问", RelevanceHint{}, 500, 0, 30, nil, "s1", false)
	if res.SharedCount != 0 {
		t.Errorf("空库 SharedCount 应为 0, got %d", res.SharedCount)
	}
	if res.TierBBlock != "" || res.FactsUsed != 0 {
		t.Errorf("空库不应有检索块: %q / %d", res.TierBBlock, res.FactsUsed)
	}
}

func TestRetrieve_EpisodesSearched(t *testing.T) {
	fs := NewFactStore()
	fs.Add(MemoryFact{Domain: "DAILY_LIFE", Subcategory: "ROUTINES", Subject: "用户", Summary: "今天没特别事", Weight: 1, Confidence: 0.7, SelfRelevance: 0.5})
	es := NewEpisodicStore()
	es.Add(Episode{Summary: "一起喝了咖啡", Keywords: []string{"咖啡"}, EmotionalIntensity: 0.7, CreatedAt: time.Now()})
	mr := NewRetriever(fs, nil)
	mr.Episodes = es

	res := mr.Retrieve("想喝咖啡", RelevanceHint{}, 500, 0, 30, nil, "s1", false)
	if res.EpisodesUsed == 0 {
		t.Error("情节检索应命中咖啡情节")
	}
	if !strings.Contains(res.TierBBlock, "咖啡") {
		t.Errorf("TierBBlock 应含情节摘要, got %q", res.TierBBlock)
	}
}

func TestRetrieve_AssociationDiffusion(t *testing.T) {
	fs := NewFactStore()
	f1 := fs.Add(MemoryFact{Domain: "SOCIAL", Subcategory: "FRIENDS", Subject: "用户", Summary: "和阿珍常去爬山", Weight: 2, Confidence: 0.9, SelfRelevance: 0.8, Triggers: []string{"爬山"}})
	f2 := fs.Add(MemoryFact{Domain: "SOCIAL", Subcategory: "FRIENDS", Subject: "用户", Summary: "阿珍喜欢摄影", Weight: 2, Confidence: 0.9, SelfRelevance: 0.8})
	index := NewAssociationIndex()
	index.Add(Association{FactIDA: f1.ID, FactIDB: f2.ID, AssociationType: "thematic", Strength: 0.6})
	mr := NewRetriever(fs, nil)
	mr.AssocIndex = index

	res := mr.Retrieve("周末去爬山", RelevanceHint{}, 500, 0, 30, nil, "s1", false)
	if res.AssociationActivations == 0 {
		t.Error("关联扩散应激活关联边")
	}
	if len(res.ActivatedAssocIDs) == 0 {
		t.Error("应记录激活的关联 ID")
	}
}

func TestComputeMemoryEchoFacts_Empty(t *testing.T) {
	echo := ComputeMemoryEchoFacts(nil, 30)
	if echo.Aff != 0 || echo.Sec != 0 || echo.Aro != 0 || echo.Dom != 0 {
		t.Errorf("空输入应返回零值, got %+v", echo)
	}
}

func TestComputeMemoryEchoFacts_WithEmotionalContext(t *testing.T) {
	now := time.Now()
	ranked := []sfPair{
		{f: &Fact{MemoryFact: MemoryFact{
			Weight: 2, SelfRelevance: 0.8, Confidence: 0.9,
			EmotionalContext: &EmotionalContext{Valence: 0.6, Intensity: 0.7, Trust: 70},
			CreatedAt:        now,
		}}, s: 3},
	}
	echo := ComputeMemoryEchoFacts(ranked, 30)
	if echo.Aff < -MemoryEchoCap || echo.Aff > MemoryEchoCap {
		t.Errorf("Aff 应钳制在 ±%v, got %v", MemoryEchoCap, echo.Aff)
	}
	if echo.Sec < -MemoryEchoCap || echo.Sec > MemoryEchoCap {
		t.Errorf("Sec 应钳制在 ±%v, got %v", MemoryEchoCap, echo.Sec)
	}
	if echo.Aff <= 0 {
		t.Errorf("正 valence 记忆回声 Aff 应为正, got %v", echo.Aff)
	}
}
