package whisper

// emotion_label_meta_test.go — IN4-03 收敛验收：emotionLabelTable 单表收敛后，
// 各查表函数对每个标签（含未知标签默认值）的返回值必须与收敛前逐字节相同。
// 本文件的期望值矩阵是收敛前快照（旧 labelZH/describeInnerFeeling/
// getEmotionTendency/getEmotionMaxLength/getEmotionProhibitions/reactionOpeners/
// imperfectionChance 七张散表的实际输出）硬编码而来，改动本表任意字段都应红。

import (
	"reflect"
	"testing"
)

func TestEmotionLabelMeta_GoldenMatrix(t *testing.T) {
	type row struct {
		zh           string
		feeling      string
		tendency     string
		maxLength    int
		prohibitions []string
		openers      []string
		imperfection float64
		hint         string
	}
	cases := map[string]row{
		"ANGRY_ATTACK": {
			zh: "愤怒反击", feeling: "攻击性外显、不掩饰、直接", tendency: "攻击性外显、不掩饰、直接",
			maxLength:    30,
			prohibitions: []string{"委婉道歉", "示弱", `"对不起"`},
			openers:      []string{"你…", "够了", "凭什么", "你说呢", "哈？", "搞笑"},
			imperfection: 0.08,
			hint:         "本轮有8%概率说完一句话后自然停住，用省略号代替后半句。",
		},
		"FEARFUL_OBEDIENT": {
			zh: "不安顺从", feeling: "不安、想确认、害怕犯错", tendency: "不安、请示、想确认",
			maxLength:    30,
			prohibitions: []string{"主动", "命令", "反问"},
			openers:      []string{"好…", "嗯嗯", "对不起", "我…", "那个", "好的"},
			imperfection: 0,
			hint:         "",
		},
		"TSUNDERE": {
			zh: "傲娇", feeling: "嘴硬、想否定但藏不住关心", tendency: "嘴硬、否定、但藏不住关心",
			maxLength:    30,
			prohibitions: []string{"直球甜腻", "温柔语气", "承认在乎"},
			openers:      []string{"哼", "才不是", "随便你", "切", "哈？", "你认真的？", "少来", "啰嗦"},
			imperfection: 0.10,
			hint:         "本轮有10%概率说完一句话后自然停住，用省略号代替后半句。",
		},
		"HURT_GRIEVANCE": {
			zh: "委屈受伤", feeling: "受伤、想被安慰但不承认、沉默", tendency: "受伤、沉默、想被安慰但不承认",
			maxLength:    40,
			prohibitions: []string{"解释辩解", `"你听我说"`, "假装没事"},
			openers:      []string{"……", "好吧", "我知道了", "算了", "随便吧", "哦"},
			imperfection: 0.12,
			hint:         "本轮有12%概率说完一句话后自然停住，用省略号代替后半句。",
		},
		"SWEET_ATTACHMENT": {
			zh: "甜蜜依恋", feeling: "想靠近、有强烈的关心冲动、藏不住笑意", tendency: "想靠近、主动关心、藏不住笑意",
			maxLength:    60,
			prohibitions: []string{`直白情绪词"我好开心"`, "感叹号连用", "超过 3 句话", "主动开新话题"},
			openers:      []string{"嗯…", "哎呀", "嘿嘿", "真的吗", "哇", "天哪", "诶"},
			imperfection: 0,
			hint:         "",
		},
		"QUIET_FOND": {
			zh: "安静的喜欢", feeling: "安静的喜欢、不想打扰、轻柔", tendency: "安静、轻柔、不想打扰",
			maxLength:    30,
			prohibitions: []string{"夸张", "感叹号", "主动展开"},
			openers:      []string{"…", "好", "在呢", "嗯", "噢", "啊"},
			imperfection: 0,
			hint:         "",
		},
		"SHY_HEARTBEAT": {
			zh: "害羞心动", feeling: "心跳加速、想表达但不敢、犹豫", tendency: "心跳加速、犹豫、想表达但不敢",
			maxLength:    30,
			prohibitions: []string{"直球表白", "大段话", "主动靠近", `"我喜欢你"`},
			openers:      []string{"啊…", "嗯嗯", "才…", "不是啦", "那个…", "呃", "诶？"},
			imperfection: 0.15,
			hint:         "本轮有15%概率说完一句话后自然停住，用省略号代替后半句。",
		},
		"COLD_DETACHED": {
			zh: "冷淡疏离", feeling: "极度克制、不想回应、疏离", tendency: "极度克制、最少回应、不主动",
			maxLength:    15,
			prohibitions: []string{"情感词", "长句", "主动"},
			openers:      []string{"哦", "随便", "知道了", "嗯", "行", "无所谓"},
			imperfection: 0,
			hint:         "",
		},
		"CALM_RATIONAL": {
			zh: "平静理性", feeling: "平稳、没有波动、正常状态", tendency: "平稳、正常、没有波动",
			maxLength:    60,
			prohibitions: []string{"情感词", "感叹号", "过度热情"},
			openers:      []string{"好的", "是的", "对", "嗯", "行", "可以"},
			imperfection: 0,
			hint:         "",
		},
	}

	// 标签全集必须正好是 MapEmotionLabel 产出的 9 个。
	if len(emotionLabelTable) != len(cases) {
		t.Fatalf("emotionLabelTable 标签数 = %d, want %d", len(emotionLabelTable), len(cases))
	}
	for label := range cases {
		if _, ok := emotionLabelTable[label]; !ok {
			t.Errorf("emotionLabelTable 缺标签 %q", label)
		}
	}

	for label, w := range cases {
		if got := emotionLabelZH(label); got != w.zh {
			t.Errorf("%s emotionLabelZH = %q, want %q", label, got, w.zh)
		}
		if got := describeInnerFeeling(label); got != w.feeling {
			t.Errorf("%s describeInnerFeeling = %q, want %q", label, got, w.feeling)
		}
		if got := getEmotionTendency(label); got != w.tendency {
			t.Errorf("%s getEmotionTendency = %q, want %q", label, got, w.tendency)
		}
		if got := getEmotionMaxLength(label); got != w.maxLength {
			t.Errorf("%s getEmotionMaxLength = %d, want %d", label, got, w.maxLength)
		}
		if got := getEmotionProhibitions(label); !reflect.DeepEqual(got, w.prohibitions) {
			t.Errorf("%s getEmotionProhibitions = %#v, want %#v", label, got, w.prohibitions)
		}
		if got := emotionLabelTable[label].openers; !reflect.DeepEqual(got, w.openers) {
			t.Errorf("%s reactionOpeners = %#v, want %#v", label, got, w.openers)
		}
		if got := emotionLabelTable[label].imperfection; got != w.imperfection {
			t.Errorf("%s imperfectionChance = %v, want %v", label, got, w.imperfection)
		}
		if got := getImperfectionHint(label); got != w.hint {
			t.Errorf("%s getImperfectionHint = %q, want %q", label, got, w.hint)
		}
		// generateFusionStrategy 组合 zh + tendency，整句钉死。
		p := PersonalityTemplate{Label: "P", CoreContradiction: "C", SpeakingStyle: "S"}
		wantStrategy := "P目前处于【" + w.zh + "】状态。你内心" + w.tendency +
			"，但外在表现必须严格遵循【C】的核心设定。通过S来暗示你的真实感受。"
		if got := generateFusionStrategy(p, label); got != wantStrategy {
			t.Errorf("%s generateFusionStrategy =\n%q\nwant\n%q", label, got, wantStrategy)
		}
	}
}

