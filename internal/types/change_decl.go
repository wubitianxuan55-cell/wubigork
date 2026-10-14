package types

import "strings"

// ── 章级变更声明（阶段八 8.5）────────────────────────────────
//
// 生成章时模型在正文末尾按 ---CHAPTER_CHANGES--- 附加的结构化声明：本章让谁的
// 状态变了、埋/收了什么伏笔、时间线推进到哪。经确定性门禁（novelgate 五类校验）
// 过闸后随章节摘要落盘，作为定稿回写（8.4）草案的预填充——生成时顺手声明零额
// 外 LLM，定稿时零成本确认。声明缺失=软降级（不阻断正文落盘）；校验不过=拒收
// （不进落库链），问题清单人可读，作者可重新生成或手动修正。

// ChangeDeclState 单条角色状态变更声明。Name 须在角色卡（含别名，门禁校验）。
type ChangeDeclState struct {
	Name string `json:"name"`
	Old  string `json:"old,omitempty"`
	New  string `json:"new"` // 本章后的状态（一句话）
	// Kind 归类：survival=存活（死亡/重伤/退场，走存活水位）；空=心理/处境
	// （走心理水位）。与 characterstate.ApplyChapterDiff 的两级水位同一语义。
	Kind string `json:"kind,omitempty"`
}

// ChangeDeclForeshadow 单条伏笔变更声明。
type ChangeDeclForeshadow struct {
	// ID 引用既有登记表条目（stable_id）：resolved 必填（回收必须挂回既有）；
	// planted 空=新增（由同步器生成 stable_id）。
	ID          string `json:"id,omitempty"`
	Action      string `json:"action"` // planted | hinted | resolved
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

// ChapterChangeDeclaration 章级变更声明（---CHAPTER_CHANGES--- JSON）。
type ChapterChangeDeclaration struct {
	CharacterStates []ChangeDeclState      `json:"character_states"`
	Foreshadows     []ChangeDeclForeshadow `json:"foreshadows,omitempty"`
	Timeline        string                 `json:"timeline,omitempty"` // 时间线推进（一句话；缺失=S3 提示不拒收）
}

// stripExcludeHits 把 scan 中所有角色排除词的出现处替换为 NUL 占位符——之后的
// 名字/别名匹配不再命中这些位置（如排除「王爷」后，正文里的「王爷」不再让
// 「王」命中）。纯函数、确定性。（8.1 实体卡触发注入同款语义，本函数自 app 层
// 上收单源：app 注入装配与 novelgate 门禁两消费方。）
func StripExcludeHits(scan string, chars []Character) string {
	for _, ch := range chars {
		for _, e := range ch.ExcludeHits {
			if e = strings.TrimSpace(e); e != "" {
				scan = strings.ReplaceAll(scan, e, "\x00")
			}
		}
	}
	return scan
}

// CharacterDetected 名字或任一别名是否命中（已剔除排除词的）扫描文本。
// 子串包含即命中（中文无词边界）；空名/空别名跳过。（单源说明同上。）
func CharacterDetected(cleanScan string, ch Character) bool {
	if ch.Name != "" && strings.Contains(cleanScan, ch.Name) {
		return true
	}
	for _, a := range ch.Aliases {
		if a = strings.TrimSpace(a); a != "" && strings.Contains(cleanScan, a) {
			return true
		}
	}
	return false
}

// FindCharacterByName 按名或别名查角色（门禁「未知实体」与定稿草案对齐共用）。
// 同名取第一个（角色卡口径：同名角色本就依赖作者显式消歧）。
func FindCharacterByName(chars []Character, name string) *Character {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	for i := range chars {
		c := &chars[i]
		if c.Name == name {
			return c
		}
		for _, a := range c.Aliases {
			if strings.TrimSpace(a) == name {
				return c
			}
		}
	}
	return nil
}
