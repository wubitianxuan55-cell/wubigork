package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/fileutil"
	"github.com/gaea/gaea/internal/project"
)

// ── 设定事实词典（chinese-novelist-skill「设定与事实词典」机制重推导蒸馏）──
//
// 双时间线账本：每条设定事实分「读者已知」与「完整真相」两栏。章节生成时
// 作为口径约束注入（Rank 0 创作约束级，与伏笔同梯队最后牺牲）：未到揭示章
// 的完整真相不得提前向读者揭示——角色可按完整真相行事，但读者视角的口径
// 以「读者已知」为上限。登记制纪律：先登记后动笔，到期由作者标记已揭示。
// 存储：<项目根>/facts.json（整表写回，作者审批制——Save 是唯一写入口）。

// novelFact 单条设定事实（双时间线）。
type novelFact struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Category      string `json:"category"` // 设定/规则/秘密/身世/金手指…（自由短标签）
	FirstChapter  int    `json:"first_chapter"`
	ReaderKnown   string `json:"reader_known"`   // 读者当前视角的口径上限
	FullTruth     string `json:"full_truth"`     // 作者真相；空=无隐藏层（口径即全部）
	RevealChapter int    `json:"reveal_chapter"` // 计划揭示章；0=未定
	Revealed      bool   `json:"revealed"`       // 作者标记：已向读者揭示
}

// novelFactsFile facts.json 结构。
type novelFactsFile struct {
	Version int         `json:"version"`
	Facts   []novelFact `json:"facts"`
}

const (
	factsStoreFile = "facts.json"
	// ctxFactsBudget 设定事实区段预算（rune，含标题——与伏笔区段同信封口径）。
	ctxFactsBudget = 800
	// ctxFactsMaxItems 注入的最多条数（宁少勿截半）。
	ctxFactsMaxItems = 10
)

// loadNovelFacts 读词典；缺档/损坏返回空表不报错（词典是增强面，不阻断写作）。
func loadNovelFacts(pm *project.Manager) *novelFactsFile {
	f := &novelFactsFile{Version: 1, Facts: []novelFact{}}
	data, err := os.ReadFile(filepath.Join(pm.Dir, factsStoreFile))
	if err != nil {
		return f
	}
	if err := json.Unmarshal(data, f); err != nil {
		return &novelFactsFile{Version: 1, Facts: []novelFact{}}
	}
	if f.Facts == nil {
		f.Facts = []novelFact{}
	}
	return f
}

// NovelFactsList 读设定事实词典（打开面板即拉；零 LLM）。
func (a *writingState) NovelFactsList() (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	f := loadNovelFacts(pm)
	return map[string]interface{}{"facts": f.Facts}, nil
}

// NovelFactsSave 整表写回（作者审批制）。校验：名称与读者已知必填；缺 ID
// 自动补（fact_<unixnano>）；揭示章为负拒收。返回落库条数。
func (a *writingState) NovelFactsSave(factsJSON string) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	var facts []novelFact
	if err := json.Unmarshal([]byte(factsJSON), &facts); err != nil {
		return nil, fmt.Errorf("解析事实 JSON 失败: %w", err)
	}
	now := time.Now().UnixNano()
	for i := range facts {
		f := &facts[i]
		f.Name = strings.TrimSpace(f.Name)
		f.ReaderKnown = strings.TrimSpace(f.ReaderKnown)
		f.FullTruth = strings.TrimSpace(f.FullTruth)
		f.Category = strings.TrimSpace(f.Category)
		if f.Name == "" {
			return nil, fmt.Errorf("第 %d 条事实缺少名称", i+1)
		}
		if f.ReaderKnown == "" {
			return nil, fmt.Errorf("事实「%s」缺少读者已知口径", f.Name)
		}
		if f.RevealChapter < 0 {
			return nil, fmt.Errorf("事实「%s」揭示章不合法: %d", f.Name, f.RevealChapter)
		}
		if f.ID == "" {
			f.ID = fmt.Sprintf("fact_%d", now+int64(i))
		}
	}
	file := &novelFactsFile{Version: 1, Facts: facts}
	if err := fileutil.AtomicWrite(filepath.Join(pm.Dir, factsStoreFile), dissectJSON(file), 0o644); err != nil {
		return nil, fmt.Errorf("保存事实词典失败: %w", err)
	}
	return map[string]interface{}{"saved": len(facts)}, nil
}

// factOpenToReader 事实在当前章是否已可向读者揭示：作者标记 or 已到揭示章。
func factOpenToReader(f novelFact, currentChapter int) bool {
	return f.Revealed || (f.RevealChapter > 0 && currentChapter >= f.RevealChapter)
}

// buildFactsSection 组装「设定事实（口径约束）」区段：未揭示条目带「不得提前
// 向读者揭示」纪律；已揭示条目按完整真相口径写。空词典返回 ""（不占额度）。
func buildFactsSection(pm *project.Manager, currentChapter int) string {
	if pm == nil {
		return ""
	}
	f := loadNovelFacts(pm)
	if len(f.Facts) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## 设定事实（口径约束）\n")
	b.WriteString("「读者已知」是当前读者视角的口径上限；未到揭示章的完整真相不得提前向读者揭示或明示过半——角色可按完整真相行事，但读者看得见的部分不得超过读者已知口径：\n")
	used := runeLen(b.String())
	items := 0
	for _, fact := range f.Facts {
		if items >= ctxFactsMaxItems {
			break
		}
		line := fmt.Sprintf("- %s", fact.Name)
		if fact.Category != "" {
			line += fmt.Sprintf("（%s", fact.Category)
			if fact.FirstChapter > 0 {
				line += fmt.Sprintf("·首现第%d章", fact.FirstChapter)
			}
			line += "）"
		} else if fact.FirstChapter > 0 {
			line += fmt.Sprintf("（首现第%d章）", fact.FirstChapter)
		}
		line += fmt.Sprintf("：读者已知：%s", fact.ReaderKnown)
		if fact.FullTruth != "" && fact.FullTruth != fact.ReaderKnown {
			if factOpenToReader(fact, currentChapter) {
				line += fmt.Sprintf("｜完整真相（已可揭示）：%s", fact.FullTruth)
			} else if fact.RevealChapter > 0 {
				line += fmt.Sprintf("｜完整真相（第%d章揭示，本章不得提前揭示）：%s", fact.RevealChapter, fact.FullTruth)
			} else {
				line += fmt.Sprintf("｜完整真相（未定揭示章，不得提前揭示）：%s", fact.FullTruth)
			}
		}
		l := runeLen(line)
		if used+l > ctxFactsBudget {
			break // 预算耗尽：整条不入（宁少勿截半——截半的事实最危险）
		}
		b.WriteString(line + "\n")
		used += l
		items++
	}
	return strings.TrimRight(b.String(), "\n")
}