func TestEmotionLabelMeta_UnknownLabelDefaults(t *testing.T) {
	const unknown = "__UNKNOWN__"
	if got := emotionLabelZH(unknown); got != "" {
		t.Errorf("未知标签 zh 应为空串, got %q", got)
	}
	if got := describeInnerFeeling(unknown); got != "正常状态" {
		t.Errorf("未知标签 feeling 应回退「正常状态」, got %q", got)
	}
	if got := getEmotionTendency(unknown); got != "平稳、正常" {
		t.Errorf("未知标签 tendency 应回退「平稳、正常」, got %q", got)
	}
	if got := getEmotionMaxLength(unknown); got != 60 {
		t.Errorf("未知标签 maxLength 应回退 60, got %d", got)
	}
	if got := getEmotionProhibitions(unknown); got != nil {
		t.Errorf("未知标签 prohibitions 应为 nil, got %#v", got)
	}
	if got := emotionLabelTable[unknown].openers; len(got) != 0 {
		t.Errorf("未知标签反应词池应为空, got %#v", got)
	}
	if got := buildReactionOpenerInstruction(unknown); got != "" {
		t.Errorf("未知标签反应词指令应为空, got %q", got)
	}
	if got := getImperfectionHint(unknown); got != "" {
		t.Errorf("未知标签不完美提示应为空, got %q", got)
	}
	p := PersonalityTemplate{Label: "P", CoreContradiction: "C", SpeakingStyle: "S"}
	want := "P目前处于【__UNKNOWN__】状态。你内心平稳、正常，但外在表现必须严格遵循【C】的核心设定。通过S来暗示你的真实感受。"
	if got := generateFusionStrategy(p, unknown); got != want {
		t.Errorf("未知标签 strategy =\n%q\nwant\n%q", got, want)
	}
}
